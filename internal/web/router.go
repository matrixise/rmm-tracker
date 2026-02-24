package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewWebRouter creates a Chi router with web UI routes.
func NewWebRouter(h *WebHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.Dashboard)
	r.Get("/wallets", h.Wallets)
	r.Get("/wallets/{wallet}", h.WalletDetail)
	return r
}
