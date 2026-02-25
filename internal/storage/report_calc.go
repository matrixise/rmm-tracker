package storage

import (
	"math"
	"time"

	"github.com/shopspring/decimal"
)

// dayEntry holds one data point for a token returned by the daily CTE query.
// Rows are ordered day_bucket DESC (newest first) within each symbol group.
type dayEntry struct {
	symbol       string
	tokenAddress string
	dayBucket    time.Time
	balance      decimal.Decimal
}

// computeDailyReport builds a DailyReport for each consecutive (day, day-1) pair per symbol.
// bySymbol maps symbol → []dayEntry ordered day_bucket DESC (entries[0] = most recent day).
// symbolOrder preserves the original query ordering.
func computeDailyReport(symbolOrder []string, bySymbol map[string][]dayEntry) []DailyReport {
	hundred := decimal.NewFromInt(100)
	one := decimal.NewFromInt(1)

	var results []DailyReport
	for _, sym := range symbolOrder {
		entries := bySymbol[sym]
		if len(entries) < 2 {
			continue
		}
		for i := 0; i < len(entries)-1; i++ {
			current := entries[i].balance
			previous := entries[i+1].balance

			change := current.Sub(previous)

			var changePercent decimal.Decimal
			if !previous.IsZero() {
				changePercent = change.Div(previous).Mul(hundred)
			}

			// APY = (1 + dailyRate)^365 - 1
			var apy decimal.Decimal
			if !previous.IsZero() {
				ratio, _ := one.Add(change.Div(previous)).Float64()
				if ratio > 0 {
					v := math.Pow(ratio, 365) - 1
					if !math.IsInf(v, 0) && !math.IsNaN(v) {
						apy = decimal.NewFromFloat(v).Mul(hundred)
					}
				}
			}

			results = append(results, DailyReport{
				Symbol:          sym,
				TokenAddress:    entries[i].tokenAddress,
				Day:             entries[i].dayBucket,
				CurrentBalance:  current,
				PreviousBalance: previous,
				Change:          change,
				ChangePercent:   changePercent,
				APY:             apy,
			})
		}
	}
	return results
}

// computeDailyPeriodYield computes the total yield per token over the full set of day buckets.
// bySymbol maps symbol → []dayEntry ordered day_bucket DESC (entries[0] = most recent).
// The yield spans from the oldest bucket's balance to the most recent bucket's balance.
func computeDailyPeriodYield(symbolOrder []string, bySymbol map[string][]dayEntry) []PeriodYield {
	hundred := decimal.NewFromInt(100)
	var results []PeriodYield
	for _, sym := range symbolOrder {
		entries := bySymbol[sym]
		if len(entries) < 2 {
			continue
		}
		newest := entries[0]
		oldest := entries[len(entries)-1]
		end := newest.balance
		start := oldest.balance
		change := end.Sub(start)
		var changePct decimal.Decimal
		if !start.IsZero() {
			changePct = change.Div(start).Mul(hundred)
		}
		results = append(results, PeriodYield{
			Symbol:        sym,
			TokenAddress:  newest.tokenAddress,
			FromDate:      oldest.dayBucket,
			ToDate:        newest.dayBucket,
			StartBalance:  start,
			EndBalance:    end,
			Change:        change,
			ChangePercent: changePct,
		})
	}
	return results
}

// weekEntry holds one data point for a token returned by the weekly CTE query.
// Rows are ordered week_bucket DESC (newest first) within each symbol group.
type weekEntry struct {
	symbol       string
	tokenAddress string
	weekBucket   time.Time
	balance      decimal.Decimal
}

