package store

import (
	"database/sql"
	"fmt"

	"github.com/redhatinsights/mbop/internal/config"
)

// GetStore is a function that will return the currently configured store. this
// allows it to be overridden for testing or alternative implementations
var GetStore func() Store

// persistent ref to an in-memory store if present
var mem Store

func SetupStore() error {
	switch config.Get().StoreBackend {
	case "postgres":
		pgStore, err := setupPostgresStore()
		if err != nil {
			return err
		}

		GetStore = func() Store { return pgStore }
	case "memory":
		mem = &inMemoryStore{}
		GetStore = func() Store { return mem }
	}

	return nil
}

func setupPostgresStore() (*postgresStore, error) {
	c := config.Get()

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=prefer",
		c.DatabaseUser, c.DatabasePassword, c.DatabaseHost, c.DatabasePort, c.DatabaseName)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	err = migrateDatabase()
	if err != nil {
		return nil, err
	}

	return &postgresStore{db: db}, nil
}
