package scopes

import (
	"context"
)

// Storer is an interface for storing and retrieving Scopes and the metadata
// surrounding them.
type Storer interface {
	Create(ctx context.Context, scope Scope) error
	GetMulti(ctx context.Context, ids []string) (map[string]Scope, error)
	ListDefault(ctx context.Context) ([]Scope, error)
	Update(ctx context.Context, id string, change Change) error
	Delete(ctx context.Context, id string) error
}
