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

func TestIntegration_InsertAndGetBalances(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()

	// Run migrations
	err := RunMigrations(ctx, dsn)
	require.NoError(t, err, "migrations should run without error")

	// Connect to store
	store, err := NewStore(ctx, dsn)
	require.NoError(t, err, "store should be created successfully")
	defer store.Close()

	// Clean up after test
	t.Cleanup(func() {
		_, err := store.pool.Exec(ctx, "TRUNCATE TABLE token_balances RESTART IDENTITY CASCADE")
		if err != nil {
			t.Logf("cleanup truncate failed: %v", err)
		}
	})

	wallet := "0x1234567890123456789012345678901234567890"
	tokenAddress := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"
	queriedAt := time.Now().UTC().Truncate(time.Millisecond)

	balances := []TokenBalance{
		{
			QueriedAt:    queriedAt,
			Wallet:       wallet,
			TokenAddress: tokenAddress,
			Symbol:       "armmXDAI",
			Decimals:     18,
			RawBalance:   big.NewInt(1_500_000_000_000_000_000),
			Balance:      decimal.NewFromFloat(1.5),
		},
		{
			QueriedAt:    queriedAt,
			Wallet:       wallet,
			TokenAddress: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Symbol:       "armmUSDC",
			Decimals:     6,
			RawBalance:   big.NewInt(2_000_000),
			Balance:      decimal.NewFromFloat(2.0),
		},
	}

	// Insert balances
	err = store.BatchInsertBalances(ctx, balances)
	require.NoError(t, err, "BatchInsertBalances should succeed")

	// Retrieve balances without filter
	got, err := store.GetBalances(ctx, "", "", 100)
	require.NoError(t, err, "GetBalances should succeed")
	require.Len(t, got, 2, "should have 2 records")

	// Retrieve balances filtered by wallet
	got, err = store.GetBalances(ctx, wallet, "", 100)
	require.NoError(t, err, "GetBalances filtered by wallet should succeed")
	require.Len(t, got, 2, "should have 2 records for the wallet")

	// Retrieve balances filtered by symbol
	got, err = store.GetBalances(ctx, "", "armmXDAI", 100)
	require.NoError(t, err, "GetBalances filtered by symbol should succeed")
	require.Len(t, got, 1, "should have 1 record for armmXDAI")
	require.Equal(t, "armmXDAI", got[0].Symbol)
	require.Equal(t, wallet, got[0].Wallet)
	require.True(t, got[0].Balance.Equal(decimal.NewFromFloat(1.5)), "balance should be 1.5")

	// Retrieve balances filtered by wallet and symbol
	got, err = store.GetBalances(ctx, wallet, "armmUSDC", 100)
	require.NoError(t, err, "GetBalances filtered by wallet and symbol should succeed")
	require.Len(t, got, 1, "should have 1 record for armmUSDC")
	require.Equal(t, "armmUSDC", got[0].Symbol)

	// Verify empty result for non-existent wallet
	got, err = store.GetBalances(ctx, "0x0000000000000000000000000000000000000000", "", 100)
	require.NoError(t, err, "GetBalances for unknown wallet should succeed")
	require.Empty(t, got, "should have no records for unknown wallet")
}

func TestIntegration_BatchInsertEmpty(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()

	err := RunMigrations(ctx, dsn)
	require.NoError(t, err, "migrations should run without error")

	store, err := NewStore(ctx, dsn)
	require.NoError(t, err, "store should be created successfully")
	defer store.Close()

	// Empty batch should be a no-op
	err = store.BatchInsertBalances(ctx, []TokenBalance{})
	require.NoError(t, err, "BatchInsertBalances with empty slice should be a no-op")
}
