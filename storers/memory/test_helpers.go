package memory

import (
	"context"

	"lockbox.dev/scopes"
)

// Factory is a generator of Storers for testing purposes.
type Factory struct{}

// NewStorer creates a new, isolated, in-memory Storer for tests.
func (Factory) NewStorer(_ context.Context) (scopes.Storer, error) { //nolint:ireturn // the interface we're filling returns an interface here
	return NewStorer()
}

// TeardownStorers does nothing and is only included to fill an interface.
func (Factory) TeardownStorers() error {
	return nil
}
