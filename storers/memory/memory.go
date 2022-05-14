package memory

import (
	"context"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"

	"lockbox.dev/scopes"
)

var (
	schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"scope": {
				Name: "scope",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID", Lowercase: true},
					},
				},
			},
		},
	}
)

// Storer is an in-memory implementation of the Storer
// interface.
type Storer struct {
	db *memdb.MemDB
}

// NewStorer returns a Storer instance that is ready
// to be used as a Storer.
func NewStorer() (*Storer, error) {
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}
	return &Storer{
		db: db,
	}, nil
}

// Create inserts the passed Scope into the Storer,
// returning an ErrScopeAlreadyExists error if a Scope
// with the same ID already exists in the Storer.
func (s *Storer) Create(_ context.Context, scope scopes.Scope) error {
	txn := s.db.Txn(true)
	defer txn.Abort()
	exists, err := txn.First("scope", "id", scope.ID)
	if err != nil {
		return fmt.Errorf("error retrieving scope: %w", err)
	}
	if exists != nil {
		return scopes.ErrScopeAlreadyExists
	}
	err = txn.Insert("scope", &scope)
	if err != nil {
		return fmt.Errorf("error inserting scope: %w", err)
	}
	txn.Commit()
	return nil
}

// GetMulti retrieves the Scopes specified by the passed IDs
// from the Storer, returning an empty map if no matching
// Scopes are found. If a Scope is not found, no error will
// be returned, it will just be omitted from the map.
func (s *Storer) GetMulti(_ context.Context, ids []string) (map[string]scopes.Scope, error) {
	results := map[string]scopes.Scope{}
	for _, id := range ids {
		txn := s.db.Txn(false)
		res, err := txn.First("scope", "id", id)
		if err != nil {
			return results, fmt.Errorf("error retrieving scope %s: %w", id, err)
		}
		if res == nil {
			continue
		}
		scope, ok := res.(*scopes.Scope)
		if !ok || scope == nil {
			return results, fmt.Errorf("unexpected response type for scope %s: %T (%v)", id, res, res) //nolint:goerr113 // not going to be handled, for debug only
		}
		results[id] = *scope
	}
	return results, nil
}

// Update applies the passed Change to the Scope that matches
// the specified ID in the Storer, if any Scope matches the
// specified ID in the Storer.
func (s *Storer) Update(_ context.Context, id string, change scopes.Change) error {
	txn := s.db.Txn(true)
	defer txn.Abort()
	scope, err := txn.First("scope", "id", id)
	if err != nil {
		return fmt.Errorf("error retrieving scope: %w", err)
	}
	if scope == nil {
		return nil
	}
	newScope, ok := scope.(*scopes.Scope)
	if !ok || newScope == nil {
		return fmt.Errorf("unexpected response type %T (%v)", scope, scope) //nolint:goerr113 // not going to be handled, for debug only
	}
	updated := scopes.Apply(change, *newScope)
	err = txn.Insert("scope", &updated)
	if err != nil {
		return fmt.Errorf("error writing scope: %w", err)
	}
	txn.Commit()
	return nil
}

// Delete removes the Scope that matches the specified ID from
// the Storer, if any Scope matches the specified ID in the
// Storer.
func (s *Storer) Delete(_ context.Context, id string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()
	exists, err := txn.First("scope", "id", id)
	if err != nil {
		return fmt.Errorf("error retrieving scope: %w", err)
	}
	if exists == nil {
		return nil
	}
	err = txn.Delete("scope", exists)
	if err != nil {
		return fmt.Errorf("error deleting scope: %w", err)
	}
	txn.Commit()
	return nil
}

// ListDefault returns all the Scopes with IsDefault set to true.
// sorted lexicographically by their ID.
func (s *Storer) ListDefault(_ context.Context) ([]scopes.Scope, error) {
	txn := s.db.Txn(false)
	var results []scopes.Scope
	acctIter, err := txn.Get("scope", "id")
	if err != nil {
		return nil, fmt.Errorf("error listing scopes: %w", err)
	}
	for {
		nextScope := acctIter.Next()
		if nextScope == nil {
			break
		}
		scope, ok := nextScope.(*scopes.Scope)
		if !ok || scope == nil {
			return nil, fmt.Errorf("unexpected response type %T (%v)", nextScope, nextScope) //nolint:goerr113 // not going to be handled, for debug only
		}
		if !scope.IsDefault {
			continue
		}
		results = append(results, *scope)
	}
	scopes.ByID(results)
	return results, nil
}
