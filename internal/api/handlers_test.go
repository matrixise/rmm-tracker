package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matrixise/rmm-tracker/internal/storage"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock store ---

// mockStore implements storage.Storer for testing.
// Set only the function fields you need for each test.
type mockStore struct {
	getBalancesFn       func(ctx context.Context, wallet, symbol string, limit int) ([]storage.TokenBalance, error)
	getWeeklyBalancesFn func(ctx context.Context, wallet string) ([]storage.WeeklyBalance, error)
	getWeeklyReportFn   func(ctx context.Context, wallet string, weeks int) ([]storage.WeeklyReport, error)
	getWalletsFn        func(ctx context.Context) ([]string, error)
	pingFn              func(ctx context.Context) error
	batchInsertFn       func(ctx context.Context, balances []storage.TokenBalance) error
}

func (m *mockStore) GetBalances(ctx context.Context, wallet, symbol string, limit int) ([]storage.TokenBalance, error) {
	if m.getBalancesFn != nil {
		return m.getBalancesFn(ctx, wallet, symbol, limit)
	}
	return []storage.TokenBalance{}, nil
}

func (m *mockStore) GetWeeklyBalances(ctx context.Context, wallet string) ([]storage.WeeklyBalance, error) {
	if m.getWeeklyBalancesFn != nil {
		return m.getWeeklyBalancesFn(ctx, wallet)
	}
	return []storage.WeeklyBalance{}, nil
}

func (m *mockStore) GetWeeklyReport(ctx context.Context, wallet string, weeks int) ([]storage.WeeklyReport, error) {
	if m.getWeeklyReportFn != nil {
		return m.getWeeklyReportFn(ctx, wallet, weeks)
	}
	return []storage.WeeklyReport{}, nil
}

func (m *mockStore) GetWallets(ctx context.Context) ([]string, error) {
	if m.getWalletsFn != nil {
		return m.getWalletsFn(ctx)
	}
	return []string{}, nil
}

func (m *mockStore) BatchInsertBalances(ctx context.Context, balances []storage.TokenBalance) error {
	if m.batchInsertFn != nil {
		return m.batchInsertFn(ctx, balances)
	}
	return nil
}

func (m *mockStore) Ping(ctx context.Context) error {
	if m.pingFn != nil {
		return m.pingFn(ctx)
	}
	return nil
}

func (m *mockStore) Close() {}

// --- helpers ---

// newRouter builds a test router wired to the given mock store.
func newRouter(ms *mockStore) http.Handler {
	h := NewHandler(ms)
	return NewRouter(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, h)
}

func get(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v))
	return v
}

// --- sample fixtures ---

func sampleBalance() storage.TokenBalance {
	return storage.TokenBalance{
		ID:           1,
		QueriedAt:    time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC),
		Wallet:       "0xWALLET",
		TokenAddress: "0xTOKEN",
		Symbol:       "armmUSDC",
		Decimals:     6,
		Balance:      decimal.RequireFromString("10000"),
	}
}

func sampleWeeklyBalance() storage.WeeklyBalance {
	return storage.WeeklyBalance{
		Week:         time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		Wallet:       "0xWALLET",
		TokenAddress: "0xTOKEN",
		Symbol:       "armmUSDC",
		Decimals:     6,
		Balance:      decimal.RequireFromString("10000"),
		QueriedAt:    time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC),
	}
}

func sampleWeeklyReport() storage.WeeklyReport {
	return storage.WeeklyReport{
		Symbol:          "armmUSDC",
		TokenAddress:    "0xTOKEN",
		WeekStart:       time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		WeekEnd:         time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		CurrentBalance:  decimal.RequireFromString("10100"),
		PreviousBalance: decimal.RequireFromString("10000"),
		Change:          decimal.RequireFromString("100"),
		ChangePercent:   decimal.RequireFromString("1"),
		DailyAvgChange:  decimal.RequireFromString("14.285714"),
		APY:             decimal.RequireFromString("67.768"),
	}
}

// =============================================================================
// GetBalances
// =============================================================================

