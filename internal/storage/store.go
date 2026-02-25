package storage

import "context"

// Commander is the write-side interface (used by the blockchain tracker).
type Commander interface {
	BatchInsertBalances(ctx context.Context, balances []TokenBalance) error
}

// Querier is the read-side interface (used by API, web UI).
type Querier interface {
	GetBalances(ctx context.Context, wallet, symbol string, limit int) ([]TokenBalance, error)
	GetDailyBalances(ctx context.Context, wallet string) ([]DailyBalance, error)
	GetDailyReport(ctx context.Context, wallet string, days int) ([]DailyReport, error)
	GetWeeklyBalances(ctx context.Context, wallet string) ([]WeeklyBalance, error)
	GetWeeklyReport(ctx context.Context, wallet string, weeks int) ([]WeeklyReport, error)
	GetWallets(ctx context.Context) ([]string, error)
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
