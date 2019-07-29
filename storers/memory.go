package storers

import (
	"context"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/pkg/errors"

	"lockbox.dev/scopes"
)

var (
	schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"scope": &memdb.TableSchema{
				Name: "scope",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID", Lowercase: true},
					},
				},
			},
		},
	}
)

// Memstore is an in-memory implementation of the Storer
// interface.
type Memstore struct {
	db *memdb.MemDB
}

// NewMemstore returns a Memstore instance that is ready
// to be used as a Storer.
func NewMemstore() (*Memstore, error) {
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}
	return &Memstore{
		db: db,
	}, nil
}

// Create inserts the passed Scope into the Memstore,
// returning an ErrScopeAlreadyExists error if a Scope
// with the same ID already exists in the Memstore.
func (m *Memstore) Create(ctx context.Context, scope scopes.Scope) error {
	txn := m.db.Txn(true)
	defer txn.Abort()
	exists, err := txn.First("scope", "id", scope.ID)
	if err != nil {
		return errors.Wrap(err, "error retrieving scope")
	}
	if exists != nil {
		return scopes.ErrScopeAlreadyExists
	}
	err = txn.Insert("scope", &scope)
	if err != nil {
		return errors.Wrap(err, "error inserting scope")
	}
	txn.Commit()
	return nil
}

// GetMulti retrieves the Scopes specified by the passed IDs
// from the Memstore, returning an empty map if no matching
// Scopes are found. If a Scope is not found, no error will
// be returned, it will just be omitted from the map.
func (m *Memstore) GetMulti(ctx context.Context, ids []string) (map[string]scopes.Scope, error) {
	results := map[string]scopes.Scope{}
	for _, id := range ids {
		txn := m.db.Txn(false)
		s, err := txn.First("scope", "id", id)
		if err != nil {
			return results, errors.Wrap(err, "error retrieving scope "+id)
		}
		if s != nil {
			results[id] = *s.(*scopes.Scope)
		}
	}
	return results, nil
}

// Update applies the passed Change to the Scope that matches
// the specified ID in the Memstore, if any Scope matches the
// specified ID in the Memstore.
func (m *Memstore) Update(ctx context.Context, id string, change scopes.Change) error {
	txn := m.db.Txn(true)
	defer txn.Abort()
	scope, err := txn.First("scope", "id", id)
	if err != nil {
		return errors.Wrap(err, "error retrieving scope")
	}
	if scope == nil {
		return nil
	}
	updated := scopes.Apply(change, *scope.(*scopes.Scope))
	err = txn.Insert("scope", &updated)
	if err != nil {
		return errors.Wrap(err, "error writing scope")
	}
	txn.Commit()
	return nil
}

// Delete removes the Scope that matches the specified ID from
// the Memstore, if any Scope matches the specified ID in the
// Memstore.
func (m *Memstore) Delete(ctx context.Context, id string) error {
	txn := m.db.Txn(true)
	defer txn.Abort()
	exists, err := txn.First("scope", "id", id)
	if err != nil {
		return errors.Wrap(err, "error retrieving scope")
	}
	if exists == nil {
		return nil
	}
	err = txn.Delete("scope", exists)
	if err != nil {
		return errors.Wrap(err, "error deleting scope")
	}
	txn.Commit()
	return nil
}

// ListDefault returns all the Scopes with IsDefault set to true.
// sorted lexicographically by their ID.
func (m *Memstore) ListDefault(ctx context.Context) ([]scopes.Scope, error) {
	txn := m.db.Txn(false)
	var results []scopes.Scope
	acctIter, err := txn.Get("scope", "id")
	if err != nil {
		return nil, errors.Wrap(err, "error listing scopes")
	}
	for {
		scope := acctIter.Next()
		if scope == nil {
			break
		}
		if !scope.(*scopes.Scope).IsDefault {
			continue
		}
		results = append(results, *scope.(*scopes.Scope))
	}
	scopes.ByID(results)
	return results, nil
}
