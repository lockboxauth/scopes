package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"darlinggo.co/pan"
	"github.com/lib/pq"
	"impractical.co/pqarrays"
	"yall.in"

	"lockbox.dev/scopes"
)

//go:generate go-bindata -pkg migrations -o migrations/generated.go sql/

const (
	// TestConnStringEnvVar is the environment variable to use when
	// specifying a connection string for the database to run tests
	// against. Tests will run in their own isolated databases, not in the
	// default database the connection string is for.
	TestConnStringEnvVar = "PG_TEST_DB"
)

// Storer is an implementation of the Storer interface
// that stores data in a PostgreSQL database.
type Storer struct {
	db *sql.DB
}

// NewStorer returns a Storer instance that is backed by the specified
// *sql.DB. The returned Storer instance is ready to be used as a Storer.
func NewStorer(_ context.Context, conn *sql.DB) *Storer {
	return &Storer{db: conn}
}

func createSQL(_ context.Context, scope Scope) *pan.Query {
	return pan.Insert(scope)
}

// Create inserts the passed Scope into the database,
// returning an ErrScopeAlreadyExists error if a Scope
// with the same ID already exists in the database.
func (s *Storer) Create(ctx context.Context, scope scopes.Scope) error {
	query := createSQL(ctx, toPostgres(scope))
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return fmt.Errorf("error generating insert SQL: %w", err)
	}
	_, err = s.db.Exec(queryStr, query.Args()...)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Constraint == "scopes_pkey" {
		return scopes.ErrScopeAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("error inserting scope: %w", err)
	}
	return nil
}

func getMultiSQL(_ context.Context, ids []string) *pan.Query {
	var scope Scope
	query := pan.New("SELECT " + pan.Columns(scope).String() + " FROM " + pan.Table(scope))
	query.Where()
	intIDs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		intIDs = append(intIDs, id)
	}
	query.In(scope, "ID", intIDs...)
	return query.Flush(" ")
}

// GetMulti retrieves the Scopes specified by the passed IDs
// from the database, returning an empty map if no matching
// Scopes are found. If a Scope is not found, no error will
// be returned, it will just be omitted from the map.
func (s *Storer) GetMulti(ctx context.Context, ids []string) (map[string]scopes.Scope, error) {
	query := getMultiSQL(ctx, ids)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL: %w", err)
	}
	rows, err := s.db.Query(queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return nil, fmt.Errorf("error querying scopes: %w", err)
	}
	defer closeRows(ctx, rows)
	results := map[string]scopes.Scope{}
	for rows.Next() {
		var scope Scope
		err = pan.Unmarshal(rows, &scope)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling scope: %w", err)
		}
		results[scope.ID] = fromPostgres(scope)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error querying scopes: %w", err)
	}
	return results, nil
}

func updateSQL(_ context.Context, id string, change scopes.Change) *pan.Query {
	var scope Scope
	query := pan.New("UPDATE " + pan.Table(scope) + " SET ")
	if change.UserPolicy != nil {
		query.Comparison(scope, "UserPolicy", "=", *change.UserPolicy)
	}
	if change.UserExceptions != nil {
		query.Comparison(scope, "UserExceptions", "=", pqarrays.StringArray(*change.UserExceptions))
	}
	if change.ClientPolicy != nil {
		query.Comparison(scope, "ClientPolicy", "=", *change.ClientPolicy)
	}
	if change.ClientExceptions != nil {
		query.Comparison(scope, "ClientExceptions", "=", pqarrays.StringArray(*change.ClientExceptions))
	}
	if change.IsDefault != nil {
		query.Comparison(scope, "IsDefault", "=", *change.IsDefault)
	}
	query.Flush(", ")
	query.Where()
	query.Comparison(scope, "ID", "=", id)
	return query.Flush(" ")
}

// Update applies the passed Change to the Scope that matches
// the specified ID in the Memstore, if any Scope matches the
// specified ID in the Memstore.
func (s *Storer) Update(ctx context.Context, id string, change scopes.Change) error {
	if change.IsEmpty() {
		return nil
	}
	query := updateSQL(ctx, id, change)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return fmt.Errorf("error generating update SQL: %w", err)
	}
	_, err = s.db.Exec(queryStr, query.Args()...)
	if err != nil {
		return fmt.Errorf("error updating scope: %w", err)
	}
	return nil
}

func deleteSQL(_ context.Context, id string) *pan.Query {
	var scope Scope
	q := pan.New("DELETE FROM " + pan.Table(scope))
	q.Where()
	q.Comparison(scope, "ID", "=", id)
	return q.Flush(" ")
}

// Delete removes the Scope that matches the specified ID from
// the Memstore, if any Scope matches the specified ID in the
// Memstore.
func (s *Storer) Delete(ctx context.Context, id string) error {
	query := deleteSQL(ctx, id)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return fmt.Errorf("error generating delete SQL: %w", err)
	}
	_, err = s.db.Exec(queryStr, query.Args()...)
	if err != nil {
		return fmt.Errorf("error deleting scope: %w", err)
	}
	return nil
}

func listDefaultSQL(_ context.Context) *pan.Query {
	var scope Scope
	q := pan.New("SELECT " + pan.Columns(scope).String() + " FROM " + pan.Table(scope))
	q.Where()
	q.Comparison(scope, "IsDefault", "=", true)
	q.OrderBy(pan.Column(scope, "ID"))
	return q.Flush(" ")
}

// ListDefault returns all the Scopes with IsDefault set to true.
// sorted lexicographically by their ID.
func (s *Storer) ListDefault(ctx context.Context) ([]scopes.Scope, error) {
	query := listDefaultSQL(ctx)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL: %w", err)
	}
	rows, err := s.db.Query(queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return nil, fmt.Errorf("error querying scopes: %w", err)
	}
	defer closeRows(ctx, rows)
	var results []scopes.Scope
	for rows.Next() {
		var scope Scope
		err = pan.Unmarshal(rows, &scope)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling scope: %w", err)
		}
		results = append(results, fromPostgres(scope))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error querying scopes: %w", err)
	}
	return results, nil
}

func closeRows(ctx context.Context, rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		yall.FromContext(ctx).WithError(err).Error("failed to close rows")
	}
}
