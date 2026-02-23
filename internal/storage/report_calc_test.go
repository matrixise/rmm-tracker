package storage

import (
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// monday returns a Monday at 00:00 UTC for use as a week bucket.
func monday(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func dec(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

// assertDecimalApprox asserts two decimals are equal within epsilon.
func assertDecimalApprox(t *testing.T, expected, actual decimal.Decimal, epsilon string, msgAndArgs ...any) {
	t.Helper()
	diff := expected.Sub(actual).Abs()
	eps := dec(epsilon)
	assert.True(t, diff.LessThan(eps),
		"expected %s ≈ %s (diff %s) — %v", expected, actual, diff, msgAndArgs)
}

// assertDecEqual compares two decimals by their string representations, which
// normalises differences in internal precision (e.g. {100,-2} vs {1,0} both "1").
func assertDecEqual(t *testing.T, expected string, actual decimal.Decimal, msgAndArgs ...any) {
	t.Helper()
	assert.Equal(t, expected, actual.String(), msgAndArgs...)
}

// --- computeWeeklyReport ---

func TestComputeWeeklyReport_Empty(t *testing.T) {
	results := computeWeeklyReport(nil, nil)
	assert.Nil(t, results)
}

func TestComputeWeeklyReport_EmptySymbolOrder(t *testing.T) {
	results := computeWeeklyReport([]string{}, map[string][]weekEntry{})
	assert.Nil(t, results)
}

func TestComputeWeeklyReport_SingleEntry_NoHistory(t *testing.T) {
	// Only current week — no previous data.
	now := monday(2026, time.February, 23)
	bySymbol := map[string][]weekEntry{
		"USDC": {
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: now, balance: dec("1000")},
		},
	}

	results := computeWeeklyReport([]string{"USDC"}, bySymbol)

	require.Len(t, results, 1)
	r := results[0]
	assert.Equal(t, "USDC", r.Symbol)
	assertDecEqual(t, "1000", r.CurrentBalance)
	assert.True(t, r.PreviousBalance.IsZero(), "PreviousBalance must be zero")
	assert.True(t, r.Change.IsZero(), "Change must be zero")
	assert.True(t, r.ChangePercent.IsZero(), "ChangePercent must be zero")
	assert.True(t, r.DailyAvgChange.IsZero(), "DailyAvgChange must be zero")
	assert.True(t, r.APY.IsZero(), "APY must be zero")
	// With single entry: weekStart == weekEnd - 7d
	assert.Equal(t, now, r.WeekStart)
	assert.Equal(t, now.Add(7*24*time.Hour), r.WeekEnd)
}

func TestComputeWeeklyReport_TwoWeeks_NormalGrowth(t *testing.T) {
	current  := monday(2026, time.February, 23)
	previous := monday(2026, time.February, 16)

	bySymbol := map[string][]weekEntry{
		"USDC": {
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: current, balance: dec("10100")},
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: previous, balance: dec("10000")},
		},
	}

	results := computeWeeklyReport([]string{"USDC"}, bySymbol)

	require.Len(t, results, 1)
	r := results[0]

	// Balances
	assertDecEqual(t, "10100", r.CurrentBalance)
	assertDecEqual(t, "10000", r.PreviousBalance)
	assertDecEqual(t, "100", r.Change)

	// change_percent = 100/10000 * 100 = 1%
	assertDecEqual(t, "1", r.ChangePercent)

	// actual_days = 7
	// daily_avg = 100/7 ≈ 14.285...
	expectedDailyAvg := dec("100").Div(dec("7"))
	assertDecimalApprox(t, expectedDailyAvg, r.DailyAvgChange, "0.000001")

	// APY = (1.01)^(365/7) - 1
	expectedAPY := decimal.NewFromFloat((math.Pow(1.01, 365.0/7.0) - 1) * 100)
	assertDecimalApprox(t, expectedAPY, r.APY, "0.001", "APY formula")

	// week_start = oldest bucket (Feb 16), week_end = newest bucket + 7d (Mar 2)
	assert.Equal(t, previous, r.WeekStart)
	assert.Equal(t, current.Add(7*24*time.Hour), r.WeekEnd)
}

