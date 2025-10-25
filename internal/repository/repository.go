package repository

import (
	"context"
	"database/sql" // Import standard sql
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/justinndidit/forex/internal/database"
	"github.com/justinndidit/forex/internal/errs"
	"github.com/justinndidit/forex/internal/model"
	"github.com/rs/zerolog"
)

// Define constants for table names
const (
	countriesTable = "countries"
	appStatusTable = "app_status"
	batchSize      = 1000 // Standard batch size for bulk inserts
)

type ForexRepository struct {
	logger *zerolog.Logger
	db     *database.Database
}

func NewForexRepository(logger *zerolog.Logger, db *database.Database) *ForexRepository {
	return &ForexRepository{
		logger: logger,
		db:     db,
	}
}

func (r *ForexRepository) UpdateCountries(ctx context.Context, rowsToInsert []model.CountryDBRow, refreshTime time.Time) error {
	tx, err := r.db.Pool.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to begin transaction")
		return err
	}
	defer tx.Rollback()

	dropTempTableSQL := `DROP TEMPORARY TABLE IF EXISTS temp_countries;`
	if _, err = tx.ExecContext(ctx, dropTempTableSQL); err != nil {
		r.logger.Error().Err(err).Msg("Failed to drop old temporary table")
		return err
	}

	createTempTableSQL := `
        CREATE TEMPORARY TABLE temp_countries (
            name VARCHAR(256) NOT NULL PRIMARY KEY,
            capital VARCHAR(256),
            region VARCHAR(256),
            population BIGINT NOT NULL,
            currency_code VARCHAR(20),
            exchange_rate DECIMAL(15, 6),
            estimated_gdp DECIMAL(20, 2),
            flag_url VARCHAR(256),
            last_refreshed_at TIMESTAMP NOT NULL
        ) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    `
	if _, err = tx.ExecContext(ctx, createTempTableSQL); err != nil {
		r.logger.Error().Err(err).Msg("Failed to create temporary table")
		return err
	}

	if len(rowsToInsert) == 0 {
		r.logger.Info().Msg("No countries to update, skipping bulk insert.")
	} else {
		stmtSQL := `
            INSERT INTO temp_countries (
                name, capital, region, population,
                currency_code, exchange_rate, estimated_gdp,
                flag_url, last_refreshed_at
            ) VALUES %s
        `

		for i := 0; i < len(rowsToInsert); i += batchSize {
			end := i + batchSize
			if end > len(rowsToInsert) {
				end = len(rowsToInsert)
			}
			batch := rowsToInsert[i:end]

			valueStrings := make([]string, 0, len(batch))
			valueArgs := make([]any, 0, len(batch)*9)

			for _, row := range batch {
				valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
				valueArgs = append(valueArgs,
					row.Name, row.Capital, row.Region, row.Population,
					row.CurrencyCode, row.ExchangeRate, row.EstimatedGDP,
					row.FlagURL, row.LastRefreshedAt,
				)
			}

			batchStmt := fmt.Sprintf(stmtSQL, strings.Join(valueStrings, ","))

			if _, err = tx.ExecContext(ctx, batchStmt, valueArgs...); err != nil {
				r.logger.Error().Err(err).Msg("Failed to bulk insert batch to temp table")
				return err
			}
		}
	}
	// --- End of Batch Insert ---

	// --- MySQL "UPSERT" syntax ---
	mergeSQL := `
        INSERT INTO countries (
            name, capital, region, population,
            currency_code, exchange_rate, estimated_gdp,
            flag_url, last_refreshed_at
        )
        SELECT * FROM temp_countries
        ON DUPLICATE KEY UPDATE
            capital = VALUES(capital),
            region = VALUES(region),
            population = VALUES(population),
            currency_code = VALUES(currency_code),
            exchange_rate = VALUES(exchange_rate),
            estimated_gdp = VALUES(estimated_gdp),
            flag_url = VALUES(flag_url),
            last_refreshed_at = VALUES(last_refreshed_at);
    `
	if _, err = tx.ExecContext(ctx, mergeSQL); err != nil {
		r.logger.Error().Err(err).Msg("Failed to merge from temp table")
		return err
	}

	// Use ? for placeholder
	updateStatusSQL := fmt.Sprintf("UPDATE %s SET last_refreshed_at = ? WHERE id = 1", appStatusTable)
	if _, err = tx.ExecContext(ctx, updateStatusSQL, refreshTime); err != nil {
		r.logger.Error().Err(err).Msg("Failed to update app_status")
		return err
	}

	// If all commands succeeded, commit the transaction
	return tx.Commit()
}

