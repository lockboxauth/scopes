package storers

import (
	"context"

	"cloud.google.com/go/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"impractical.co/auth/scopes"
)

const datastoreScopeKind = "Scope"

// Datastore is an implementation of the Storer interface
// that stores data in a PostgreSQL database.
type Datastore struct {
	client    *datastore.Client
	namespace string
}

// NewDatastore returns a Datastore instance that is backed by the specified
// datastore client. The returned Datastore instance is ready to be used as a Storer.
func NewDatastore(ctx context.Context, client *datastore.Client) *Datastore {
	return &Datastore{client: client}
}

func (d *Datastore) key(id string) *datastore.Key {
	key := datastore.NameKey(datastoreScopeKind, id, nil)
	if d.namespace != "" {
		key.Namespace = d.namespace
	}
	return key
}

// Create inserts the passed Scope into the database,
// returning an ErrScopeAlreadyExists error if a Scope
// with the same ID already exists in the database.
func (d *Datastore) Create(ctx context.Context, scope scopes.Scope) error {
	s := toDatastore(scope)
	mut := datastore.NewInsert(d.key(s.ID), &s)
	_, err := d.client.Mutate(ctx, mut)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return scopes.ErrScopeAlreadyExists
		}
		return err
	}
	return nil
}

// GetMulti retrieves the Scopes specified by the passed IDs
// from the database, returning an empty map if no matching
// Scopes are found. If a Scope is not found, no error will
// be returned, it will just be omitted from the map.
func (d *Datastore) GetMulti(ctx context.Context, ids []string) (map[string]scopes.Scope, error) {
	keys := make([]*datastore.Key, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, d.key(id))
	}
	// we need to keep track of which keys have results
	// so we can piece them all together at the end
	// we start by assuming they'll all have results
	hasResults := map[string]bool{}
	for _, key := range keys {
		hasResults[key.Name] = true
	}
	scops := make([]datastoreScope, len(keys))
	err := d.client.GetMulti(ctx, keys, scops)
	if err != nil {
		if e, ok := err.(datastore.MultiError); ok {
			// this may not be a not found error, in which case
			// we need to not swallow it
			var hasRealError bool
			for pos, er := range e {
				if er != nil && er != datastore.ErrNoSuchEntity {
					hasRealError = true
				} else if er == datastore.ErrNoSuchEntity {
					// if we have a not found error, the key
					// in that position wasn't found, so unset
					// it
					hasResults[keys[pos].Name] = false
				}
			}
			if hasRealError {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	results := map[string]scopes.Scope{}
	for pos, scope := range scops {
		if !hasResults[keys[pos].Name] {
			continue
		}
		scope.ID = keys[pos].Name
		if scope.ID == "" {
			continue
		}
		results[scope.ID] = fromDatastore(scope)
	}
	return results, nil
}

// Update applies the passed Change to the Scope that matches
// the specified ID in the Memstore, if any Scope matches the
// specified ID in the Memstore.
func (d *Datastore) Update(ctx context.Context, id string, change scopes.Change) error {
	if change.IsEmpty() {
		return nil
	}
	_, err := d.client.RunInTransaction(ctx, func(txn *datastore.Transaction) error {
		var scope datastoreScope
		err := txn.Get(d.key(id), &scope)
		if err == datastore.ErrNoSuchEntity {
			return nil
		} else if err != nil {
			return err
		}
		s := toDatastore(scopes.Apply(change, fromDatastore(scope)))
		_, err = txn.Put(d.key(id), &s)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

// Delete removes the Scope that matches the specified ID from
// the Memstore, if any Scope matches the specified ID in the
// Memstore.
func (d *Datastore) Delete(ctx context.Context, id string) error {
	return d.client.Delete(ctx, d.key(id))
}

// ListDefault returns all the Scopes with IsDefault set to true.
// sorted lexicographically by their ID.
func (d *Datastore) ListDefault(ctx context.Context) ([]scopes.Scope, error) {
	q := datastore.NewQuery(datastoreScopeKind).Filter("IsDefault =", true).KeysOnly()
	if d.namespace != "" {
		q = q.Namespace(d.namespace)
	}
	keys, err := d.client.GetAll(ctx, q, nil)
	if err != nil {
		return nil, err
	}
	scops := make([]datastoreScope, len(keys))
	err = d.client.GetMulti(ctx, keys, scops)
	if err != nil {
		return nil, err
	}
	results := make([]scopes.Scope, 0, len(scops))
	for pos, scope := range scops {
		scope.ID = keys[pos].Name
		results = append(results, fromDatastore(scope))
	}
	return results, nil
}
