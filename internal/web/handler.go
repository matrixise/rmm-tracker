package web

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/matrixise/rmm-tracker/internal/health"
	"github.com/matrixise/rmm-tracker/internal/storage"
	"github.com/matrixise/rmm-tracker/internal/web/templates"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// WebHandler holds dependencies for web UI handlers.
type WebHandler struct {
	store         storage.Querier
	checker       *health.Checker
	changelogHTML string
}

// NewWebHandler creates a new WebHandler, sets the app version for templates,
// and pre-renders the changelog Markdown to HTML.
func NewWebHandler(store storage.Querier, checker *health.Checker, version string, changelogMD []byte) *WebHandler {
	templates.AppVersion = version

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	var buf bytes.Buffer
	rendered := "<p>Could not render changelog.</p>"
	if err := md.Convert(changelogMD, &buf); err == nil {
		rendered = buf.String()
	}

	return &WebHandler{store: store, checker: checker, changelogHTML: rendered}
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
	wallet := strings.ToLower(chi.URLParam(r, "wallet"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.WalletDetail(wallet).Render(r.Context(), w); err != nil {
		slog.Error("render wallet detail", "error", err)
	}
}

// Changelog handles GET /changelog
func (h *WebHandler) Changelog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Changelog(h.changelogHTML).Render(r.Context(), w); err != nil {
		slog.Error("render changelog", "error", err)
	}
}
