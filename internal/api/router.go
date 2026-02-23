package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a Chi router with all application routes.
func NewRouter(healthHandler http.HandlerFunc, apiHandler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	r.Get("/health", healthHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/balances", apiHandler.GetBalances)
		r.Get("/wallets", apiHandler.GetWallets)
		r.Get("/wallets/{wallet}/balances/weekly", apiHandler.GetWeeklyBalances)
		r.Get("/wallets/{wallet}/report/weekly", apiHandler.GetWeeklyReport)
	})

	return r
}
