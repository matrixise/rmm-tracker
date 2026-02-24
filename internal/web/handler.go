package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matrixise/rmm-tracker/internal/health"
	"github.com/matrixise/rmm-tracker/internal/storage"
	"github.com/matrixise/rmm-tracker/internal/web/templates"
)

var chartColors = []string{
	"#6366f1", // indigo
	"#10b981", // emerald
	"#f59e0b", // amber
	"#ef4444", // red
	"#8b5cf6", // violet
	"#06b6d4", // cyan
	"#f97316", // orange
	"#84cc16", // lime
}

type chartPoint struct {
	X string  `json:"x"`
	Y float64 `json:"y"`
}

type chartDataset struct {
	Label           string       `json:"label"`
	Data            []chartPoint `json:"data"`
	BorderColor     string       `json:"borderColor"`
	BackgroundColor string       `json:"backgroundColor"`
	Tension         float64      `json:"tension"`
	Fill            bool         `json:"fill"`
	PointRadius     int          `json:"pointRadius"`
}

// WebHandler holds dependencies for web UI handlers.
type WebHandler struct {
	store   storage.Storer
	checker *health.Checker
}

// NewWebHandler creates a new WebHandler.
func NewWebHandler(store storage.Storer, checker *health.Checker) *WebHandler {
	return &WebHandler{store: store, checker: checker}
}

// Dashboard handles GET /
func (h *WebHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := "unknown"
	statusColor := "bg-gray-400"
	lastUpdate := "—"

	if h.checker != nil {
		resp := h.checker.Check(ctx)
		switch resp.Status {
		case health.StatusOK:
			status = "ok"
			statusColor = "bg-green-500"
		case health.StatusDegraded:
			status = "degraded"
			statusColor = "bg-yellow-500"
		case health.StatusError:
			status = "error"
			statusColor = "bg-red-500"
		}
		lastUpdate = resp.Timestamp.UTC().Format(time.RFC3339)
	}

	wallets, err := h.store.GetWallets(ctx)
	if err != nil {
		slog.Error("Dashboard: GetWallets failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	balances, err := h.store.GetBalances(ctx, "", "", 1)
	if err != nil {
		slog.Error("Dashboard: GetBalances failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	tokenCount := 0
	if len(balances) > 0 {
		// distinct token count: re-use a 100-row sample to count distinct symbols
		sample, _ := h.store.GetBalances(ctx, "", "", 100)
		seen := make(map[string]struct{})
		for _, b := range sample {
			seen[b.Symbol] = struct{}{}
		}
		tokenCount = len(seen)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Dashboard(status, statusColor, len(wallets), tokenCount, lastUpdate).Render(ctx, w); err != nil {
		slog.Error("Dashboard: render failed", "error", err)
	}
}

// WalletDetail handles GET /wallets/{wallet}
func (h *WebHandler) WalletDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wallet := chi.URLParam(r, "wallet")

	balances, err := h.store.GetWeeklyBalances(ctx, wallet)
	if err != nil {
		slog.Error("WalletDetail: GetWeeklyBalances failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Group by symbol
	bySymbol := make(map[string][]storage.WeeklyBalance)
	for _, b := range balances {
		bySymbol[b.Symbol] = append(bySymbol[b.Symbol], b)
	}

	// Build Chart.js datasets — GetWeeklyBalances returns desc, reverse to chronological
	var datasets []chartDataset
	i := 0
	for symbol, rows := range bySymbol {
		for l, r := 0, len(rows)-1; l < r; l, r = l+1, r-1 {
			rows[l], rows[r] = rows[r], rows[l]
		}
		points := make([]chartPoint, len(rows))
		for j, row := range rows {
			points[j] = chartPoint{
				X: row.Week.Format(time.DateOnly),
				Y: row.Balance.InexactFloat64(),
			}
		}
		color := chartColors[i%len(chartColors)]
		datasets = append(datasets, chartDataset{
			Label:           symbol,
			Data:            points,
			BorderColor:     color,
			BackgroundColor: color + "26",
			Tension:         0.3,
			Fill:            false,
			PointRadius:     3,
		})
		i++
	}
	sort.Slice(datasets, func(i, j int) bool {
		return datasets[i].Label < datasets[j].Label
	})

	datasetsJSON, err := json.Marshal(datasets)
	if err != nil {
		slog.Error("WalletDetail: marshal failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	report, err := h.store.GetWeeklyReport(ctx, wallet, 8)
	if err != nil {
		slog.Error("WalletDetail: GetWeeklyReport failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if report == nil {
		report = []storage.WeeklyReport{}
	}

	dailyBalances, err := h.store.GetDailyBalances(ctx, wallet)
	if err != nil {
		slog.Error("WalletDetail: GetDailyBalances failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Build daily Chart.js datasets (same pattern as weekly)
	dailyBySymbol := make(map[string][]storage.DailyBalance)
	for _, b := range dailyBalances {
		dailyBySymbol[b.Symbol] = append(dailyBySymbol[b.Symbol], b)
	}
	var dailyDatasets []chartDataset
	i = 0
	for symbol, rows := range dailyBySymbol {
		for l, r := 0, len(rows)-1; l < r; l, r = l+1, r-1 {
			rows[l], rows[r] = rows[r], rows[l]
		}
		points := make([]chartPoint, len(rows))
		for j, row := range rows {
			points[j] = chartPoint{
				X: row.Day.Format(time.DateOnly),
				Y: row.Balance.InexactFloat64(),
			}
		}
		color := chartColors[i%len(chartColors)]
		dailyDatasets = append(dailyDatasets, chartDataset{
			Label:           symbol,
			Data:            points,
			BorderColor:     color,
			BackgroundColor: color + "26",
			Tension:         0.3,
			Fill:            false,
			PointRadius:     2,
		})
		i++
	}
	sort.Slice(dailyDatasets, func(i, j int) bool {
		return dailyDatasets[i].Label < dailyDatasets[j].Label
	})

	dailyDatasetsJSON, err := json.Marshal(dailyDatasets)
	if err != nil {
		slog.Error("WalletDetail: daily marshal failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	dailyReport, err := h.store.GetDailyReport(ctx, wallet, 31)
	if err != nil {
		slog.Error("WalletDetail: GetDailyReport failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if dailyReport == nil {
		dailyReport = []storage.DailyReport{}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.WalletDetail(wallet, string(datasetsJSON), report, string(dailyDatasetsJSON), dailyReport).Render(ctx, w); err != nil {
		slog.Error("WalletDetail: render failed", "error", err)
	}
}

// Wallets handles GET /wallets
func (h *WebHandler) Wallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	balances, err := h.store.GetBalances(ctx, "", "", 100)
	if err != nil {
		slog.Error("Wallets: GetBalances failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if balances == nil {
		balances = []storage.TokenBalance{}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Wallets(balances).Render(ctx, w); err != nil {
		slog.Error("Wallets: render failed", "error", err)
	}
}