func TestGetBalances_ReturnsBalances(t *testing.T) {
	ms := &mockStore{
		getBalancesFn: func(_ context.Context, wallet, symbol string, limit int) ([]storage.TokenBalance, error) {
			assert.Equal(t, "0xABC", wallet)
			assert.Equal(t, "", symbol)
			assert.Equal(t, 100, limit)
			return []storage.TokenBalance{sampleBalance()}, nil
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/balances?wallet=0xABC")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Len(t, result, 1)
	assert.Equal(t, "armmUSDC", result[0]["symbol"])
	assert.Contains(t, result[0], "queried_at")
	assert.Contains(t, result[0], "token_address")
	assert.NotContains(t, result[0], "raw_balance")
}

func TestGetBalances_CustomLimit(t *testing.T) {
	var capturedLimit int
	ms := &mockStore{
		getBalancesFn: func(_ context.Context, _, _ string, limit int) ([]storage.TokenBalance, error) {
			capturedLimit = limit
			return []storage.TokenBalance{}, nil
		},
	}

	get(t, newRouter(ms), "/api/v1/balances?limit=25")
	assert.Equal(t, 25, capturedLimit)
}

func TestGetBalances_InvalidLimit_Returns400(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"non-integer", "?limit=abc"},
		{"zero", "?limit=0"},
		{"negative", "?limit=-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, newRouter(&mockStore{}), "/api/v1/balances"+tt.query)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestGetBalances_StoreError_Returns500(t *testing.T) {
	ms := &mockStore{
		getBalancesFn: func(_ context.Context, _, _ string, _ int) ([]storage.TokenBalance, error) {
			return nil, errors.New("db unavailable")
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/balances")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetBalances_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	ms := &mockStore{
		getBalancesFn: func(_ context.Context, _, _ string, _ int) ([]storage.TokenBalance, error) {
			return nil, nil // nil slice
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/balances")
	assert.Equal(t, http.StatusOK, rec.Code)

	result := decodeJSON[[]any](t, rec)
	assert.NotNil(t, result)
	assert.Len(t, result, 0, "nil slice must serialise as []")
}

func TestGetBalances_SymbolFilter(t *testing.T) {
	var capturedSymbol string
	ms := &mockStore{
		getBalancesFn: func(_ context.Context, _, symbol string, _ int) ([]storage.TokenBalance, error) {
			capturedSymbol = symbol
			return []storage.TokenBalance{}, nil
		},
	}

	get(t, newRouter(ms), "/api/v1/balances?symbol=armmXDAI")
	assert.Equal(t, "armmXDAI", capturedSymbol)
}

// =============================================================================
// GetWeeklyBalances
// =============================================================================

func TestGetWeeklyBalances_ReturnsBalances(t *testing.T) {
	ms := &mockStore{
		getWeeklyBalancesFn: func(_ context.Context, wallet string) ([]storage.WeeklyBalance, error) {
			assert.Equal(t, "0xWALLET", wallet)
			return []storage.WeeklyBalance{sampleWeeklyBalance()}, nil
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/balances/weekly")

	assert.Equal(t, http.StatusOK, rec.Code)
	result := decodeJSON[[]map[string]any](t, rec)
	require.Len(t, result, 1)
	assert.Contains(t, result[0], "week")
	assert.Contains(t, result[0], "wallet")
	assert.Contains(t, result[0], "token_address")
	assert.Contains(t, result[0], "queried_at")
}

func TestGetWeeklyBalances_StoreError_Returns500(t *testing.T) {
	ms := &mockStore{
		getWeeklyBalancesFn: func(_ context.Context, _ string) ([]storage.WeeklyBalance, error) {
			return nil, errors.New("timeout")
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/balances/weekly")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetWeeklyBalances_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	ms := &mockStore{
		getWeeklyBalancesFn: func(_ context.Context, _ string) ([]storage.WeeklyBalance, error) {
			return nil, nil
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/balances/weekly")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, decodeJSON[[]any](t, rec), 0)
}

// =============================================================================
// GetWeeklyReport
// =============================================================================

func TestGetWeeklyReport_DefaultWeeks_Returns200(t *testing.T) {
	var capturedWeeks int
	ms := &mockStore{
		getWeeklyReportFn: func(_ context.Context, wallet string, weeks int) ([]storage.WeeklyReport, error) {
			capturedWeeks = weeks
			assert.Equal(t, "0xWALLET", wallet)
			return []storage.WeeklyReport{sampleWeeklyReport()}, nil
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/report/weekly")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 2, capturedWeeks, "default weeks must be 2")

	result := decodeJSON[[]map[string]any](t, rec)
	require.Len(t, result, 1)
	r := result[0]
	assert.Contains(t, r, "symbol")
	assert.Contains(t, r, "token_address")
	assert.Contains(t, r, "week_start")
	assert.Contains(t, r, "week_end")
	assert.Contains(t, r, "current_balance")
	assert.Contains(t, r, "previous_balance")
	assert.Contains(t, r, "change")
	assert.Contains(t, r, "change_percent")
	assert.Contains(t, r, "daily_avg_change")
	assert.Contains(t, r, "apy")
}

func TestGetWeeklyReport_CustomWeeks_PassedToStore(t *testing.T) {
	var capturedWeeks int
	ms := &mockStore{
		getWeeklyReportFn: func(_ context.Context, _ string, weeks int) ([]storage.WeeklyReport, error) {
			capturedWeeks = weeks
			return []storage.WeeklyReport{}, nil
		},
	}

	get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/report/weekly?weeks=4")
	assert.Equal(t, 4, capturedWeeks)
}

func TestGetWeeklyReport_InvalidWeeks_Returns400(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"weeks=1 (below minimum)", "?weeks=1"},
		{"weeks=0", "?weeks=0"},
		{"weeks=53 (above max)", "?weeks=53"},
		{"weeks=abc (non-integer)", "?weeks=abc"},
		{"weeks=-1 (negative)", "?weeks=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, newRouter(&mockStore{}), "/api/v1/wallets/0xWALLET/report/weekly"+tt.query)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "expected 400 for %s", tt.query)
		})
	}
}

func TestGetWeeklyReport_BoundaryWeeks(t *testing.T) {
	ms := &mockStore{
		getWeeklyReportFn: func(_ context.Context, _ string, _ int) ([]storage.WeeklyReport, error) {
			return []storage.WeeklyReport{}, nil
		},
	}

	t.Run("weeks=2 (minimum)", func(t *testing.T) {
		rec := get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/report/weekly?weeks=2")
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("weeks=52 (maximum)", func(t *testing.T) {
		rec := get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/report/weekly?weeks=52")
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestGetWeeklyReport_StoreError_Returns500(t *testing.T) {
	ms := &mockStore{
		getWeeklyReportFn: func(_ context.Context, _ string, _ int) ([]storage.WeeklyReport, error) {
			return nil, errors.New("connection lost")
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/report/weekly")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetWeeklyReport_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	ms := &mockStore{
		getWeeklyReportFn: func(_ context.Context, _ string, _ int) ([]storage.WeeklyReport, error) {
			return nil, nil
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets/0xWALLET/report/weekly")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, decodeJSON[[]any](t, rec), 0)
}

// =============================================================================
// GetWallets
// =============================================================================

func TestGetWallets_ReturnsList(t *testing.T) {
	wallets := []string{"0xAAA", "0xBBB", "0xCCC"}
	ms := &mockStore{
		getWalletsFn: func(_ context.Context) ([]string, error) {
			return wallets, nil
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets")

	assert.Equal(t, http.StatusOK, rec.Code)
	result := decodeJSON[[]string](t, rec)
	assert.Equal(t, wallets, result)
}

func TestGetWallets_StoreError_Returns500(t *testing.T) {
	ms := &mockStore{
		getWalletsFn: func(_ context.Context) ([]string, error) {
			return nil, errors.New("db error")
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetWallets_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	ms := &mockStore{
		getWalletsFn: func(_ context.Context) ([]string, error) {
			return nil, nil
		},
	}

	rec := get(t, newRouter(ms), "/api/v1/wallets")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, decodeJSON[[]any](t, rec), 0)
}

// =============================================================================
// Content-Type
// =============================================================================

func TestAllEndpoints_ContentTypeJSON(t *testing.T) {
	ms := &mockStore{}
	router := newRouter(ms)

	endpoints := []string{
		"/api/v1/balances",
		"/api/v1/wallets",
		"/api/v1/wallets/0xWALLET/balances/weekly",
		"/api/v1/wallets/0xWALLET/report/weekly",
	}

	for _, path := range endpoints {
		t.Run(path, func(t *testing.T) {
			rec := get(t, router, path)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		})
	}
}
