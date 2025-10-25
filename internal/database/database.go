package database

import (
	"context"
	"database/sql" // Use standard database/sql
	"fmt"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql" // Import MySQL driver
	"github.com/justinndidit/forex/internal/config"
	"github.com/rs/zerolog"
)

type Database struct {
	Pool *sql.DB // Changed from pgxpool.Pool
	log  *zerolog.Logger
}

const DatabasePingTimeout = 10

func New(cfg *config.Config, logger *zerolog.Logger) (*Database, error) {
	// MySQL DSN: user:password@tcp(host:port)/dbname?parseTime=true
	// We must add parseTime=true to handle time.Time fields

	// Map PostgreSQL sslmode to MySQL tls
	// This is a basic mapping; 'verify-full' would require more setup
	tlsMode := "false"
	if cfg.Database.SSLMode == "require" || cfg.Database.SSLMode == "verify-full" {
		tlsMode = "true"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&tls=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		strconv.Itoa(cfg.Database.Port),
		cfg.Database.Name,
		tlsMode,
	)

	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	database := &Database{
		Pool: pool,
		log:  logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTimeout*time.Second)
	defer cancel()

	// Use PingContext for database/sql
	if err = pool.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().Msg("connected to the database")

	return database, nil
}

func (db *Database) Close() error {
	db.log.Info().Msg("closing database connection pool")
	// Close on *sql.DB returns an error
	return db.Pool.Close()
}
