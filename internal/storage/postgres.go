package storage

import (
	"context"
	"fmt"
	"time"

	shop "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	defer func() { _ = br.Close() }()

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

// GetDailyBalances returns the last recorded balance per (day, symbol) for a wallet,
// ordered by day descending.
func (s *Store) GetDailyBalances(ctx context.Context, wallet string) ([]DailyBalance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (day_bucket, symbol)
			day_bucket AS day,
			wallet,
			token_address,
			symbol,
			decimals,
			balance,
			queried_at
		FROM token_balances
		WHERE wallet = $1
		ORDER BY day_bucket DESC, symbol, queried_at DESC`,
		wallet,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []DailyBalance
	for rows.Next() {
		var b DailyBalance
		if err := rows.Scan(&b.Day, &b.Wallet, &b.TokenAddress, &b.Symbol, &b.Decimals, &b.Balance, &b.QueriedAt); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, b)
	}
	return results, rows.Err()
}

// GetDailyReport returns per-token day-over-day balance comparisons for a wallet.
// days must be >= 2 and <= 365.
func (s *Store) GetDailyReport(ctx context.Context, wallet string, days int) ([]DailyReport, error) {
	if days < 2 {
		return nil, fmt.Errorf("days must be >= 2")
	}
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
			SELECT DISTINCT ON (day_bucket, symbol)
				day_bucket,
				symbol, token_address, balance
			FROM token_balances
			WHERE wallet = $1
			ORDER BY day_bucket DESC, symbol, queried_at DESC
		),
		recent_days AS (
			SELECT day_bucket FROM ranked
			GROUP BY day_bucket
			ORDER BY day_bucket DESC
			LIMIT $2
		)
		SELECT r.symbol, r.token_address, r.day_bucket, r.balance
		FROM ranked r
		WHERE r.day_bucket IN (SELECT day_bucket FROM recent_days)
		ORDER BY r.symbol, r.day_bucket DESC`,
		wallet, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	bySymbol := make(map[string][]dayEntry)
	symbolOrder := []string{}
	for rows.Next() {
		var e dayEntry
		if err := rows.Scan(&e.symbol, &e.tokenAddress, &e.dayBucket, &e.balance); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if _, seen := bySymbol[e.symbol]; !seen {
			symbolOrder = append(symbolOrder, e.symbol)
		}
		bySymbol[e.symbol] = append(bySymbol[e.symbol], e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return computeDailyReport(symbolOrder, bySymbol), nil
}

// GetDailyPeriodYield returns the total yield per token over the last N day buckets for a wallet.
// days must be >= 2 and <= 365.
func (s *Store) GetDailyPeriodYield(ctx context.Context, wallet string, days int) ([]PeriodYield, error) {
	if days < 2 {
		return nil, fmt.Errorf("days must be >= 2")
	}
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
			SELECT DISTINCT ON (day_bucket, symbol)
				day_bucket,
				symbol, token_address, balance
			FROM token_balances
			WHERE wallet = $1
			ORDER BY day_bucket DESC, symbol, queried_at DESC
		),
		recent_days AS (
			SELECT day_bucket FROM ranked
			GROUP BY day_bucket
			ORDER BY day_bucket DESC
			LIMIT $2
		)
		SELECT r.symbol, r.token_address, r.day_bucket, r.balance
		FROM ranked r
		WHERE r.day_bucket IN (SELECT day_bucket FROM recent_days)
		ORDER BY r.symbol, r.day_bucket DESC`,
		wallet, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	bySymbol := make(map[string][]dayEntry)
	symbolOrder := []string{}
	for rows.Next() {
		var e dayEntry
		if err := rows.Scan(&e.symbol, &e.tokenAddress, &e.dayBucket, &e.balance); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if _, seen := bySymbol[e.symbol]; !seen {
			symbolOrder = append(symbolOrder, e.symbol)
		}
		bySymbol[e.symbol] = append(bySymbol[e.symbol], e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return computeDailyPeriodYield(symbolOrder, bySymbol), nil
}

// GetWeeklyPeriodYield returns the total yield per token over the last N week buckets for a wallet.
// weeks must be >= 2 and <= 52.
func (s *Store) GetWeeklyPeriodYield(ctx context.Context, wallet string, weeks int) ([]PeriodYield, error) {
	if weeks < 2 {
		return nil, fmt.Errorf("weeks must be >= 2")
	}
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
			SELECT DISTINCT ON (week_bucket, symbol)
				week_bucket,
				symbol, token_address, balance
			FROM token_balances
			WHERE wallet = $1
			ORDER BY week_bucket DESC, symbol, queried_at DESC
		),
		recent_weeks AS (
			SELECT week_bucket FROM ranked
			GROUP BY week_bucket
			ORDER BY week_bucket DESC
			LIMIT $2
		)
		SELECT r.symbol, r.token_address, r.week_bucket, r.balance
		FROM ranked r
		WHERE r.week_bucket IN (SELECT week_bucket FROM recent_weeks)
		ORDER BY r.symbol, r.week_bucket DESC`,
		wallet, weeks,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	bySymbol := make(map[string][]weekEntry)
	symbolOrder := []string{}
	for rows.Next() {
		var e weekEntry
		if err := rows.Scan(&e.symbol, &e.tokenAddress, &e.weekBucket, &e.balance); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if _, seen := bySymbol[e.symbol]; !seen {
			symbolOrder = append(symbolOrder, e.symbol)
		}
		bySymbol[e.symbol] = append(bySymbol[e.symbol], e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return computeWeeklyPeriodYield(symbolOrder, bySymbol), nil
}

// GetWeeklyBalances returns the last recorded balance per (week, symbol) for a wallet,
// ordered by week descending.
// Uses the stored week_bucket column + idx_token_balances_wallet_wbucket_symbol to avoid
// a full sort on DATE_TRUNC.
func (s *Store) GetWeeklyBalances(ctx context.Context, wallet string) ([]WeeklyBalance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (week_bucket, symbol)
			week_bucket AS week,
			wallet,
			token_address,
			symbol,
			decimals,
			balance,
			queried_at
		FROM token_balances
		WHERE wallet = $1
		ORDER BY week_bucket DESC, symbol, queried_at DESC`,
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

// GetWeeklyReport returns per-token balance comparison between current and N-1 previous weeks for a wallet.
// weeks must be >= 2 and <= 52.
func (s *Store) GetWeeklyReport(ctx context.Context, wallet string, weeks int) ([]WeeklyReport, error) {
	if weeks < 2 {
		return nil, fmt.Errorf("weeks must be >= 2")
	}
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
			SELECT DISTINCT ON (week_bucket, symbol)
				week_bucket,
				symbol, token_address, balance
			FROM token_balances
			WHERE wallet = $1
			ORDER BY week_bucket DESC, symbol, queried_at DESC
		),
		recent_weeks AS (
			SELECT week_bucket FROM ranked
			GROUP BY week_bucket
			ORDER BY week_bucket DESC
			LIMIT $2
		)
		SELECT r.symbol, r.token_address, r.week_bucket, r.balance
		FROM ranked r
		WHERE r.week_bucket IN (SELECT week_bucket FROM recent_weeks)
		ORDER BY r.symbol, r.week_bucket DESC`,
		wallet, weeks,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Group rows by symbol: first row = current week, last = oldest week
	bySymbol := make(map[string][]weekEntry)
	symbolOrder := []string{}

	for rows.Next() {
		var e weekEntry
		if err := rows.Scan(&e.symbol, &e.tokenAddress, &e.weekBucket, &e.balance); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if _, seen := bySymbol[e.symbol]; !seen {
			symbolOrder = append(symbolOrder, e.symbol)
		}
		bySymbol[e.symbol] = append(bySymbol[e.symbol], e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return computeWeeklyReport(symbolOrder, bySymbol), nil
}

// SetLastRun upserts the singleton tracker_metadata row with the latest run time and outcome.
func (s *Store) SetLastRun(ctx context.Context, at time.Time, succeeded bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tracker_metadata (id, last_run_at, succeeded)
		VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE
			SET last_run_at = EXCLUDED.last_run_at,
			    succeeded   = EXCLUDED.succeeded`,
		at, succeeded,
	)
	return err
}

// GetLastRun reads the singleton tracker_metadata row.
func (s *Store) GetLastRun(ctx context.Context) (time.Time, bool, error) {
	var at time.Time
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT last_run_at, succeeded FROM tracker_metadata WHERE id = 1`).
		Scan(&at, &ok)
	return at, ok, err
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
