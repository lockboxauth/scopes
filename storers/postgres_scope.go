package storers

import (
	"impractical.co/pqarrays"

	"lockbox.dev/scopes"
)

type postgresScope struct {
	ID               string               `sql_column:"id"`
	UserPolicy       string               `sql_column:"user_policy"`
	UserExceptions   pqarrays.StringArray `sql_column:"user_exceptions"`
	ClientPolicy     string               `sql_column:"client_policy"`
	ClientExceptions pqarrays.StringArray `sql_column:"client_exceptions"`
	IsDefault        bool                 `sql_column:"is_default"`
}

func (p postgresScope) GetSQLTableName() string {
	return "scopes"
}

func fromPostgres(s postgresScope) scopes.Scope {
	return scopes.Scope{
		ID:               s.ID,
		UserPolicy:       s.UserPolicy,
		UserExceptions:   []string(s.UserExceptions),
		ClientPolicy:     s.ClientPolicy,
		ClientExceptions: []string(s.ClientExceptions),
		IsDefault:        s.IsDefault,
	}
}

func toPostgres(s scopes.Scope) postgresScope {
	return postgresScope{
		ID:               s.ID,
		UserPolicy:       s.UserPolicy,
		UserExceptions:   pqarrays.StringArray(s.UserExceptions),
		ClientPolicy:     s.ClientPolicy,
		ClientExceptions: pqarrays.StringArray(s.ClientExceptions),
		IsDefault:        s.IsDefault,
	}
}
