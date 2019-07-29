package storers

import (
	"context"

	"lockbox.dev/scopes"
)

func init() {
	storerFactories = append(storerFactories, MemstoreFactory{})
}

type MemstoreFactory struct{}

func (m MemstoreFactory) NewStorer(ctx context.Context) (scopes.Storer, error) {
	return NewMemstore()
}

func (m MemstoreFactory) TeardownStorers() error {
	return nil
}
