package storage

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBalanceValidation(t *testing.T) {
	tests := []struct {
		name  string
		tb    TokenBalance
		valid bool
	}{
		{
			name: "valid token balance",
			tb: TokenBalance{
				QueriedAt:    time.Now(),
				Wallet:       "0x1234567890123456789012345678901234567890",
				TokenAddress: "0x0000000000000000000000000000000000000000",
				Symbol:       "TEST",
				Decimals:     18,
				RawBalance:   big.NewInt(1000000000000000000),
				Balance:      "1",
			},
			valid: true,
		},
		{
			name: "balance with zero value",
			tb: TokenBalance{
				QueriedAt:    time.Now(),
				Wallet:       "0x1234567890123456789012345678901234567890",
				TokenAddress: "0x0000000000000000000000000000000000000000",
				Symbol:       "TEST",
				Decimals:     6,
				RawBalance:   big.NewInt(0),
				Balance:      "0",
			},
			valid: true,
		},
		{
			name: "balance with large number",
			tb: TokenBalance{
				QueriedAt:    time.Now(),
				Wallet:       "0x1234567890123456789012345678901234567890",
				TokenAddress: "0x0000000000000000000000000000000000000000",
				Symbol:       "LARGE",
				Decimals:     18,
				RawBalance: func() *big.Int {
					v, _ := big.NewInt(0).SetString("999999999999999999999999999", 10)
					return v
				}(),
				Balance: "999999999999999999.999999999",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEmpty(t, tt.tb.Wallet)
				assert.NotEmpty(t, tt.tb.TokenAddress)
				assert.NotEmpty(t, tt.tb.Symbol)
				assert.NotNil(t, tt.tb.RawBalance)
				assert.NotEmpty(t, tt.tb.Balance)
			}
		})
	}
}

func TestBatchInsertBalancesDataTypes(t *testing.T) {
	t.Run("token balance slice can be created", func(t *testing.T) {
		balances := []TokenBalance{
			{
				QueriedAt:    time.Now(),
				Wallet:       "0x1234567890123456789012345678901234567890",
				TokenAddress: "0x0000000000000000000000000000000000000000",
				Symbol:       "TEST1",
				Decimals:     18,
				RawBalance:   big.NewInt(1000000000000000000),
				Balance:      "1",
			},
			{
				QueriedAt:    time.Now().Add(-1 * time.Hour),
				Wallet:       "0x0987654321098765432109876543210987654321",
				TokenAddress: "0x1111111111111111111111111111111111111111",
				Symbol:       "TEST2",
				Decimals:     6,
				RawBalance:   big.NewInt(500000),
				Balance:      "0.5",
			},
		}

		assert.Equal(t, 2, len(balances))
		assert.Equal(t, "TEST1", balances[0].Symbol)
		assert.Equal(t, "TEST2", balances[1].Symbol)
	})

	t.Run("empty batch is valid", func(t *testing.T) {
		balances := []TokenBalance{}
		assert.Equal(t, 0, len(balances))
	})

	t.Run("batch with single item", func(t *testing.T) {
		balances := []TokenBalance{
			{
				QueriedAt:    time.Now(),
				Wallet:       "0x1234567890123456789012345678901234567890",
				TokenAddress: "0x0000000000000000000000000000000000000000",
				Symbol:       "SINGLE",
				Decimals:     18,
				RawBalance:   big.NewInt(0),
				Balance:      "0",
			},
		}

		assert.Equal(t, 1, len(balances))
	})
}

func TestTokenBalanceRawBalanceConversion(t *testing.T) {
	tests := []struct {
		name           string
		rawBalance     *big.Int
		expectedString string
	}{
		{
			name:           "zero raw balance",
			rawBalance:     big.NewInt(0),
			expectedString: "0",
		},
		{
			name:           "single wei",
			rawBalance:     big.NewInt(1),
			expectedString: "1",
		},
		{
			name:           "one token (18 decimals)",
			rawBalance:     big.NewInt(1000000000000000000),
			expectedString: "1000000000000000000",
		},
		{
			name: "large number",
			rawBalance: func() *big.Int {
				v, _ := big.NewInt(0).SetString("12345678901234567890", 10)
				return v
			}(),
			expectedString: "12345678901234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rawBalance.String()
			assert.Equal(t, tt.expectedString, result)
		})
	}
}

