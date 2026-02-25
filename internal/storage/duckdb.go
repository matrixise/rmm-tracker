package storage

import (
	"context"
	"database/sql"
	"fmt"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/shopspring/decimal"
)

const duckdbSchema = `
CREATE SEQUENCE IF NOT EXISTS token_balances_id_seq;
CREATE TABLE IF NOT EXISTS token_balances (
    id            BIGINT DEFAULT nextval('token_balances_id_seq'),
    queried_at    TIMESTAMPTZ NOT NULL,
    wallet        VARCHAR NOT NULL,
    token_address VARCHAR NOT NULL,
    symbol        VARCHAR NOT NULL,
    decimals      UTINYINT NOT NULL,
    balance       DOUBLE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_wallet_queried ON token_balances(wallet, queried_at DESC);
`

// DuckDBStore implements Storer using DuckDB as the storage backend.
type DuckDBStore struct {
	connector *duckdb.Connector
	db        *sql.DB
}

// NewDuckDBStore opens a DuckDB database at path (empty string = in-memory)
// and creates the schema if it doesn't exist.
func NewDuckDBStore(ctx context.Context, path string) (*DuckDBStore, error) {
	connector, err := duckdb.NewConnector(path, nil)
	if err != nil {
		return nil, fmt.Errorf("open DuckDB: %w", err)
	}

	db := sql.OpenDB(connector)

	if _, err := db.ExecContext(ctx, duckdbSchema); err != nil {
		_ = db.Close()
		_ = connector.Close()
		return nil, fmt.Errorf("create DuckDB schema: %w", err)
	}

	return &DuckDBStore{connector: connector, db: db}, nil
}

// Close closes the DuckDB database.
func (s *DuckDBStore) Close() {
	_ = s.db.Close()
}

