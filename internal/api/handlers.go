package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/matrixise/rmm-tracker/internal/storage"
)

// Handler holds dependencies for API handlers.
type Handler struct {
	store storage.Querier
}

// NewHandler creates a new Handler.
func NewHandler(store storage.Querier) *Handler {
	return &Handler{store: store}
}

// GetBalances handles GET /api/v1/balances
// Query params: wallet, symbol, limit (default 100)
func (h *Handler) GetBalances(w http.ResponseWriter, r *http.Request) {
	wallet := r.URL.Query().Get("wallet")
	symbol := r.URL.Query().Get("symbol")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v <= 0 {
			http.Error(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
		limit = v
	}

	balances, err := h.store.GetBalances(r.Context(), wallet, symbol, limit)
	if err != nil {
		slog.Error("GetBalances query failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if balances == nil {
		balances = []storage.TokenBalance{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(balances); err != nil {
		slog.Error("GetBalances encode failed", "error", err)
	}
}

// GetWeeklyBalances handles GET /api/v1/wallets/{wallet}/balances/weekly
func (h *Handler) GetWeeklyBalances(w http.ResponseWriter, r *http.Request) {
	wallet := chi.URLParam(r, "wallet")
	if wallet == "" {
		http.Error(w, "wallet parameter required", http.StatusBadRequest)
		return
	}

	balances, err := h.store.GetWeeklyBalances(r.Context(), wallet)
	if err != nil {
		slog.Error("GetWeeklyBalances query failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if balances == nil {
		balances = []storage.WeeklyBalance{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(balances); err != nil {
		slog.Error("GetWeeklyBalances encode failed", "error", err)
	}
}

// GetWeeklyReport handles GET /api/v1/wallets/{wallet}/report/weekly
// Optional query param: weeks (integer >= 2, default 2)
func (h *Handler) GetWeeklyReport(w http.ResponseWriter, r *http.Request) {
	wallet := chi.URLParam(r, "wallet")
	if wallet == "" {
		http.Error(w, "wallet parameter required", http.StatusBadRequest)
		return
	}

	weeks := 2
	if weeksStr := r.URL.Query().Get("weeks"); weeksStr != "" {
		v, err := strconv.Atoi(weeksStr)
		if err != nil || v < 2 || v > 52 {
			http.Error(w, "weeks must be an integer between 2 and 52", http.StatusBadRequest)
			return
		}
		weeks = v
	}

	report, err := h.store.GetWeeklyReport(r.Context(), wallet, weeks)
	if err != nil {
		slog.Error("GetWeeklyReport query failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if report == nil {
		report = []storage.WeeklyReport{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		slog.Error("GetWeeklyReport encode failed", "error", err)
	}
}

// GetDailyBalances handles GET /api/v1/wallets/{wallet}/balances/daily
func (h *Handler) GetDailyBalances(w http.ResponseWriter, r *http.Request) {
	wallet := chi.URLParam(r, "wallet")
	if wallet == "" {
		http.Error(w, "wallet parameter required", http.StatusBadRequest)
		return
	}

	balances, err := h.store.GetDailyBalances(r.Context(), wallet)
	if err != nil {
		slog.Error("GetDailyBalances query failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if balances == nil {
		balances = []storage.DailyBalance{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(balances); err != nil {
		slog.Error("GetDailyBalances encode failed", "error", err)
	}
}

// GetDailyReport handles GET /api/v1/wallets/{wallet}/report/daily
// Optional query param: days (integer 2-365, default 31)
func (h *Handler) GetDailyReport(w http.ResponseWriter, r *http.Request) {
	wallet := chi.URLParam(r, "wallet")
	if wallet == "" {
		http.Error(w, "wallet parameter required", http.StatusBadRequest)
		return
	}

	days := 31
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		v, err := strconv.Atoi(daysStr)
		if err != nil || v < 2 || v > 365 {
			http.Error(w, "days must be an integer between 2 and 365", http.StatusBadRequest)
			return
		}
		days = v
	}

	report, err := h.store.GetDailyReport(r.Context(), wallet, days)
	if err != nil {
		slog.Error("GetDailyReport query failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if report == nil {
		report = []storage.DailyReport{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		slog.Error("GetDailyReport encode failed", "error", err)
	}
}

// GetWallets handles GET /api/v1/wallets
func (h *Handler) GetWallets(w http.ResponseWriter, r *http.Request) {
	wallets, err := h.store.GetWallets(r.Context())
	if err != nil {
		slog.Error("GetWallets query failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if wallets == nil {
		wallets = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(wallets); err != nil {
		slog.Error("GetWallets encode failed", "error", err)
	}
}
