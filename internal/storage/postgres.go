package storage

import (
	"context"
	"fmt"
	"time"

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
