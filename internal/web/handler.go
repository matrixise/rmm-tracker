package web

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matrixise/rmm-tracker/internal/health"
	"github.com/matrixise/rmm-tracker/internal/storage"
	"github.com/matrixise/rmm-tracker/internal/web/templates"
)

// WebHandler holds dependencies for web UI handlers.
type WebHandler struct {
	store   storage.Querier
	checker *health.Checker
}

// NewWebHandler creates a new WebHandler.
func NewWebHandler(store storage.Querier, checker *health.Checker) *WebHandler {
	return &WebHandler{store: store, checker: checker}
}

// Dashboard handles GET /
func (h *WebHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Dashboard().Render(r.Context(), w); err != nil {
		slog.Error("render dashboard", "error", err)
	}
}

// Wallets handles GET /wallets
func (h *WebHandler) Wallets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Wallets().Render(r.Context(), w); err != nil {
		slog.Error("render wallets", "error", err)
	}
}

// WalletDetail handles GET /wallets/{wallet}
func (h *WebHandler) WalletDetail(w http.ResponseWriter, r *http.Request) {
	wallet := chi.URLParam(r, "wallet")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.WalletDetail(wallet).Render(r.Context(), w); err != nil {
		slog.Error("render wallet detail", "error", err)
	}
}
