package apiv1

import (
	"impractical.co/auth/scopes"
)

// Scope is the API representation of an Scope.
// it dictates what the JSON representation of Scopes
// will be.
type Scope struct {
	ID               string   `json:"id"`
	UserPolicy       string   `json:"userPolicy"`
	UserExceptions   []string `json:"userExceptions"`
	ClientPolicy     string   `json:"clientPolicy"`
	ClientExceptions []string `json:"clientExceptions"`
	IsDefault        bool     `json:"isDefault"`
}

// Change is the API representation of a Change.
// It dictates what the JSON representation of Changes
// will be.
type Change struct {
	UserPolicy       *string   `json:"userPolicy"`
	UserExceptions   *[]string `json:"userExceptions"`
	ClientPolicy     *string   `json:"clientPolicy"`
	ClientExceptions *[]string `json:"clientExceptions"`
	IsDefault        *bool     `json:"isDefault"`
}

func coreScope(scope Scope) scopes.Scope {
	return scopes.Scope{
		ID:               scope.ID,
		UserPolicy:       scope.UserPolicy,
		UserExceptions:   scope.UserExceptions,
		ClientPolicy:     scope.ClientPolicy,
		ClientExceptions: scope.ClientExceptions,
		IsDefault:        scope.IsDefault,
	}
}

func coreScopes(scops []Scope) []scopes.Scope {
	res := make([]scopes.Scope, 0, len(scops))
	for _, scop := range scops {
		res = append(res, coreScope(scop))
	}
	return res
}

func apiScope(scope scopes.Scope) Scope {
	return Scope{
		ID:               scope.ID,
		UserPolicy:       scope.UserPolicy,
		UserExceptions:   scope.UserExceptions,
		ClientPolicy:     scope.ClientPolicy,
		ClientExceptions: scope.ClientExceptions,
		IsDefault:        scope.IsDefault,
	}
}

func apiScopes(scops []scopes.Scope) []Scope {
	res := make([]Scope, 0, len(scops))
	for _, scop := range scops {
		res = append(res, apiScope(scop))
	}
	return res
}

func coreChange(change Change) scopes.Change {
	return scopes.Change{
		UserPolicy:       change.UserPolicy,
		UserExceptions:   change.UserExceptions,
		ClientPolicy:     change.ClientPolicy,
		ClientExceptions: change.ClientExceptions,
		IsDefault:        change.IsDefault,
	}
}

func apiChange(change scopes.Change) Change {
	return Change{
		UserPolicy:       change.UserPolicy,
		UserExceptions:   change.UserExceptions,
		ClientPolicy:     change.ClientPolicy,
		ClientExceptions: change.ClientExceptions,
		IsDefault:        change.IsDefault,
	}
}
