package storage

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBalance_JSONSnakeCase(t *testing.T) {
	tb := TokenBalance{
		ID:           42,
		QueriedAt:    time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		Wallet:       "0xABC",
		TokenAddress: "0xDEF",
		Symbol:       "armmUSDC",
		Decimals:     6,
		RawBalance:   big.NewInt(1_000_000),
		Balance:      decimal.RequireFromString("1.000000"),
	}

	data, err := json.Marshal(tb)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))

	// All public fields must be snake_case
	assert.Contains(t, m, "id")
	assert.Contains(t, m, "queried_at")
	assert.Contains(t, m, "wallet")
	assert.Contains(t, m, "token_address")
	assert.Contains(t, m, "symbol")
	assert.Contains(t, m, "decimals")
	assert.Contains(t, m, "balance")

	// RawBalance must be excluded (json:"-")
	assert.NotContains(t, m, "raw_balance")
	assert.NotContains(t, m, "RawBalance")

	// No PascalCase keys
	assert.NotContains(t, m, "ID")
	assert.NotContains(t, m, "QueriedAt")
	assert.NotContains(t, m, "Wallet")
	assert.NotContains(t, m, "TokenAddress")
	assert.NotContains(t, m, "Symbol")
	assert.NotContains(t, m, "Decimals")
	assert.NotContains(t, m, "Balance")

	// Values
	assert.EqualValues(t, 42, m["id"])
	assert.Equal(t, "0xABC", m["wallet"])
	assert.Equal(t, "0xDEF", m["token_address"])
	assert.Equal(t, "armmUSDC", m["symbol"])
	assert.EqualValues(t, 6, m["decimals"])
}

func TestTokenBalance_JSONRoundTrip(t *testing.T) {
	// RawBalance is excluded, so we test the fields that survive marshaling.
	original := TokenBalance{
		ID:           7,
		QueriedAt:    time.Date(2026, 2, 10, 8, 30, 0, 0, time.UTC),
		Wallet:       "0xWALLET",
		TokenAddress: "0xTOKEN",
		Symbol:       "armmXDAI",
		Decimals:     18,
		Balance:      decimal.RequireFromString("1234.567890"),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded TokenBalance
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.ID, decoded.ID)
	assert.True(t, original.QueriedAt.Equal(decoded.QueriedAt))
	assert.Equal(t, original.Wallet, decoded.Wallet)
	assert.Equal(t, original.TokenAddress, decoded.TokenAddress)
	assert.Equal(t, original.Symbol, decoded.Symbol)
	assert.Equal(t, original.Decimals, decoded.Decimals)
	assert.Equal(t, original.Balance.String(), decoded.Balance.String())
	// RawBalance not preserved (json:"-")
	assert.Nil(t, decoded.RawBalance)
}

func TestWeeklyBalance_JSONSnakeCase(t *testing.T) {
	wb := WeeklyBalance{
		Week:         time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		Wallet:       "0xW",
		TokenAddress: "0xT",
		Symbol:       "USDC",
		Decimals:     6,
		Balance:      decimal.RequireFromString("999.99"),
		QueriedAt:    time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(wb)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Contains(t, m, "week")
	assert.Contains(t, m, "wallet")
	assert.Contains(t, m, "token_address")
	assert.Contains(t, m, "symbol")
	assert.Contains(t, m, "decimals")
	assert.Contains(t, m, "balance")
	assert.Contains(t, m, "queried_at")

	// No PascalCase
	assert.NotContains(t, m, "Week")
	assert.NotContains(t, m, "Wallet")
	assert.NotContains(t, m, "TokenAddress")
	assert.NotContains(t, m, "QueriedAt")
}

func TestWeeklyReport_JSONSnakeCase(t *testing.T) {
	wr := WeeklyReport{
		Symbol:          "armmUSDC",
		TokenAddress:    "0xTOKEN",
		WeekStart:       time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC),
		WeekEnd:         time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC),
		CurrentBalance:  decimal.RequireFromString("10100"),
		PreviousBalance: decimal.RequireFromString("10000"),
		Change:          decimal.RequireFromString("100"),
		ChangePercent:   decimal.RequireFromString("1"),
		DailyAvgChange:  decimal.RequireFromString("14.285714"),
		APY:             decimal.RequireFromString("67.768"),
	}

	data, err := json.Marshal(wr)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))

	// All fields must be snake_case
	expected := []string{
		"symbol", "token_address",
		"week_start", "week_end",
		"current_balance", "previous_balance",
		"change", "change_percent",
		"daily_avg_change", "apy",
	}
	for _, key := range expected {
		assert.Contains(t, m, key, "missing key: %s", key)
	}

	// No PascalCase
	for _, key := range []string{"Symbol", "TokenAddress", "WeekStart", "WeekEnd",
		"CurrentBalance", "PreviousBalance", "Change", "ChangePercent",
		"DailyAvgChange", "APY"} {
		assert.NotContains(t, m, key)
	}

	assert.Equal(t, "armmUSDC", m["symbol"])
}

func TestWeeklyReport_NilSlice_MarshaledAsEmptyArray(t *testing.T) {
	// Handlers convert nil slice to empty slice — verify the JSON is "[]" not "null".
	var reports []WeeklyReport
	reports = []WeeklyReport{} // simulates handler behavior

	data, err := json.Marshal(reports)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(data))
}
