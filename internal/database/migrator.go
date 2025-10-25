package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strconv"

	"github.com/justinndidit/forex/internal/config"
	"github.com/rs/zerolog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql" // Import mysql db driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, logger *zerolog.Logger, cfg *config.Config) error {
	// DSN for golang-migrate: mysql://user:password@tcp(host:port)/dbname
	migrateDSN := fmt.Sprintf("mysql://%s:%s@tcp(%s:%s)/%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		strconv.Itoa(cfg.Database.Port),
		cfg.Database.Name,
	)

	subtree, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("retrieving database migrations subtree: %w", err)
	}

	sourceInstance, err := iofs.New(subtree, ".")
	if err != nil {
		return fmt.Errorf("creating migrate source instance: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceInstance, migrateDSN)
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}

	currentVersion, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("retrieving current migration version: %w", err)
	}
	if dirty {
		return errors.New("database is in a dirty migration state, please fix manually")
	}

	// Run migrations
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running database migrations: %w", err)
	}

	newVersion, _, _ := m.Version()
	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info().Msgf("database schema up to date, version %d", currentVersion)
	} else {
		logger.Info().Msgf("migrated database schema, from %d to %d", currentVersion, newVersion)
	}

	return nil
}
