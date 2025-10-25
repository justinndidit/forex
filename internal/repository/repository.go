package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justinndidit/forex/internal/database"
	"github.com/justinndidit/forex/internal/errs"
	"github.com/justinndidit/forex/internal/model"
	"github.com/rs/zerolog"
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

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to begin transaction")
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
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
		) ON COMMIT DROP;
	`)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to create temporary table")
		return err
	}
	copyData := make([][]any, len(rowsToInsert))
	for i, row := range rowsToInsert {
		copyData[i] = []any{
			row.Name,
			row.Capital,
			row.Region,
			row.Population,
			row.CurrencyCode,
			row.ExchangeRate,
			row.EstimatedGDP,
			row.FlagURL,
			row.LastRefreshedAt,
		}
	}

	columnNames := []string{
		"name", "capital", "region", "population",
		"currency_code", "exchange_rate", "estimated_gdp",
		"flag_url", "last_refreshed_at",
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"temp_countries"},
		columnNames,
		pgx.CopyFromRows(copyData),
	)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to bulk copy to temp table")
		return err
	}

	mergeSQL := `
		INSERT INTO countries (
			name, capital, region, population,
			currency_code, exchange_rate, estimated_gdp,
			flag_url, last_refreshed_at
		)
		SELECT * FROM temp_countries
		ON CONFLICT (name) DO UPDATE SET -- This is the PostgreSQL "upsert"
			capital = EXCLUDED.capital,
			region = EXCLUDED.region,
			population = EXCLUDED.population,
			currency_code = EXCLUDED.currency_code,
			exchange_rate = EXCLUDED.exchange_rate,
			estimated_gdp = EXCLUDED.estimated_gdp,
			flag_url = EXCLUDED.flag_url,
			last_refreshed_at = EXCLUDED.last_refreshed_at;
	`
	_, err = tx.Exec(ctx, mergeSQL)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to merge from temp table")
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE app_status SET last_refreshed_at = $1 WHERE id = 1", refreshTime)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to update app_status")
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		r.logger.Error().Err(err).Msg("Failed to commit transaction")
		return err
	}

	r.logger.Info().Int("count", len(rowsToInsert)).Msg("Database refresh successful")
	return nil
}

func (r *ForexRepository) GetCountries(ctx context.Context, filters model.CountryFilters) ([]model.CountryDBRow, error) {

	baseQuery := "SELECT * FROM countries"
	whereClauses := []string{}
	args := []interface{}{}
	argCount := 1

	if filters.Region != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("region = $%d", argCount))
		args = append(args, *filters.Region)
		argCount++
	}

	if filters.Currency != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("currency_code = $%d", argCount))
		args = append(args, *filters.Currency)
		argCount++
	}

	finalQuery := baseQuery
	if len(whereClauses) > 0 {
		finalQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	orderByClause := ""
	switch filters.SortKey {
	case "gdp_desc":
		orderByClause = " ORDER BY estimated_gdp DESC NULLS LAST"
	case "gdp_asc":
		orderByClause = " ORDER BY estimated_gdp ASC NULLS FIRST"
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

	rows, err := r.db.Pool.Query(ctx, finalQuery, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to query countries")
		return nil, err
	}
	defer rows.Close()

	countries, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.CountryDBRow])
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to scan countries")
		return nil, err
	}

	return countries, nil
}

func (r *ForexRepository) GetCountryByName(ctx context.Context, name string) (*model.CountryDBRow, error) {
	stmt := `
		SELECT
			*
		FROM
			countries
		WHERE
			(name = @name::varchar)
	`
	rows, err := r.db.Pool.Query(ctx, stmt, pgx.NamedArgs{
		"name": name,
	})

	if err != nil {
		r.logger.Error().Err(err).Msg("Query Failed!")
		return nil, err
	}

	record, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[model.CountryDBRow])

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info().Msg("No rows matching query")
			return nil, nil
		}
		r.logger.Error().Err(err).Msg("Failed to collect row")
		return nil, err
	}
	return record, nil
}

func (r *ForexRepository) GetTotalCountries(ctx context.Context) (int, error) {
	var total int

	query := "SELECT COUNT(*) FROM countries;"

	err := r.db.Pool.QueryRow(ctx, query).Scan(&total)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get total countries count")
		return 0, err
	}

	return total, nil
}

func (r *ForexRepository) GetTop5ByGDP(ctx context.Context) ([]model.CountryDBRow, error) {
	query := `
		SELECT
			id, name, capital, region, population,
			currency_code, exchange_rate, estimated_gdp,
			flag_url, last_refreshed_at
		FROM countries
		ORDER BY estimated_gdp DESC NULLS LAST
		LIMIT 5;
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to query top 5 countries")
		return nil, err
	}
	defer rows.Close()

	// Use pgx.CollectRows for efficient scanning into a slice
	topCountries, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.CountryDBRow])
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to scan top 5 countries")
		return nil, err
	}

	return topCountries, nil
}

func (r *ForexRepository) DeleteByName(ctx context.Context, name string) error {
	stmt := `DELETE FROM countries WHERE name = @name`

	rows, err := r.db.Pool.Exec(ctx, stmt, pgx.NamedArgs{
		"name": name,
	})

	if err != nil {
		r.logger.Error().Err(err).Msg("Delete query failed!")
		return fmt.Errorf("failed to execute delete string query: %w", err)
	}

	if rows.RowsAffected() == 0 {
		return errs.ErrNotFound
	}

	return nil

}

func (r *ForexRepository) GetStats(ctx context.Context) (*model.Stats, error) {
	stmt := `

	SELECT
		(SELECT COUNT(*) FROM countries) AS total_countries,
		(SELECT last_refreshed_at FROM app_status WHERE id = 1) AS last_refreshed_at;
	`
	row, err := r.db.Pool.Query(ctx, stmt)

	if err != nil {
		r.logger.Error().Err(err).Msg("Delete query failed!")
		return nil, fmt.Errorf("failed to execute delete string query: %w", err)
	}
	defer row.Close()
	stats, err := pgx.CollectOneRow(row, pgx.RowToStructByName[model.Stats])
	if err != nil {
		r.logger.Error().Err(err).Msg("No")
		return nil, fmt.Errorf("failed to collect row: %w", err)
	}
	return &stats, nil
}
