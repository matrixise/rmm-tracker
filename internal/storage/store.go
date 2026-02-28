package storage

import (
	"context"
	"time"
)

// Commander is the write-side interface (used by the blockchain tracker).
type Commander interface {
	// BatchInsertBalances persists a batch of token balances and updates
	// tracker_metadata.last_run_at with the MAX queried_at from the batch.
	BatchInsertBalances(ctx context.Context, balances []TokenBalance) error
	// SetLastRunStatus records whether the last tracker run succeeded or failed.
	// last_run_at is managed by BatchInsertBalances; this only updates succeeded.
	SetLastRunStatus(ctx context.Context, succeeded bool) error
}

// Querier is the read-side interface (used by API, web UI).
type Querier interface {
	GetBalances(ctx context.Context, wallet, symbol string, limit int) ([]TokenBalance, error)
	GetLatestBalances(ctx context.Context, wallet string) ([]LatestBalance, error)
	GetDailyBalances(ctx context.Context, wallet string) ([]DailyBalance, error)
	GetDailyPeriodYield(ctx context.Context, wallet string, days int) ([]PeriodYield, error)
	GetDailyReport(ctx context.Context, wallet string, days int) ([]DailyReport, error)
	GetDashboardSummary(ctx context.Context) (DashboardSummary, error)
	GetWeeklyBalances(ctx context.Context, wallet string) ([]WeeklyBalance, error)
	GetWeeklyPeriodYield(ctx context.Context, wallet string, weeks int) ([]PeriodYield, error)
	GetWeeklyReport(ctx context.Context, wallet string, weeks int) ([]WeeklyReport, error)
	GetWallets(ctx context.Context) ([]string, error)
	GetLastRun(ctx context.Context) (time.Time, bool, error)
}

// Pinger is a connectivity probe interface (used by health checks).
type Pinger interface {
	Ping(ctx context.Context) error
}

// Storer composes Commander, Querier, and Pinger. It is the wiring point used
// in cmd/ and implemented by every storage backend.
type Storer interface {
	Commander
	Querier
	Pinger
	Close()
}
