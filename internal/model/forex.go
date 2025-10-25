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
	ID              int             `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	Capital         string          `json:"capital" db:"capital"`
	Region          string          `json:"region" db:"region"`
	Population      int64           `json:"population" db:"population"`
	CurrencyCode    sql.NullString  `json:"currency_code" db:"currency_code"`
	ExchangeRate    sql.NullFloat64 `json:"exchange_rate" db:"exchange_rate"`
	EstimatedGDP    sql.NullFloat64 `json:"estimated_gdp" db:"estimated_gdp"`
	FlagURL         string          `json:"flag_url" db:"flag_url"`
	LastRefreshedAt time.Time       `json:"last_refreshed_at" db:"last_refreshed_at"`
}

type CountryResponse struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Capital         string    `json:"capital"`
	Region          string    `json:"region"`
	Population      int64     `json:"population"`
	CurrencyCode    string    `json:"currency_code"`
	ExchangeRate    float64   `json:"exchange_rate"`
	EstimatedGDP    float64   `json:"estimated_gdp"`
	FlagURL         string    `json:"flag_url"`
	LastRefreshedAt time.Time `json:"last_refreshed_at"`
}

func (db *CountryDBRow) ToResponse() CountryResponse {
	return CountryResponse{
		ID:              db.ID,
		Name:            db.Name,
		Capital:         db.Capital,
		Region:          db.Region,
		Population:      db.Population,
		CurrencyCode:    db.CurrencyCode.String,  // Will be "" if not valid
		ExchangeRate:    db.ExchangeRate.Float64, // Will be 0 if not valid
		EstimatedGDP:    db.EstimatedGDP.Float64, // Will be 0 if not valid
		FlagURL:         db.FlagURL,
		LastRefreshedAt: db.LastRefreshedAt,
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
	TotalCountries int       `json:"total_countries" db:"total_countries"`
	LastReference  time.Time `json:"last_refreshed_at" db:"last_refreshed_at"`
}
