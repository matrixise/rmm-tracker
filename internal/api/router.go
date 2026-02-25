package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/matrixise/rmm-tracker/internal/health"
	"github.com/matrixise/rmm-tracker/internal/storage"
	"github.com/matrixise/rmm-tracker/internal/web"
)

// slogLogger is a chi middleware that logs HTTP requests using slog.
func slogLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		slog.Info("HTTP",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start),
			"remote", r.RemoteAddr,
		)
	})
}

// NewRouter creates a Chi router with all application routes.
// When enableWeb is true, the web UI is mounted at "/" using the provided store and checker.
func NewRouter(healthHandler http.HandlerFunc, apiHandler *Handler, checker *health.Checker, enableWeb bool, store storage.Querier) *chi.Mux {
	r := chi.NewRouter()
	r.Use(slogLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	r.Get("/health", healthHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/balances", apiHandler.GetBalances)
		r.Get("/wallets", apiHandler.GetWallets)
		r.Get("/wallets/{wallet}/balances/weekly", apiHandler.GetWeeklyBalances)
		r.Get("/wallets/{wallet}/report/weekly", apiHandler.GetWeeklyReport)
		r.Get("/wallets/{wallet}/balances/daily", apiHandler.GetDailyBalances)
		r.Get("/wallets/{wallet}/report/daily", apiHandler.GetDailyReport)
	})

	if enableWeb {
		webHandler := web.NewWebHandler(store, checker)
		r.Mount("/", web.NewWebRouter(webHandler))
	}

	return r
}
