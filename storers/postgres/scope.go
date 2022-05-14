package postgres

import (
	"impractical.co/pqarrays"

	"lockbox.dev/scopes"
)

// Scope is a representation of the scopes.Scope type that is suitable to be
// stored in a PostgreSQL database.
type Scope struct {
	ID               string               `sql_column:"id"`
	UserPolicy       string               `sql_column:"user_policy"`
	UserExceptions   pqarrays.StringArray `sql_column:"user_exceptions"`
	ClientPolicy     string               `sql_column:"client_policy"`
	ClientExceptions pqarrays.StringArray `sql_column:"client_exceptions"`
	IsDefault        bool                 `sql_column:"is_default"`
}

// GetSQLTableName returns the name of the SQL table that the data for this
// type will be stored in.
func (Scope) GetSQLTableName() string {
	return "scopes"
}

func fromPostgres(scope Scope) scopes.Scope {
	return scopes.Scope{
		ID:               scope.ID,
		UserPolicy:       scope.UserPolicy,
		UserExceptions:   []string(scope.UserExceptions),
		ClientPolicy:     scope.ClientPolicy,
		ClientExceptions: []string(scope.ClientExceptions),
		IsDefault:        scope.IsDefault,
	}
}

func toPostgres(scope scopes.Scope) Scope {
	return Scope{
		ID:               scope.ID,
		UserPolicy:       scope.UserPolicy,
		UserExceptions:   pqarrays.StringArray(scope.UserExceptions),
		ClientPolicy:     scope.ClientPolicy,
		ClientExceptions: pqarrays.StringArray(scope.ClientExceptions),
		IsDefault:        scope.IsDefault,
	}
}