func (r *ForexRepository) GetCountries(ctx context.Context, filters model.CountryFilters) ([]model.CountryDBRow, error) {
	// Use constants for table names and be explicit with columns
	baseQuery := fmt.Sprintf(`
        SELECT
            id, name, capital, region, population, currency_code,
            exchange_rate, estimated_gdp, flag_url, last_refreshed_at
        FROM %s
    `, countriesTable)

	whereClauses := []string{}
	args := []any{}

	if filters.Region != nil {
		whereClauses = append(whereClauses, "region = ?")
		args = append(args, *filters.Region)
	}

	if filters.Currency != nil {
		whereClauses = append(whereClauses, "currency_code = ?")
		args = append(args, *filters.Currency)
	}

	finalQuery := baseQuery
	if len(whereClauses) > 0 {
		finalQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Whitelist approach for sorting is excellent
	var orderByClause string
	switch filters.SortKey {
	case "gdp_desc":
		// FIX: Use MySQL syntax for NULLS LAST
		orderByClause = " ORDER BY estimated_gdp IS NULL ASC, estimated_gdp DESC"
	case "gdp_asc":
		// FIX: Use MySQL syntax for NULLS FIRST
		orderByClause = " ORDER BY estimated_gdp IS NULL DESC, estimated_gdp ASC"
	case "population_desc":
		orderByClause = " ORDER BY population DESC"
	case "population_asc":
		orderByClause = " ORDER BY population ASC"
	case "name_desc":
		orderByClause = " ORDER BY name DESC"
	default:
		orderByClause = " ORDER BY name ASC"
	}
	finalQuery += orderByClause

	rows, err := r.db.Pool.QueryContext(ctx, finalQuery, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to query countries")
		return nil, err
	}
	defer rows.Close()

	// Use the helper to scan rows
	return r.scanCountries(rows)
}

func (r *ForexRepository) GetCountryByName(ctx context.Context, name string) (*model.CountryDBRow, error) {
	stmt := fmt.Sprintf(`
        SELECT
            id, name, capital, region, population,
            currency_code, exchange_rate, estimated_gdp,
            flag_url, last_refreshed_at
        FROM %s
        WHERE name = ?
    `, countriesTable)

	row := r.db.Pool.QueryRowContext(ctx, stmt, name)

	var c model.CountryDBRow
	err := row.Scan(
		&c.ID, &c.Name, &c.Capital, &c.Region, &c.Population,
		&c.CurrencyCode, &c.ExchangeRate, &c.EstimatedGDP,
		&c.FlagURL, &c.LastRefreshedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Info().Err(err).Msg("No country found")
			return nil, errs.ErrNotFound
		}
		r.logger.Error().Err(err).Msg("Failed to scan row")
		return nil, err
	}
	return &c, nil
}

func (r *ForexRepository) GetTotalCountries(ctx context.Context) (int, error) {
	var total int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s;", countriesTable)

	if err := r.db.Pool.QueryRowContext(ctx, query).Scan(&total); err != nil {
		r.logger.Error().Err(err).Msg("Failed to get total countries count")
		return 0, err
	}
	return total, nil
}

func (r *ForexRepository) GetTop5ByGDP(ctx context.Context) ([]model.CountryDBRow, error) {
	// --- BUG FIX: Use MySQL syntax for ORDER BY ... NULLS LAST ---
	// `estimated_gdp IS NULL ASC` puts NULLs (1) after NOT NULLs (0).
	query := fmt.Sprintf(`
        SELECT
            id, name, capital, region, population,
            currency_code, exchange_rate, estimated_gdp,
            flag_url, last_refreshed_at
        FROM %s
        ORDER BY estimated_gdp IS NULL ASC, estimated_gdp DESC
        LIMIT 5;
    `, countriesTable)

	rows, err := r.db.Pool.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to query top 5 countries")
		return nil, err
	}
	defer rows.Close()

	// Use the helper to scan rows
	return r.scanCountries(rows)
}

func (r *ForexRepository) DeleteByName(ctx context.Context, name string) error {
	stmt := fmt.Sprintf("DELETE FROM %s WHERE name = ?", countriesTable)

	result, err := r.db.Pool.ExecContext(ctx, stmt, name)
	if err != nil {
		r.logger.Error().Err(err).Msg("Delete query failed!")
		return fmt.Errorf("failed to execute delete string query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get rows affected")
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// This is good, return a standard error
		return errs.ErrNotFound
	}

	return nil
}

func (r *ForexRepository) GetStats(ctx context.Context) (*model.Stats, error) {
	stmt := fmt.Sprintf(`
        SELECT
            (SELECT COUNT(*) FROM %s) AS total_countries,
            (SELECT last_refreshed_at FROM %s WHERE id = 1) AS last_refreshed_at;
    `, countriesTable, appStatusTable)

	row := r.db.Pool.QueryRowContext(ctx, stmt)

	var stats model.Stats
	if err := row.Scan(&stats.TotalCountries, &stats.LastRefreshedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Warn().Msg("No stats row found (app_status table might be empty)")
			// Return an empty/zero struct is fine
			return &stats, nil
		}
		r.logger.Error().Err(err).Msg("Failed to scan stats row")
		return nil, fmt.Errorf("failed to scan stats: %w", err)
	}

	return &stats, nil
}

// --- REFACTOR: Private helper to reduce code duplication ---
// scanCountries iterates over sql.Rows and scans them into a slice.
func (r *ForexRepository) scanCountries(rows *sql.Rows) ([]model.CountryDBRow, error) {
	countries := []model.CountryDBRow{}
	for rows.Next() {
		var c model.CountryDBRow
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Capital,
			&c.Region,
			&c.Population,
			&c.CurrencyCode,
			&c.ExchangeRate,
			&c.EstimatedGDP,
			&c.FlagURL,
			&c.LastRefreshedAt,
		); err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan country row")
			return nil, err
		}
		countries = append(countries, c)
	}

	// Always check for an error from the rows.Next() loop
	if err := rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error during row iteration")
		return nil, err
	}

	return countries, nil
}
