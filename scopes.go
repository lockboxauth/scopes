package scopes

//go:generate go-bindata -pkg migrations -o migrations/generated.go sql/

import (
	"context"
	"errors"
	"sort"

	"impractical.co/pqarrays"
)

const (
	// PolicyDenyAll defines a string to use to deny all access.
	PolicyDenyAll = "DENY_ALL"
	// PolicyDefaultDeny defines a string to use to deny access by default, with exceptions.
	PolicyDefaultDeny = "DEFAULT_DENY"
	// PolicyAllowAll defines a string to use to allow all access.
	PolicyAllowAll = "ALLOW_ALL"
	// PolicyDefaultAllow defines a string to use to allow access by default, with exceptions.
	PolicyDefaultAllow = "DEFAULT_ALLOW"
)

var (
	// ErrScopeAlreadyExists is returned when attempting to create a Scope that already exists.
	ErrScopeAlreadyExists = errors.New("scope already exists")
)

// Scope defines a scope of access to user data that users can grant.
type Scope struct {
	ID               string               `sql_column:"id"`
	UserPolicy       string               `sql_column:"user_policy"`
	UserExceptions   pqarrays.StringArray `sql_column:"user_exceptions"`
	ClientPolicy     string               `sql_column:"client_policy"`
	ClientExceptions pqarrays.StringArray `sql_column:"client_exceptions"`
	IsDefault        bool                 `sql_column:"is_default"`
}

// GetSQLTableName returns the table name to store data in when using SQL.
func (s Scope) GetSQLTableName() string {
	return "scopes"
}

// IsValidPolicy returns whether a string is a valid policy or not.
func IsValidPolicy(p string) bool {
	if p == PolicyDenyAll ||
		p == PolicyDefaultDeny ||
		p == PolicyDefaultAllow ||
		p == PolicyAllowAll {
		return true
	}
	return false
}

// Change represents a change to a Scope.
type Change struct {
	UserPolicy   *string
	ClientPolicy *string
	// TODO(paddy): need to be able to manage exceptions
	IsDefault *bool
}

// IsEmpty returns true if the Change should be considered empty.
func (c Change) IsEmpty() bool {
	if c.UserPolicy != nil {
		return false
	}
	if c.ClientPolicy != nil {
		return false
	}
	if c.IsDefault != nil {
		return false
	}
	return true
}

// Apply returns a Scope that is a copy of `scope` with Change applied.
func Apply(change Change, scope Scope) Scope {
	if change.IsEmpty() {
		return scope
	}
	res := scope
	if change.UserPolicy != nil {
		res.UserPolicy = *change.UserPolicy
	}
	if change.ClientPolicy != nil {
		res.ClientPolicy = *change.ClientPolicy
	}
	if change.IsDefault != nil {
		res.IsDefault = *change.IsDefault
	}
	return res
}

// Storer is an interface for storing and retrieving Scopes and the metadata
// surrounding them.
type Storer interface {
	Create(ctx context.Context, scope Scope) error
	GetMulti(ctx context.Context, ids []string) (map[string]Scope, error)
	ListDefault(ctx context.Context) ([]Scope, error)
	Update(ctx context.Context, id string, change Change) error
	Delete(ctx context.Context, id string) error
}

// Dependencies holds the common dependencies that will be used throughout the
// package.
type Dependencies struct {
	Storer Storer
}

// ByID sorts the passed Scopes in place lexicographically by their IDs.
func ByID(scopes []Scope) {
	sort.Slice(scopes, func(i, j int) bool {
		return scopes[i].ID < scopes[j].ID
	})
}
