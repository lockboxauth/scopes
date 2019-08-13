package postgres

import (
	"context"
	"database/sql"

	"darlinggo.co/pan"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"impractical.co/pqarrays"

	"lockbox.dev/scopes"
)

//go:generate go-bindata -pkg migrations -o migrations/generated.go sql/

const (
	TestConnStringEnvVar = "PG_TEST_DB"
)

// Storer is an implementation of the Storer interface
// that stores data in a PostgreSQL database.
type Storer struct {
	db *sql.DB
}

// NewStorer returns a Storer instance that is backed by the specified
// *sql.DB. The returned Storer instance is ready to be used as a Storer.
func NewStorer(ctx context.Context, conn *sql.DB) *Storer {
	return &Storer{db: conn}
}

func createSQL(ctx context.Context, scope Scope) *pan.Query {
	return pan.Insert(scope)
}

// Create inserts the passed Scope into the database,
// returning an ErrScopeAlreadyExists error if a Scope
// with the same ID already exists in the database.
func (s *Storer) Create(ctx context.Context, scope scopes.Scope) error {
	query := createSQL(ctx, toPostgres(scope))
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return errors.Wrap(err, "error generating insert SQL")
	}
	_, err = s.db.Exec(queryStr, query.Args()...)
	if e, ok := err.(*pq.Error); ok {
		if e.Constraint == "scopes_pkey" {
			return scopes.ErrScopeAlreadyExists
		}
	}
	if err != nil {
		return errors.Wrap(err, "error inserting scope")
	}
	return nil
}

func getMultiSQL(ctx context.Context, ids []string) *pan.Query {
	var scope Scope
	q := pan.New("SELECT " + pan.Columns(scope).String() + " FROM " + pan.Table(scope))
	q.Where()
	intIDs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		intIDs = append(intIDs, id)
	}
	q.In(scope, "ID", intIDs...)
	return q.Flush(" ")
}

// GetMulti retrieves the Scopes specified by the passed IDs
// from the database, returning an empty map if no matching
// Scopes are found. If a Scope is not found, no error will
// be returned, it will just be omitted from the map.
func (s *Storer) GetMulti(ctx context.Context, ids []string) (map[string]scopes.Scope, error) {
	query := getMultiSQL(ctx, ids)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return nil, errors.Wrap(err, "error generating SQL")
	}
	rows, err := s.db.Query(queryStr, query.Args()...)
	if err != nil {
		return nil, errors.Wrap(err, "error querying scopes")
	}
	results := map[string]scopes.Scope{}
	for rows.Next() {
		var scope Scope
		err = pan.Unmarshal(rows, &scope)
		if err != nil {
			return nil, errors.Wrap(err, "error unmarshaling scope")
		}
		results[scope.ID] = fromPostgres(scope)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error querying scopes")
	}
	return results, nil
}

func updateSQL(ctx context.Context, id string, change scopes.Change) *pan.Query {
	var scope Scope
	q := pan.New("UPDATE " + pan.Table(scope) + " SET ")
	if change.UserPolicy != nil {
		q.Comparison(scope, "UserPolicy", "=", *change.UserPolicy)
	}
	if change.UserExceptions != nil {
		q.Comparison(scope, "UserExceptions", "=", pqarrays.StringArray(*change.UserExceptions))
	}
	if change.ClientPolicy != nil {
		q.Comparison(scope, "ClientPolicy", "=", *change.ClientPolicy)
	}
	if change.ClientExceptions != nil {
		q.Comparison(scope, "ClientExceptions", "=", pqarrays.StringArray(*change.ClientExceptions))
	}
	if change.IsDefault != nil {
		q.Comparison(scope, "IsDefault", "=", *change.IsDefault)
	}
	q.Flush(", ")
	q.Where()
	q.Comparison(scope, "ID", "=", id)
	return q.Flush(" ")
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
		return errors.Wrap(err, "error generating update SQL")
	}
	_, err = s.db.Exec(queryStr, query.Args()...)
	if err != nil {
		return errors.Wrap(err, "error updating scope")
	}
	return nil
}

func deleteSQL(ctx context.Context, id string) *pan.Query {
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
		return errors.Wrap(err, "error generating delete SQL")
	}
	_, err = s.db.Exec(queryStr, query.Args()...)
	if err != nil {
		return errors.Wrap(err, "error deleting scope")
	}
	return nil
}

func listDefaultSQL(ctx context.Context) *pan.Query {
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
		return nil, errors.Wrap(err, "error generating SQL")
	}
	rows, err := s.db.Query(queryStr, query.Args()...)
	if err != nil {
		return nil, errors.Wrap(err, "error querying scopes")
	}
	var results []scopes.Scope
	for rows.Next() {
		var scope Scope
		err = pan.Unmarshal(rows, &scope)
		if err != nil {
			return nil, errors.Wrap(err, "error unmarshaling scope")
		}
		results = append(results, fromPostgres(scope))
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error querying scopes")
	}
	return results, nil
}
