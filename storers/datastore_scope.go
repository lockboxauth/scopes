package storers

import (
	"impractical.co/auth/scopes"
)

type datastoreScope struct {
	ID               string
	UserPolicy       string
	UserExceptions   []string
	ClientPolicy     string
	ClientExceptions []string
	IsDefault        bool
}

func fromDatastore(s datastoreScope) scopes.Scope {
	return scopes.Scope(s)
}

func toDatastore(s scopes.Scope) datastoreScope {
	return datastoreScope(s)
}
