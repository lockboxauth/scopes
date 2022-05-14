package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net/url"
	"os"
	"sync"

	uuid "github.com/hashicorp/go-uuid"
	migrate "github.com/rubenv/sql-migrate"

	"lockbox.dev/scopes"
	"lockbox.dev/scopes/storers/postgres/migrations"
)

// Factory is a generator of Storers for testing purposes. It knows how to
// create, track, and clean up PostgreSQL databases that tests can be run
// against.
type Factory struct {
	db        *sql.DB
	databases map[string]*sql.DB
	lock      sync.Mutex
}

// NewFactory returns a Factory that is ready to be used. The passed sql.DB
// will be used as a control plane connection, but each test will have its own
// database created for that test.
func NewFactory(db *sql.DB) *Factory {
	return &Factory{
		db:        db,
		databases: map[string]*sql.DB{},
	}
}

// NewStorer retrieves the connection string from the environment (using
// TestConnStringEnvVar), parses it, and injects a new database name into it.
// The new database name is a random name prefixed with scopes_test_, and it
// will be automatically created in NewStorer. NewStorer also runs migrations,
// and keeps track of these test databases so they can be deleted automatically
// later.
func (f *Factory) NewStorer(ctx context.Context) (scopes.Storer, error) { //nolint:ireturn // the interface we're filling wants an interface returned
	connString, err := url.Parse(os.Getenv(TestConnStringEnvVar))
	if err != nil {
		log.Printf("Error parsing "+TestConnStringEnvVar+" as a URL: %+v\n", err)
		return nil, err
	}
	if connString.Scheme != "postgres" {
		return nil, errors.New(TestConnStringEnvVar + " must begin with postgres://") //nolint:goerr113 // not going to be handled, for logging purposes only
	}

	tableSuffix, err := uuid.GenerateRandomBytes(6) //nolint:gomnd // not magic, just arbitrary
	if err != nil {
		log.Printf("Error generating table suffix: %+v\n", err)
		return nil, err
	}
	table := "accounts_test_" + hex.EncodeToString(tableSuffix)

	_, err = f.db.Exec("CREATE DATABASE " + table + ";")
	if err != nil {
		log.Printf("Error creating database %s: %+v\n", table, err)
		return nil, err
	}

	connString.Path = "/" + table
	newConn, err := sql.Open("postgres", connString.String())
	if err != nil {
		log.Println("Accidentally orphaned", table, "it will need to be cleaned up manually")
		return nil, err
	}

	f.lock.Lock()
	f.databases[table] = newConn
	f.lock.Unlock()

	migs := &migrate.AssetMigrationSource{
		Asset:    migrations.Asset,
		AssetDir: migrations.AssetDir,
		Dir:      "sql",
	}
	_, err = migrate.Exec(newConn, "postgres", migs, migrate.Up)
	if err != nil {
		return nil, err
	}

	storer := NewStorer(ctx, newConn)

	return storer, nil
}

// TeardownStorers automatically deletes all the tracked databases created by
// NewStorer.
func (f *Factory) TeardownStorers() error {
	f.lock.Lock()
	defer f.lock.Unlock()

	for table, conn := range f.databases {
		if err := conn.Close(); err != nil {
			return err
		}
		_, err := f.db.Exec("DROP DATABASE " + table + ";")
		if err != nil {
			return err
		}
	}
	return f.db.Close()
}
