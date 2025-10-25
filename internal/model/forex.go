package model

import (
	"database/sql"
	"time"
)

type CountryCurrency struct {
	Code string `json:"code"`
}

type Country struct {
	Name       string            `json:"name"`
	Capital    string            `json:"capital"`
	Region     string            `json:"region"`
	Population int64             `json:"population"`
	FlagURL    string            `json:"flag"`
	Currencies []CountryCurrency `json:"currencies"`
}

type ExchangeRates struct {
	Rates map[string]float64 `json:"rates"`
}

type CountryDBRow struct {
	ID              int64
	Name            string
	Capital         sql.NullString
	Region          sql.NullString
	Population      int64
	CurrencyCode    sql.NullString
	ExchangeRate    sql.NullFloat64
	EstimatedGDP    sql.NullFloat64
	FlagURL         sql.NullString
	LastRefreshedAt sql.NullTime // Use sql.NullTime
}

type CountryResponse struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Capital         *string    `json:"capital"`
	Region          *string    `json:"region"`
	Population      int64      `json:"population"`
	CurrencyCode    *string    `json:"currency_code"`
	ExchangeRate    *float64   `json:"exchange_rate"`
	EstimatedGDP    *float64   `json:"estimated_gdp"`
	FlagURL         *string    `json:"flag_url"`
	LastRefreshedAt *time.Time `json:"last_refreshed_at"`
}

func (db *CountryDBRow) ToResponse() CountryResponse {
	var capital, region, currencyCode, flagURL *string
	var exchangeRate, estimatedGDP *float64
	var lastRefreshed *time.Time

	if db.Capital.Valid {
		capital = &db.Capital.String
	}
	if db.Region.Valid {
		region = &db.Region.String
	}
	if db.CurrencyCode.Valid {
		currencyCode = &db.CurrencyCode.String
	}
	if db.FlagURL.Valid {
		flagURL = &db.FlagURL.String
	}
	if db.ExchangeRate.Valid {
		exchangeRate = &db.ExchangeRate.Float64
	}
	if db.EstimatedGDP.Valid {
		estimatedGDP = &db.EstimatedGDP.Float64
	}
	if db.LastRefreshedAt.Valid {
		lastRefreshed = &db.LastRefreshedAt.Time
	}

	return CountryResponse{
		ID:              db.ID,
		Name:            db.Name,
		Population:      db.Population,
		Capital:         capital,
		Region:          region,
		CurrencyCode:    currencyCode,
		ExchangeRate:    exchangeRate,
		EstimatedGDP:    estimatedGDP,
		FlagURL:         flagURL,
		LastRefreshedAt: lastRefreshed,
	}
}

// Convert slice
func ToCountryResponses(dbCountries []CountryDBRow) []CountryResponse {
	responses := make([]CountryResponse, len(dbCountries))
	for i, country := range dbCountries {
		responses[i] = country.ToResponse()
	}
	return responses
}

type CountryFilters struct {
	Region   *string
	Currency *string
	SortKey  string
}

type Stats struct {
	TotalCountries  int          `db:"total_countries"`
	LastRefreshedAt sql.NullTime `db:"last_refreshed_at"`
}
type StatsResponse struct {
	TotalCountries  int        `json:"total_countries"`
	LastRefreshedAt *time.Time `json:"last_refreshed_at"`
}

func (s *Stats) ToResponse() StatsResponse {
	var lastRefresh *time.Time

	if s.LastRefreshedAt.Valid {
		lastRefresh = &s.LastRefreshedAt.Time
	}

	return StatsResponse{
		TotalCountries:  s.TotalCountries,
		LastRefreshedAt: lastRefresh,
	}
}
