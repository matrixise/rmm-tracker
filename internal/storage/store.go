package storage

import "context"

// Storer is the interface implemented by Store.
// It is exposed for dependency injection and mocking in tests.
type Storer interface {
	BatchInsertBalances(ctx context.Context, balances []TokenBalance) error
	GetBalances(ctx context.Context, wallet, symbol string, limit int) ([]TokenBalance, error)
	GetDailyBalances(ctx context.Context, wallet string) ([]DailyBalance, error)
	GetDailyPeriodYield(ctx context.Context, wallet string, days int) ([]PeriodYield, error)
	GetDailyReport(ctx context.Context, wallet string, days int) ([]DailyReport, error)
	GetWeeklyBalances(ctx context.Context, wallet string) ([]WeeklyBalance, error)
	GetWeeklyPeriodYield(ctx context.Context, wallet string, weeks int) ([]PeriodYield, error)
	GetWeeklyReport(ctx context.Context, wallet string, weeks int) ([]WeeklyReport, error)
	GetWallets(ctx context.Context) ([]string, error)
	Ping(ctx context.Context) error
	Close()
}