// computeWeeklyPeriodYield computes the total yield per token over the full set of week buckets.
// bySymbol maps symbol → []weekEntry ordered week_bucket DESC (entries[0] = most recent).
// The yield spans from the oldest bucket's balance to the most recent bucket's balance.
func computeWeeklyPeriodYield(symbolOrder []string, bySymbol map[string][]weekEntry) []PeriodYield {
	hundred := decimal.NewFromInt(100)
	var results []PeriodYield
	for _, sym := range symbolOrder {
		entries := bySymbol[sym]
		if len(entries) < 2 {
			continue
		}
		newest := entries[0]
		oldest := entries[len(entries)-1]
		end := newest.balance
		start := oldest.balance
		change := end.Sub(start)
		var changePct decimal.Decimal
		if !start.IsZero() {
			changePct = change.Div(start).Mul(hundred)
		}
		results = append(results, PeriodYield{
			Symbol:        sym,
			TokenAddress:  newest.tokenAddress,
			FromDate:      oldest.weekBucket,
			ToDate:        newest.weekBucket.Add(7 * 24 * time.Hour),
			StartBalance:  start,
			EndBalance:    end,
			Change:        change,
			ChangePercent: changePct,
		})
	}
	return results
}

// computeWeeklyReport builds a WeeklyReport for each symbol from pre-grouped rows.
// bySymbol maps symbol → []weekEntry ordered week_bucket DESC (entries[0] = current week).
// symbolOrder preserves the original query ordering.
func computeWeeklyReport(symbolOrder []string, bySymbol map[string][]weekEntry) []WeeklyReport {
	hundred := decimal.NewFromInt(100)
	one := decimal.NewFromInt(1)
	daysPerYear := decimal.NewFromInt(365)

	var results []WeeklyReport
	for _, sym := range symbolOrder {
		entries := bySymbol[sym]
		if len(entries) == 0 {
			continue
		}

		current := entries[0].balance

		// Single entry — no previous week to compare against.
		// Return only the current balance; all change/growth metrics stay zero.
		if len(entries) == 1 {
			results = append(results, WeeklyReport{
				Symbol:         sym,
				TokenAddress:   entries[0].tokenAddress,
				WeekStart:      entries[0].weekBucket,
				WeekEnd:        entries[0].weekBucket.Add(7 * 24 * time.Hour),
				CurrentBalance: current,
			})
			continue
		}

		previous := entries[len(entries)-1].balance

		change := current.Sub(previous)

		var changePercent decimal.Decimal
		if !previous.IsZero() {
			changePercent = change.Div(previous).Mul(hundred)
		}

		// Use the actual elapsed days between the oldest and newest buckets
		// rather than a theoretical value — handles the case where fewer weeks
		// exist in the DB than requested.
		d := entries[0].weekBucket.Sub(entries[len(entries)-1].weekBucket).Hours() / 24
		actualDays := decimal.NewFromFloat(d)

		var dailyAvg decimal.Decimal
		if actualDays.IsPositive() {
			dailyAvg = change.Div(actualDays)
		}

		// APY = (1 + change/previous)^(365/actualDays) - 1
		// Guard: math.Pow(ratio, non-integer) returns NaN when ratio <= 0.
		var apy decimal.Decimal
		if !previous.IsZero() && actualDays.IsPositive() {
			ratio, _ := one.Add(change.Div(previous)).Float64()
			if ratio > 0 {
				exponent, _ := daysPerYear.Div(actualDays).Float64()
				v := math.Pow(ratio, exponent) - 1
				if !math.IsInf(v, 0) && !math.IsNaN(v) {
					apy = decimal.NewFromFloat(v).Mul(hundred)
				}
			}
		}

		// week_start = beginning of the oldest week in the comparison period.
		// week_end   = end of the current (most recent) week = newest bucket + 7 days.
		weekStart := entries[len(entries)-1].weekBucket
		weekEnd := entries[0].weekBucket.Add(7 * 24 * time.Hour)

		results = append(results, WeeklyReport{
			Symbol:          sym,
			TokenAddress:    entries[0].tokenAddress,
			WeekStart:       weekStart,
			WeekEnd:         weekEnd,
			CurrentBalance:  current,
			PreviousBalance: previous,
			Change:          change,
			ChangePercent:   changePercent,
			DailyAvgChange:  dailyAvg,
			APY:             apy,
		})
	}

	return results
}
