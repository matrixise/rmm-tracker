package storage

import "context"

// Storer is the interface implemented by Store.
// It is exposed for dependency injection and mocking in tests.
type Storer interface {
	BatchInsertBalances(ctx context.Context, balances []TokenBalance) error
	GetBalances(ctx context.Context, wallet, symbol string, limit int) ([]TokenBalance, error)
	GetWeeklyBalances(ctx context.Context, wallet string) ([]WeeklyBalance, error)
	GetWeeklyReport(ctx context.Context, wallet string, weeks int) ([]WeeklyReport, error)
	GetWallets(ctx context.Context) ([]string, error)
	Ping(ctx context.Context) error
	Close()
}