func TestComputeWeeklyReport_FourWeeks_ActualDays21(t *testing.T) {
	// weeks=4 requested but actualDays must come from data, not theoretical value.
	w0 := monday(2026, time.February, 23) // current
	w1 := monday(2026, time.February, 16)
	w2 := monday(2026, time.February, 9)
	w3 := monday(2026, time.February, 2) // oldest

	bySymbol := map[string][]weekEntry{
		"XDAI": {
			{symbol: "XDAI", tokenAddress: "0xXDAI", weekBucket: w0, balance: dec("2100")},
			{symbol: "XDAI", tokenAddress: "0xXDAI", weekBucket: w1, balance: dec("2070")},
			{symbol: "XDAI", tokenAddress: "0xXDAI", weekBucket: w2, balance: dec("2040")},
			{symbol: "XDAI", tokenAddress: "0xXDAI", weekBucket: w3, balance: dec("2000")},
		},
	}

	results := computeWeeklyReport([]string{"XDAI"}, bySymbol)

	require.Len(t, results, 1)
	r := results[0]

	// change = 2100 - 2000 = 100
	assertDecEqual(t, "100", r.Change)

	// actualDays = w0 - w3 = 21 days
	// daily_avg = 100/21
	expectedDailyAvg := dec("100").Div(dec("21"))
	assertDecimalApprox(t, expectedDailyAvg, r.DailyAvgChange, "0.000001", "daily avg over 21 days")

	// APY = (1 + 100/2000)^(365/21) - 1
	ratio := 1 + 100.0/2000.0
	expectedAPY := decimal.NewFromFloat((math.Pow(ratio, 365.0/21.0) - 1) * 100)
	assertDecimalApprox(t, expectedAPY, r.APY, "0.001", "APY over 21 days")

	// week_start = w3 (oldest), week_end = w0 + 7d
	assert.Equal(t, w3, r.WeekStart)
	assert.Equal(t, w0.Add(7*24*time.Hour), r.WeekEnd)
}

func TestComputeWeeklyReport_FewerWeeksThanRequested(t *testing.T) {
	// weeks=4 requested but only 2 weeks in DB — actualDays must be 7, not 21.
	current  := monday(2026, time.February, 23)
	previous := monday(2026, time.February, 16)

	bySymbol := map[string][]weekEntry{
		"USDC": {
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: current, balance: dec("500")},
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: previous, balance: dec("490")},
		},
	}

	results := computeWeeklyReport([]string{"USDC"}, bySymbol)
	require.Len(t, results, 1)
	r := results[0]

	// actualDays = 7 (actual data span), not 21 (theoretical weeks=4)
	expectedDailyAvg := dec("10").Div(dec("7"))
	assertDecimalApprox(t, expectedDailyAvg, r.DailyAvgChange, "0.000001", "dailyAvg uses actual days")
}

func TestComputeWeeklyReport_NegativeChange(t *testing.T) {
	current  := monday(2026, time.February, 23)
	previous := monday(2026, time.February, 16)

	bySymbol := map[string][]weekEntry{
		"USDC": {
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: current, balance: dec("9500")},
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: previous, balance: dec("10000")},
		},
	}

	results := computeWeeklyReport([]string{"USDC"}, bySymbol)
	require.Len(t, results, 1)
	r := results[0]

	assertDecEqual(t, "-500", r.Change)
	// change_percent = -500/10000 * 100 = -5
	assertDecEqual(t, "-5", r.ChangePercent)
	// ratio = 1 + (-500/10000) = 0.95 > 0 → APY computed, should be negative
	assert.True(t, r.APY.IsNegative(), "APY should be negative for a loss")
	// daily_avg = -500/7 < 0
	assert.True(t, r.DailyAvgChange.IsNegative())
}

func TestComputeWeeklyReport_APY_NegativeRatioGuard(t *testing.T) {
	// Balance drops by more than 100% → ratio <= 0 → APY must be zero (NaN guard).
	current  := monday(2026, time.February, 23)
	previous := monday(2026, time.February, 16)

	bySymbol := map[string][]weekEntry{
		"DEBT": {
			{symbol: "DEBT", tokenAddress: "0xDEBT", weekBucket: current, balance: dec("0")},
			{symbol: "DEBT", tokenAddress: "0xDEBT", weekBucket: previous, balance: dec("1000")},
		},
	}

	results := computeWeeklyReport([]string{"DEBT"}, bySymbol)
	require.Len(t, results, 1)
	r := results[0]

	// ratio = 1 + (-1000/1000) = 0 → guard triggers → APY = 0
	assert.True(t, r.APY.IsZero(), "APY must be zero when ratio <= 0")
}

