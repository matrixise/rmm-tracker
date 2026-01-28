package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const createTableSQL = `
-- Drop existing table to migrate schema (removes label column)
CREATE TABLE IF NOT EXISTS token_balances (
	id            BIGSERIAL PRIMARY KEY,
	queried_at    TIMESTAMPTZ NOT NULL,
	wallet        TEXT NOT NULL,
	token_address TEXT NOT NULL,
	symbol        TEXT NOT NULL,
	decimals      SMALLINT NOT NULL,
	raw_balance   TEXT NOT NULL,
	balance       TEXT NOT NULL
);

-- Composite index for historical queries by wallet and token
CREATE INDEX IF NOT EXISTS idx_token_balances_wallet_token_time
	ON token_balances(wallet, token_address, queried_at DESC);

-- Index for time-based queries across all wallets
CREATE INDEX IF NOT EXISTS idx_token_balances_queried_at
	ON token_balances(queried_at DESC);

-- Index for wallet-wide queries
CREATE INDEX IF NOT EXISTS idx_token_balances_wallet
	ON token_balances(wallet);
`

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

// CreateSchema creates the table and indexes
func (s *Store) CreateSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
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
