package routes

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/justinndidit/forex/internal/app"
)

func SetupAuthRoutes(app *app.Application) *chi.Mux {
	r := chi.NewRouter()

	r.Post("/countries/refresh", app.Handler.HandleRefresh)
	r.Get("/countries", app.Handler.HandleGetCountry)
	r.Get("/countries/{name}", app.Handler.HandleGetCountryByName)
	r.Get("/status", app.Handler.HandleStatus)
	r.Get("/countries/image", app.Handler.HandleGetImage)
	r.Delete("/countries/{name}", app.Handler.HandleDeleteCountryByName)

	r.Get("/kaithheathcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Country and Exchange API"))
	})

	return r
}