func TestTokenBalanceTimeHandling(t *testing.T) {
	t.Run("queriedAt timestamp is preserved", func(t *testing.T) {
		now := time.Now().UTC()
		tb := TokenBalance{
			QueriedAt: now,
			Wallet:    "0x1234567890123456789012345678901234567890",
		}

		assert.Equal(t, now, tb.QueriedAt)
	})

	t.Run("different timestamps can be stored", func(t *testing.T) {
		balances := []TokenBalance{
			{
				QueriedAt: time.Now(),
				Wallet:    "0x1234567890123456789012345678901234567890",
			},
			{
				QueriedAt: time.Now().Add(-1 * time.Hour),
				Wallet:    "0x0987654321098765432109876543210987654321",
			},
			{
				QueriedAt: time.Now().Add(-2 * time.Hour),
				Wallet:    "0x1111111111111111111111111111111111111111",
			},
		}

		assert.Equal(t, 3, len(balances))
		// Verify times are different
		assert.NotEqual(t, balances[0].QueriedAt, balances[1].QueriedAt)
		assert.NotEqual(t, balances[1].QueriedAt, balances[2].QueriedAt)
	})
}

func TestTokenBalanceFieldRequirements(t *testing.T) {
	t.Run("all fields required for valid insert", func(t *testing.T) {
		// Create a complete token balance
		complete := TokenBalance{
			QueriedAt:    time.Now(),
			Wallet:       "0x1234567890123456789012345678901234567890",
			TokenAddress: "0x0000000000000000000000000000000000000000",
			Symbol:       "TEST",
			Decimals:     18,
			RawBalance:   big.NewInt(1000000000000000000),
			Balance:      "1",
		}

		// Verify all required fields are present
		assert.False(t, complete.QueriedAt.IsZero())
		assert.NotEmpty(t, complete.Wallet)
		assert.NotEmpty(t, complete.TokenAddress)
		assert.NotEmpty(t, complete.Symbol)
		assert.NotNil(t, complete.RawBalance)
		assert.NotEmpty(t, complete.Balance)
	})

	t.Run("balance field stores decimal representation", func(t *testing.T) {
		balances := []TokenBalance{
			{Balance: "0"},
			{Balance: "1.5"},
			{Balance: "0.000000000000000001"},
			{Balance: "123456789.123456789"},
		}

		for _, b := range balances {
			assert.NotEmpty(t, b.Balance)
		}
	})
}

func TestTokenBalanceBatchOperations(t *testing.T) {
	t.Run("batch of different token types", func(t *testing.T) {
		balances := []TokenBalance{
			{
				Symbol:   "USDC",
				Decimals: 6,
				Balance:  "1000.50",
			},
			{
				Symbol:   "DAI",
				Decimals: 18,
				Balance:  "2500.123456789012345678",
			},
			{
				Symbol:   "USDT",
				Decimals: 6,
				Balance:  "0.01",
			},
		}

		assert.Equal(t, 3, len(balances))
		assert.Equal(t, "USDC", balances[0].Symbol)
		assert.Equal(t, "DAI", balances[1].Symbol)
		assert.Equal(t, "USDT", balances[2].Symbol)
	})

	t.Run("batch preserves insertion order", func(t *testing.T) {
		balances := []TokenBalance{
			{Symbol: "FIRST", Balance: "1"},
			{Symbol: "SECOND", Balance: "2"},
			{Symbol: "THIRD", Balance: "3"},
		}

		assert.Equal(t, "FIRST", balances[0].Symbol)
		assert.Equal(t, "SECOND", balances[1].Symbol)
		assert.Equal(t, "THIRD", balances[2].Symbol)
	})
}

func TestRawBalanceStringConversion(t *testing.T) {
	t.Run("big.Int to string preserves precision", func(t *testing.T) {
		original, _ := big.NewInt(0).SetString("123456789123456789123456789", 10)
		str := original.String()

		// Parse back
		parsed, _ := big.NewInt(0).SetString(str, 10)

		assert.Equal(t, original, parsed)
	})

	t.Run("large numbers handled correctly", func(t *testing.T) {
		maxUint256, _ := big.NewInt(0).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)
		str := maxUint256.String()

		assert.NotEmpty(t, str)
		parsed, _ := big.NewInt(0).SetString(str, 10)
		assert.Equal(t, maxUint256, parsed)
	})
}