func TestComputeWeeklyReport_ZeroPreviousBalance(t *testing.T) {
	// previous = 0: no APY, no change_percent, no daily_avg (actualDays = 0 not relevant,
	// but previous.IsZero() guards APY and changePercent).
	current  := monday(2026, time.February, 23)
	previous := monday(2026, time.February, 16)

	bySymbol := map[string][]weekEntry{
		"XDAI": {
			{symbol: "XDAI", tokenAddress: "0xXDAI", weekBucket: current, balance: dec("100")},
			{symbol: "XDAI", tokenAddress: "0xXDAI", weekBucket: previous, balance: dec("0")},
		},
	}

	results := computeWeeklyReport([]string{"XDAI"}, bySymbol)
	require.Len(t, results, 1)
	r := results[0]

	assert.True(t, r.ChangePercent.IsZero(), "no change_percent when previous is zero")
	assert.True(t, r.APY.IsZero(), "no APY when previous is zero")
	// dailyAvg is still computed (actualDays > 0)
	assert.True(t, r.DailyAvgChange.IsPositive())
}

func TestComputeWeeklyReport_MultipleSymbols_PreservesOrder(t *testing.T) {
	w0 := monday(2026, time.February, 23)
	w1 := monday(2026, time.February, 16)

	bySymbol := map[string][]weekEntry{
		"USDC": {
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: w0, balance: dec("1010")},
			{symbol: "USDC", tokenAddress: "0xUSDC", weekBucket: w1, balance: dec("1000")},
		},
		"XDAI": {
			{symbol: "XDAI", tokenAddress: "0xXDAI", weekBucket: w0, balance: dec("2020")},
			{symbol: "XDAI", tokenAddress: "0xXDAI", weekBucket: w1, balance: dec("2000")},
		},
	}
	// symbolOrder simulates the DB query order
	symbolOrder := []string{"USDC", "XDAI"}

	results := computeWeeklyReport(symbolOrder, bySymbol)

	require.Len(t, results, 2)
	assert.Equal(t, "USDC", results[0].Symbol)
	assert.Equal(t, "XDAI", results[1].Symbol)

	assertDecEqual(t, "10", results[0].Change)
	assertDecEqual(t, "20", results[1].Change)
}

func TestComputeWeeklyReport_APY_Positive_LargeGrowth(t *testing.T) {
	// +10% weekly → very high APY (~14000%)
	w0 := monday(2026, time.February, 23)
	w1 := monday(2026, time.February, 16)

	bySymbol := map[string][]weekEntry{
		"TOKEN": {
			{symbol: "TOKEN", tokenAddress: "0xT", weekBucket: w0, balance: dec("11000")},
			{symbol: "TOKEN", tokenAddress: "0xT", weekBucket: w1, balance: dec("10000")},
		},
	}

	results := computeWeeklyReport([]string{"TOKEN"}, bySymbol)
	require.Len(t, results, 1)
	r := results[0]

	assert.True(t, r.APY.IsPositive())
	// APY = (1.10)^(365/7) - 1 ≈ 141.8 × 100 = 14180%
	expectedAPY := decimal.NewFromFloat((math.Pow(1.10, 365.0/7.0) - 1) * 100)
	assertDecimalApprox(t, expectedAPY, r.APY, "1.0", "APY for 10% weekly gain")
}

func TestComputeWeeklyReport_WeekStartEnd_TwoWeeks(t *testing.T) {
	// week_start = oldest bucket, week_end = newest bucket + 7 days
	feb16 := monday(2026, time.February, 16)
	feb23 := monday(2026, time.February, 23)

	bySymbol := map[string][]weekEntry{
		"X": {
			{symbol: "X", weekBucket: feb23, balance: dec("100")},
			{symbol: "X", weekBucket: feb16, balance: dec("90")},
		},
	}

	results := computeWeeklyReport([]string{"X"}, bySymbol)
	require.Len(t, results, 1)

	assert.Equal(t, feb16, results[0].WeekStart, "week_start = oldest bucket")
	assert.Equal(t, feb23.Add(7*24*time.Hour), results[0].WeekEnd, "week_end = newest + 7d")
}

func TestComputeWeeklyReport_SkipsEmptyEntries(t *testing.T) {
	bySymbol := map[string][]weekEntry{
		"EMPTY": {},
		"REAL": {
			{symbol: "REAL", weekBucket: monday(2026, 2, 23), balance: dec("50")},
		},
	}

	results := computeWeeklyReport([]string{"EMPTY", "REAL"}, bySymbol)
	require.Len(t, results, 1)
	assert.Equal(t, "REAL", results[0].Symbol)
}
