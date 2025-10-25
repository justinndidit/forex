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
	// --- UPDATED LOGIC ---
	// Map PostgreSQL sslmode to MySQL tls parameter
	var tlsMode string
	switch cfg.Database.SSLMode {
	case "require", "verify-full":
		tlsMode = "true"
	case "skip-verify":
		// This is the new option to fix the x509 error
		tlsMode = "skip-verify"
	case "disable":
		tlsMode = "false"
	default:
		// Default to false (no TLS) if unset or invalid
		tlsMode = "false"
	}

	// Build the DSN: user:password@tcp(host:port)/dbname?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&tls=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		strconv.Itoa(cfg.Database.Port),
		cfg.Database.Name,
		tlsMode, // Use the tlsMode from our switch statement
	)
	// --- END OF UPDATE ---

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
