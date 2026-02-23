//go:build integration

package storage

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (context.Context, *Store) {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()

	err := RunMigrations(ctx, dsn)
	require.NoError(t, err, "migrations should run without error")

	store, err := NewStore(ctx, dsn)
	require.NoError(t, err, "store should be created successfully")
	t.Cleanup(func() { store.Close() })

	t.Cleanup(func() {
		_, err := store.pool.Exec(ctx, "TRUNCATE TABLE token_balances RESTART IDENTITY CASCADE")
		if err != nil {
			t.Logf("cleanup truncate failed: %v", err)
		}
	})

	return ctx, store
}

func TestIntegration_InsertAndGetBalances(t *testing.T) {
	ctx, store := newTestStore(t)

	wallet := "0x1234567890123456789012345678901234567890"
	tokenAddress1 := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"
	tokenAddress2 := "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	// Use distinct timestamps to ensure deterministic ordering (sorted by queried_at DESC)
	now := time.Now().UTC().Truncate(time.Millisecond)
	t1 := now
	t2 := now.Add(-time.Second)

	balances := []TokenBalance{
		{
			QueriedAt:    t1,
			Wallet:       wallet,
			TokenAddress: tokenAddress1,
			Symbol:       "armmXDAI",
			Decimals:     18,
			RawBalance:   big.NewInt(1_500_000_000_000_000_000),
			Balance:      decimal.NewFromFloat(1.5),
		},
		{
			QueriedAt:    t2,
			Wallet:       wallet,
			TokenAddress: tokenAddress2,
			Symbol:       "armmUSDC",
			Decimals:     6,
			RawBalance:   big.NewInt(2_000_000),
			Balance:      decimal.NewFromFloat(2.0),
		},
	}

	err := store.BatchInsertBalances(ctx, balances)
	require.NoError(t, err, "BatchInsertBalances should succeed")

	// No filter — ordered by queried_at DESC: armmXDAI first
	got, err := store.GetBalances(ctx, "", "", 100)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "armmXDAI", got[0].Symbol)
	require.Equal(t, "armmUSDC", got[1].Symbol)

	// Filter by wallet
	got, err = store.GetBalances(ctx, wallet, "", 100)
	require.NoError(t, err)
	require.Len(t, got, 2)

	// Filter by symbol — full field assertions
	got, err = store.GetBalances(ctx, "", "armmXDAI", 100)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "armmXDAI", got[0].Symbol)
	require.Equal(t, wallet, got[0].Wallet)
	require.Equal(t, tokenAddress1, got[0].TokenAddress)
	require.Equal(t, uint8(18), got[0].Decimals)
	require.True(t, got[0].Balance.Equal(decimal.NewFromFloat(1.5)))
	require.True(t, t1.Equal(got[0].QueriedAt), "QueriedAt should match: expected %v, got %v", t1, got[0].QueriedAt)

	// Filter by wallet + symbol
	got, err = store.GetBalances(ctx, wallet, "armmUSDC", 100)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "armmUSDC", got[0].Symbol)
	require.Equal(t, tokenAddress2, got[0].TokenAddress)
	require.Equal(t, uint8(6), got[0].Decimals)
	require.True(t, got[0].Balance.Equal(decimal.NewFromFloat(2.0)))

	// Unknown wallet — empty result
	got, err = store.GetBalances(ctx, "0x0000000000000000000000000000000000000000", "", 100)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestIntegration_BatchInsertEmpty(t *testing.T) {
	ctx, store := newTestStore(t)

	err := store.BatchInsertBalances(ctx, []TokenBalance{})
	require.NoError(t, err, "BatchInsertBalances with empty slice should be a no-op")
}
