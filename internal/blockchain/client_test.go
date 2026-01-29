package blockchain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanBalance(t *testing.T) {
	tests := []struct {
		name     string
		raw      *big.Int
		decimals uint8
		want     string
	}{
		{
			name:     "zero balance",
			raw:      big.NewInt(0),
			decimals: 18,
			want:     "0",
		},
		{
			name:     "1 wei with 18 decimals",
			raw:      big.NewInt(1),
			decimals: 18,
			want:     "0.000000000000000001",
		},
		{
			name:     "1 token (18 decimals)",
			raw:      big.NewInt(1000000000000000000),
			decimals: 18,
			want:     "1",
		},
		{
			name:     "1.5 tokens (18 decimals)",
			raw:      big.NewInt(1500000000000000000),
			decimals: 18,
			want:     "1.5",
		},
		{
			name:     "token with no fractional part",
			raw:      big.NewInt(1000000000000000000),
			decimals: 18,
			want:     "1",
		},
		{
			name:     "6 decimals token (USDC-like)",
			raw:      big.NewInt(1500000),
			decimals: 6,
			want:     "1.5",
		},
		{
			name:     "0 decimals token",
			raw:      big.NewInt(100),
			decimals: 0,
			want:     "100",
		},
		{
			name: "large balance",
			raw: func() *big.Int {
				v, _ := big.NewInt(0).SetString("123456789000000000000000000", 10)
				return v
			}(),
			decimals: 18,
			want:     "123456789",
		},
		{
			name:     "trailing zeros trimmed",
			raw:      big.NewInt(1000000000000000000),
			decimals: 18,
			want:     "1",
		},
		{
			name:     "fractional with trailing zeros",
			raw:      big.NewInt(1100000000000000000),
			decimals: 18,
			want:     "1.1",
		},
		{
			name:     "very small fractional value",
			raw:      big.NewInt(1000),
			decimals: 18,
			want:     "0.000000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HumanBalance(tt.raw, tt.decimals)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBalanceCalculationEdgeCases(t *testing.T) {
	t.Run("balance with high precision decimals", func(t *testing.T) {
		raw, _ := big.NewInt(0).SetString("123456789123456789", 10)
		decimals := uint8(18)
		result := HumanBalance(raw, decimals)
		assert.Equal(t, "0.123456789123456789", result)
	})

	t.Run("balance at decimal boundary", func(t *testing.T) {
		raw, _ := big.NewInt(0).SetString("1000000000000000000", 10)
		decimals := uint8(18)
		result := HumanBalance(raw, decimals)
		assert.Equal(t, "1", result)
	})

	t.Run("very small balance", func(t *testing.T) {
		raw := big.NewInt(1)
		decimals := uint8(18)
		result := HumanBalance(raw, decimals)
		assert.Equal(t, "0.000000000000000001", result)
	})

	t.Run("large balance with small decimals", func(t *testing.T) {
		raw, _ := big.NewInt(0).SetString("999999999999999999", 10)
		decimals := uint8(6)
		result := HumanBalance(raw, decimals)
		assert.Equal(t, "999999999999.999999", result)
	})

	t.Run("negative balance (edge case)", func(t *testing.T) {
		// Negative balance should not occur in practice, but test the zero case
		raw := big.NewInt(-1)
		decimals := uint8(18)
		result := HumanBalance(raw, decimals)
		// The function checks rawBalance.Sign() == 0, so negative should process normally
		assert.NotEmpty(t, result)
	})
}

func TestHumanBalanceConsistency(t *testing.T) {
	t.Run("consistency across multiple calls", func(t *testing.T) {
		raw := big.NewInt(1234567890123456789)
		decimals := uint8(18)

		result1 := HumanBalance(raw, decimals)
		result2 := HumanBalance(raw, decimals)

		assert.Equal(t, result1, result2)
	})

	t.Run("preserves original big.Int", func(t *testing.T) {
		original := big.NewInt(1000000000000000000)
		originalStr := original.String()

		_ = HumanBalance(original, 18)

		// Verify original wasn't modified
		assert.Equal(t, originalStr, original.String())
	})
}

func TestHumanBalanceWithRealWorldNumbers(t *testing.T) {
	tests := []struct {
		name        string
		description string
		raw         *big.Int
		decimals    uint8
		expected    string
	}{
		{
			name:        "USDC with 1000 tokens",
			description: "1000 USDC (6 decimals)",
			raw:         big.NewInt(1000000000), // 1000 * 10^6
			decimals:    6,
			expected:    "1000",
		},
		{
			name: "DAI with fractional amount",
			description: "123.456 DAI (18 decimals)",
			raw: func() *big.Int {
				v, _ := big.NewInt(0).SetString("123456000000000000000", 10)
				return v
			}(),
			decimals: 18,
			expected: "123.456",
		},
		{
			name:        "USDT with fractional amount",
			description: "0.50 USDT (6 decimals)",
			raw:         big.NewInt(500000), // 0.5 * 10^6
			decimals:    6,
			expected:    "0.5",
		},
		{
			name: "ETH with wei",
			description: "2.5 ETH (18 decimals)",
			raw: func() *big.Int {
				v, _ := big.NewInt(0).SetString("2500000000000000000", 10)
				return v
			}(),
			decimals: 18,
			expected: "2.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HumanBalance(tt.raw, tt.decimals)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}
