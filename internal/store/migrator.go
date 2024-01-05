package store

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/redhatinsights/mbop/internal/config"

	// this is the iofs:// driver for go-migrate.
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations
var migrations embed.FS

func migrateDatabase() error {
	fs, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}

	c := config.Get()
	connStr := fmt.Sprintf("pgx://%s:%s@%s:%d/%s?sslmode=prefer",
		c.DatabaseUser, c.DatabasePassword, c.DatabaseHost, c.DatabasePort, c.DatabaseName)

	m, err := migrate.NewWithSourceInstance("iofs", fs, connStr)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
