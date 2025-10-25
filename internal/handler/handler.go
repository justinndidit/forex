package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/justinndidit/forex/internal/database"
	"github.com/justinndidit/forex/internal/errs"
	"github.com/justinndidit/forex/internal/model"
	"github.com/justinndidit/forex/internal/repository"

	"github.com/justinndidit/forex/internal/util"
	"github.com/rs/zerolog"
)

type ForexHandler struct {
	logger *zerolog.Logger
	db     *database.Database
	repo   *repository.ForexRepository
	imgGen *util.ImageService
}

func NewForexHandler(logger *zerolog.Logger, db *database.Database, repo *repository.ForexRepository, imgGen *util.ImageService) *ForexHandler {
	return &ForexHandler{
		logger: logger,
		db:     db,
		repo:   repo,
		imgGen: imgGen,
	}
}

func (h *ForexHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	const countriesURL = "https://restcountries.com/v2/all?fields=name,capital,region,population,flag,currencies"
	const ratesURL = "https://open.er-api.com/v6/latest/USD"
	ctx := r.Context()

	var wg sync.WaitGroup
	resultChan := make(chan util.FetchResult, 2)
	wg.Add(2)
	go util.FetchData(countriesURL, &wg, resultChan)
	go util.FetchData(ratesURL, &wg, resultChan)
	wg.Wait()
	close(resultChan)

	var countriesList []model.Country
	var exchangeData model.ExchangeRates
	var failedURLs []string

	for result := range resultChan {
		if result.Err != nil {
			h.logger.Error().Err(result.Err).Msg("Failed to fetch from API: " + result.URL)
			failedURLs = append(failedURLs, result.URL)
			continue
		}

		switch result.URL {
		case countriesURL:
			if err := json.Unmarshal(result.Body, &countriesList); err != nil {
				h.logger.Error().Err(err).Msg("Failed to unmarshal countries data")
				failedURLs = append(failedURLs, "countries API (unmarshal fail)")
			}
		case ratesURL:
			if err := json.Unmarshal(result.Body, &exchangeData); err != nil {
				h.logger.Error().Err(err).Msg("Failed to unmarshal exchange rate data")
				failedURLs = append(failedURLs, "exchange rate API (unmarshal fail)")
			}
		}
	}

	if len(failedURLs) > 0 {
		details := fmt.Sprintf("Could not fetch data from: %s", strings.Join(failedURLs, ", "))
		util.WriteJsonError(w, http.StatusServiceUnavailable, "External data source unavailable", &details)
		return
	}

	if len(countriesList) == 0 || exchangeData.Rates == nil {
		details := "API returned empty or invalid data"
		h.logger.Error().Msg(details)
		util.WriteJsonError(w, http.StatusServiceUnavailable, "External data source unavailable", &details)
		return
	}
	rates := exchangeData.Rates
	refreshTime := time.Now()
	var rowsToInsert []model.CountryDBRow

	for _, country := range countriesList {
		dbRow := model.CountryDBRow{
			// --- These fields are OK ---
			Name:       strings.ToLower(country.Name),
			Population: country.Population,

			// --- Corrected fields ---
			Capital: sql.NullString{
				String: strings.ToLower(country.Capital),
				Valid:  country.Capital != "", // Set to 'true' only if it's not empty
			},
			Region: sql.NullString{
				String: strings.ToLower(country.Region),
				Valid:  country.Region != "",
			},
			FlagURL: sql.NullString{
				String: country.FlagURL, // No need to lowercase a URL
				Valid:  country.FlagURL != "",
			},
			LastRefreshedAt: sql.NullTime{
				Time:  refreshTime,
				Valid: true, // We always set this
			},
		}
		if len(country.Currencies) > 0 {
			code := country.Currencies[0].Code
			dbRow.CurrencyCode = sql.NullString{String: code, Valid: true}

			if rate, ok := rates[code]; ok {
				dbRow.ExchangeRate = sql.NullFloat64{Float64: rate, Valid: true}
				randomMultiplier := util.RandFloatRange()
				gdp := (float64(country.Population) * randomMultiplier) / rate
				dbRow.EstimatedGDP = sql.NullFloat64{Float64: gdp, Valid: true}
			}

		} else {
			dbRow.EstimatedGDP = sql.NullFloat64{Float64: 0, Valid: true}
		}

		rowsToInsert = append(rowsToInsert, dbRow)
	}

	err := h.repo.UpdateCountries(ctx, rowsToInsert, refreshTime)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to update database")
		util.WriteJsonError(w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}
	go h.generateAndLogSummary(context.Background(), refreshTime)

	util.WriteJsonSuccess(w, http.StatusOK, map[string]string{"message": "Database refresh successfully initiated"})
}

