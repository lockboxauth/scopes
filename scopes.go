package scopes

// TODO: refactor storers to match pattern

//go:generate go-bindata -pkg migrations -o migrations/generated.go sql/

import (
	"context"
	"errors"
	"sort"

	yall "yall.in"
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
	ID               string
	UserPolicy       string
	UserExceptions   []string
	ClientPolicy     string
	ClientExceptions []string
	IsDefault        bool
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
	UserPolicy       *string
	UserExceptions   *[]string
	ClientPolicy     *string
	ClientExceptions *[]string
	IsDefault        *bool
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
	if c.UserExceptions != nil {
		return false
	}
	if c.ClientExceptions != nil {
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
	if change.UserExceptions != nil {
		res.UserExceptions = append([]string{}, *change.UserExceptions...)
	}
	if change.ClientPolicy != nil {
		res.ClientPolicy = *change.ClientPolicy
	}
	if change.ClientExceptions != nil {
		res.ClientExceptions = append([]string{}, *change.ClientExceptions...)
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

// FilterByClientID returns which of the Scopes of `scopes` the client specified
// by `clientID` can use.
func FilterByClientID(ctx context.Context, scopes []Scope, clientID string) []Scope {
	var results []Scope
	for _, scope := range scopes {
		if ClientCanUseScope(ctx, scope, clientID) {
			results = append(results, scope)
		}
	}
	return results
}

// ClientCanUseScope returns true if the client specified by `client` can use `scope`.
func ClientCanUseScope(ctx context.Context, scope Scope, client string) bool {
	switch scope.ClientPolicy {
	case PolicyDenyAll:
		return false
	case PolicyAllowAll:
		return true
	case PolicyDefaultDeny:
		for _, id := range scope.ClientExceptions {
			if id == client {
				return true
			}
		}
		return false
	case PolicyDefaultAllow:
		for _, id := range scope.ClientExceptions {
			if id == client {
				return false
			}
		}
		return true
	default:
		yall.FromContext(ctx).WithField("scope", scope.ID).WithField("client", client).Warn("unknown scope presented, ignoring it")
		return false
	}
}
