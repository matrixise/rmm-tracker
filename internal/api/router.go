package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/matrixise/rmm-tracker/internal/health"
	"github.com/matrixise/rmm-tracker/internal/storage"
	"github.com/matrixise/rmm-tracker/internal/web"
)

// NewRouter creates a Chi router with all application routes.
// When enableWeb is true, the web UI is mounted at "/" using the provided store and checker.
func NewRouter(healthHandler http.HandlerFunc, apiHandler *Handler, checker *health.Checker, enableWeb bool, store storage.Storer) *chi.Mux {
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

	if enableWeb {
		webHandler := web.NewWebHandler(store, checker)
		r.Mount("/", web.NewWebRouter(webHandler))
	}

	return r
}