func (h *ForexHandler) generateAndLogSummary(ctx context.Context, refreshTime time.Time) {
	total, err := h.repo.GetTotalCountries(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("ImageGen: Failed to get total countries")
		return
	}

	top5, err := h.repo.GetTop5ByGDP(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("ImageGen: Failed to get top 5 countries by GDP")
		return
	}

	err = h.imgGen.GenerateSummary(total, top5, refreshTime)
	if err != nil {
		h.logger.Error().Err(err).Msg("ImageGen: Failed to generate summary image")
		return
	}

	h.logger.Info().Msg("Summary image generated successfully")
}

func (h *ForexHandler) HandleGetCountry(w http.ResponseWriter, r *http.Request) {

	filters := model.CountryFilters{}

	if region := r.URL.Query().Get("region"); region != "" {
		filters.Region = &region
	}

	if currency := r.URL.Query().Get("currency"); currency != "" {
		filters.Currency = &currency
	}

	filters.SortKey = r.URL.Query().Get("sort")
	if filters.SortKey == "" {
		filters.SortKey = "name_asc"
	}

	countries, err := h.repo.GetCountries(r.Context(), filters)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to Fetch Countries")
		util.WriteJsonError(w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	if len(countries) < 1 {
		h.logger.Info().Msg("Database is empty")
	}

	util.WriteJsonSuccess(w, http.StatusOK, model.ToCountryResponses(countries))

}

func (h *ForexHandler) HandleGetCountryByName(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "name")

	country, err := h.repo.GetCountryByName(r.Context(), param)

	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			util.WriteJsonError(w, http.StatusNotFound, "Country not found", nil)
			return
		}
		h.logger.Error().Err(err).Msg("Failed to Fetch Countries")
		util.WriteJsonError(w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	if country == nil {
		util.WriteJsonError(w, http.StatusNotFound, "Country not found", nil)
		return
	}

	util.WriteJsonSuccess(w, http.StatusOK, country.ToResponse())
}

func (h *ForexHandler) HandleDeleteCountryByName(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "name")

	err := h.repo.DeleteByName(r.Context(), param)

	if err != nil {

		switch {
		case errors.Is(err, errs.ErrNotFound):
			util.WriteJsonError(w, http.StatusNotFound, "Country not found", nil)
		default:
			h.logger.Error().Err(err).Msg("error deleting string")
			util.WriteJsonError(w, http.StatusInternalServerError, "Internal server error", nil)
		}
		return

	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (h *ForexHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.GetStats(r.Context())

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to Fetch status")
		util.WriteJsonError(w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	util.WriteJsonSuccess(w, http.StatusOK, stats.ToResponse())
}

func (h *ForexHandler) HandleGetImage(w http.ResponseWriter, r *http.Request) {
	imagePath := filepath.Join("cache", "summary.png")
	_, err := os.Stat(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			h.logger.Error().Err(err).Msg("Summary image not found at path: " + imagePath)
			util.WriteJsonError(w, http.StatusNotFound, "Summary image not found", nil)
			return
		}

		h.logger.Error().Err(err).Msg("Failed to stat image file: " + imagePath)
		util.WriteJsonError(w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	http.ServeFile(w, r, imagePath)
}