// Ping verifies the DuckDB connection is alive.
func (s *DuckDBStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// BatchInsertBalances inserts multiple token balances using the DuckDB Appender API.
func (s *DuckDBStore) BatchInsertBalances(ctx context.Context, balances []TokenBalance) error {
	if len(balances) == 0 {
		return nil
	}

	conn, err := s.connector.Connect(ctx)
	if err != nil {
		return fmt.Errorf("connect for appender: %w", err)
	}
	defer func() { _ = conn.Close() }()

	appender, err := duckdb.NewAppenderWithColumns(conn, "", "", "token_balances",
		[]string{"queried_at", "wallet", "token_address", "symbol", "decimals", "balance"})
	if err != nil {
		return fmt.Errorf("create appender: %w", err)
	}

	for _, b := range balances {
		bal, _ := b.Balance.Float64()
		if err := appender.AppendRow(b.QueriedAt, b.Wallet, b.TokenAddress, b.Symbol, b.Decimals, bal); err != nil {
			_ = appender.Close()
			return fmt.Errorf("append row: %w", err)
		}
	}

	return appender.Close()
}

// GetBalances returns token balances with optional filters on wallet and symbol.
func (s *DuckDBStore) GetBalances(ctx context.Context, wallet, symbol string, limit int) ([]TokenBalance, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, queried_at, wallet, token_address, symbol, decimals, balance
		FROM token_balances
		WHERE (? = '' OR wallet = ?)
		  AND (? = '' OR symbol = ?)
		ORDER BY queried_at DESC
		LIMIT ?`,
		wallet, wallet, symbol, symbol, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var balances []TokenBalance
	for rows.Next() {
		var b TokenBalance
		var bal float64
		if err := rows.Scan(&b.ID, &b.QueriedAt, &b.Wallet, &b.TokenAddress, &b.Symbol, &b.Decimals, &bal); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		b.Balance = decimal.NewFromFloat(bal)
		balances = append(balances, b)
	}

	return balances, rows.Err()
}

// GetDailyBalances returns the last recorded balance per (day, symbol) for a wallet.
func (s *DuckDBStore) GetDailyBalances(ctx context.Context, wallet string) ([]DailyBalance, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT ON (DATE_TRUNC('day', queried_at)::DATE, symbol)
			DATE_TRUNC('day', queried_at)::DATE AS day,
			wallet,
			token_address,
			symbol,
			decimals,
			balance,
			queried_at
		FROM token_balances
		WHERE wallet = ?
		ORDER BY DATE_TRUNC('day', queried_at)::DATE DESC, symbol, queried_at DESC`,
		wallet,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []DailyBalance
	for rows.Next() {
		var b DailyBalance
		var bal float64
		if err := rows.Scan(&b.Day, &b.Wallet, &b.TokenAddress, &b.Symbol, &b.Decimals, &bal, &b.QueriedAt); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		b.Balance = decimal.NewFromFloat(bal)
		results = append(results, b)
	}
	return results, rows.Err()
}

// GetDailyReport returns per-token day-over-day balance comparisons for a wallet.
func (s *DuckDBStore) GetDailyReport(ctx context.Context, wallet string, days int) ([]DailyReport, error) {
	if days < 2 {
		return nil, fmt.Errorf("days must be >= 2")
	}
	rows, err := s.db.QueryContext(ctx, `
		WITH ranked AS (
			SELECT DISTINCT ON (DATE_TRUNC('day', queried_at)::DATE, symbol)
				DATE_TRUNC('day', queried_at)::DATE AS day_bucket,
				symbol, token_address, balance
			FROM token_balances
			WHERE wallet = ?
			ORDER BY DATE_TRUNC('day', queried_at)::DATE DESC, symbol, queried_at DESC
		),
		recent_days AS (
			SELECT day_bucket FROM ranked
			GROUP BY day_bucket
			ORDER BY day_bucket DESC
			LIMIT ?
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
		var bal float64
		if err := rows.Scan(&e.symbol, &e.tokenAddress, &e.dayBucket, &bal); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		e.balance = decimal.NewFromFloat(bal)
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

// GetWeeklyBalances returns the last recorded balance per (week, symbol) for a wallet.
func (s *DuckDBStore) GetWeeklyBalances(ctx context.Context, wallet string) ([]WeeklyBalance, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT ON (DATE_TRUNC('week', queried_at)::DATE, symbol)
			DATE_TRUNC('week', queried_at)::DATE AS week,
			wallet,
			token_address,
			symbol,
			decimals,
			balance,
			queried_at
		FROM token_balances
		WHERE wallet = ?
		ORDER BY DATE_TRUNC('week', queried_at)::DATE DESC, symbol, queried_at DESC`,
		wallet,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []WeeklyBalance
	for rows.Next() {
		var b WeeklyBalance
		var bal float64
		if err := rows.Scan(&b.Week, &b.Wallet, &b.TokenAddress, &b.Symbol, &b.Decimals, &bal, &b.QueriedAt); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		b.Balance = decimal.NewFromFloat(bal)
		results = append(results, b)
	}
	return results, rows.Err()
}

// GetWeeklyReport returns per-token balance comparison between current and N-1 previous weeks.
func (s *DuckDBStore) GetWeeklyReport(ctx context.Context, wallet string, weeks int) ([]WeeklyReport, error) {
	if weeks < 2 {
		return nil, fmt.Errorf("weeks must be >= 2")
	}
	rows, err := s.db.QueryContext(ctx, `
		WITH ranked AS (
			SELECT DISTINCT ON (DATE_TRUNC('week', queried_at)::DATE, symbol)
				DATE_TRUNC('week', queried_at)::DATE AS week_bucket,
				symbol, token_address, balance
			FROM token_balances
			WHERE wallet = ?
			ORDER BY DATE_TRUNC('week', queried_at)::DATE DESC, symbol, queried_at DESC
		),
		recent_weeks AS (
			SELECT week_bucket FROM ranked
			GROUP BY week_bucket
			ORDER BY week_bucket DESC
			LIMIT ?
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
		var bal float64
		if err := rows.Scan(&e.symbol, &e.tokenAddress, &e.weekBucket, &bal); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		e.balance = decimal.NewFromFloat(bal)
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

// GetWallets returns distinct wallet addresses stored in the database.
func (s *DuckDBStore) GetWallets(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT wallet FROM token_balances ORDER BY wallet`)
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
