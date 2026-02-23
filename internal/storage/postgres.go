package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	shop "github.com/jackc/pgx-shopspring-decimal"
	"github.com/shopspring/decimal"
)

// Store manages PostgreSQL operations
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new PostgreSQL store with connection pooling
func NewStore(ctx context.Context, dsn string) (*Store, error) {
	// Parse and configure connection pool
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Tune connection pool
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// Register decimal.Decimal type mapping
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		shop.Register(conn.TypeMap())
		return nil
	}

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return &Store{pool: pool}, nil
}

// Close closes the connection pool
func (s *Store) Close() {
	s.pool.Close()
}

// BatchInsertBalances inserts multiple token balances using pgx.Batch
func (s *Store) BatchInsertBalances(ctx context.Context, balances []TokenBalance) error {
	if len(balances) == 0 {
		return nil
	}

	// Use pgx.Batch for optimal performance
	batch := &pgx.Batch{}

	for _, bal := range balances {
		batch.Queue(`
			INSERT INTO token_balances
			(queried_at, wallet, token_address, symbol, decimals, raw_balance, balance)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			bal.QueriedAt,
			bal.Wallet,
			bal.TokenAddress,
			bal.Symbol,
			bal.Decimals,
			bal.RawBalance.String(),
			bal.Balance,
		)
	}

	// Execute batch
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	// Check for errors
	for range balances {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch insert failed: %w", err)
		}
	}

	return nil
}

// Ping verifies the connection is alive
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// GetBalances returns token balances with optional filters on wallet and symbol.
func (s *Store) GetBalances(ctx context.Context, wallet, symbol string, limit int) ([]TokenBalance, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, queried_at, wallet, token_address, symbol, decimals, balance
		FROM token_balances
		WHERE ($1 = '' OR wallet = $1)
		  AND ($2 = '' OR symbol = $2)
		ORDER BY queried_at DESC
		LIMIT $3`,
		wallet, symbol, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var balances []TokenBalance
	for rows.Next() {
		var b TokenBalance
		if err := rows.Scan(&b.ID, &b.QueriedAt, &b.Wallet, &b.TokenAddress, &b.Symbol, &b.Decimals, &b.Balance); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		balances = append(balances, b)
	}

	return balances, rows.Err()
}

// GetWeeklyBalances returns the last recorded balance per (week, symbol) for a wallet,
// ordered by week descending.
func (s *Store) GetWeeklyBalances(ctx context.Context, wallet string) ([]WeeklyBalance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (DATE_TRUNC('week', queried_at), symbol)
			DATE_TRUNC('week', queried_at) AS week,
			wallet,
			token_address,
			symbol,
			decimals,
			balance,
			queried_at
		FROM token_balances
		WHERE wallet = $1
		ORDER BY DATE_TRUNC('week', queried_at) DESC, symbol, queried_at DESC`,
		wallet,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []WeeklyBalance
	for rows.Next() {
		var b WeeklyBalance
		if err := rows.Scan(&b.Week, &b.Wallet, &b.TokenAddress, &b.Symbol, &b.Decimals, &b.Balance, &b.QueriedAt); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, b)
	}

	return results, rows.Err()
}

// GetWeeklyReport returns per-token balance comparison between current and previous week for a wallet.
func (s *Store) GetWeeklyReport(ctx context.Context, wallet string) ([]WeeklyReport, error) {
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
			SELECT DISTINCT ON (DATE_TRUNC('week', queried_at), symbol)
				DATE_TRUNC('week', queried_at) AS week_bucket,
				symbol, token_address, balance
			FROM token_balances
			WHERE wallet = $1
			ORDER BY DATE_TRUNC('week', queried_at) DESC, symbol, queried_at DESC
		),
		recent_weeks AS (
			SELECT week_bucket FROM ranked
			GROUP BY week_bucket
			ORDER BY week_bucket DESC
			LIMIT 2
		)
		SELECT r.symbol, r.token_address, r.week_bucket, r.balance
		FROM ranked r
		WHERE r.week_bucket IN (SELECT week_bucket FROM recent_weeks)
		ORDER BY r.symbol, r.week_bucket DESC`,
		wallet,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Group rows by symbol: first row = current week, second = previous week
	type row struct {
		symbol       string
		tokenAddress string
		weekBucket   time.Time
		balance      decimal.Decimal
	}

	bySymbol := make(map[string][]row)
	symbolOrder := []string{}

	for rows.Next() {
		var r row
		if err := rows.Scan(&r.symbol, &r.tokenAddress, &r.weekBucket, &r.balance); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if _, seen := bySymbol[r.symbol]; !seen {
			symbolOrder = append(symbolOrder, r.symbol)
		}
		bySymbol[r.symbol] = append(bySymbol[r.symbol], r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	seven := decimal.NewFromInt(7)
	hundred := decimal.NewFromInt(100)

	var results []WeeklyReport
	for _, sym := range symbolOrder {
		entries := bySymbol[sym]
		if len(entries) == 0 {
			continue
		}
		current := entries[0].balance
		var previous decimal.Decimal
		if len(entries) >= 2 {
			previous = entries[1].balance
		}

		change := current.Sub(previous)

		var changePercent decimal.Decimal
		if !previous.IsZero() {
			changePercent = change.Div(previous).Mul(hundred)
		}

		dailyAvg := change.Div(seven)

		results = append(results, WeeklyReport{
			Symbol:          sym,
			TokenAddress:    entries[0].tokenAddress,
			CurrentBalance:  current,
			PreviousBalance: previous,
			Change:          change,
			ChangePercent:   changePercent,
			DailyAvgChange:  dailyAvg,
		})
	}

	return results, nil
}

// GetWallets returns distinct wallet addresses stored in the database.
func (s *Store) GetWallets(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT wallet FROM token_balances ORDER BY wallet`)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var wallets []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		wallets = append(wallets, w)
	}

	return wallets, rows.Err()
}
