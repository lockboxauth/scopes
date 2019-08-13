package memory

import (
	"context"

	"lockbox.dev/scopes"
)

type Factory struct{}

func (f Factory) NewStorer(ctx context.Context) (scopes.Storer, error) {
	return NewStorer()
}

func (f Factory) TeardownStorers() error {
	return nil
}
