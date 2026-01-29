package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEthAddressValidator(t *testing.T) {
	cfg := &Config{}
	v := NewValidator()

	tests := []struct {
		name      string
		address   string
		wantError bool
	}{
		{
			name:      "valid address with 0x prefix",
			address:   "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			wantError: false,
		},
		{
			name:      "valid address all lowercase",
			address:   "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
			wantError: false,
		},
		{
			name:      "valid address all uppercase",
			address:   "0x742D35CC6634C0532925A3B844BC9E7595F0BEB0",
			wantError: false,
		},
		{
			name:      "zero address is valid",
			address:   "0x0000000000000000000000000000000000000000",
			wantError: false,
		},
		{
			name:      "valid address without 0x prefix",
			address:   "742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			wantError: false,
		},
		{
			name:      "too short",
			address:   "0x742d35Cc",
			wantError: true,
		},
		{
			name:      "too long",
			address:   "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb123",
			wantError: true,
		},
		{
			name:      "invalid hex character",
			address:   "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEg0",
			wantError: true,
		},
		{
			name:      "empty string",
			address:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Wallets = []string{tt.address}
			cfg.RPCUrls = []string{"https://rpc.example.com"}
			cfg.Tokens = []TokenConfig{
				{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
			}

			err := v.Struct(cfg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScheduleValidator(t *testing.T) {
	cfg := &Config{
		RPCUrls: []string{"https://rpc.example.com"},
		Wallets: []string{"0x1234567890123456789012345678901234567890"},
		Tokens: []TokenConfig{
			{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
		},
	}
	v := NewValidator()

	tests := []struct {
		name      string
		interval  string
		wantError bool
	}{
		{
			name:      "valid duration 5m",
			interval:  "5m",
			wantError: false,
		},
		{
			name:      "valid duration 1h",
			interval:  "1h",
			wantError: false,
		},
		{
			name:      "valid cron 5 fields",
			interval:  "*/5 * * * *",
			wantError: false,
		},
		{
			name:      "valid cron 6 fields with seconds",
			interval:  "*/30 * * * * *",
			wantError: false,
		},
		{
			name:      "empty interval is valid (one-shot mode)",
			interval:  "",
			wantError: false,
		},
		{
			name:      "invalid duration 7m (not divisor of 60)",
			interval:  "7m",
			wantError: true,
		},
		{
			name:      "invalid duration 5h (not divisor of 24)",
			interval:  "5h",
			wantError: true,
		},
		{
			name:      "invalid cron too few fields",
			interval:  "*/5 * * *",
			wantError: true,
		},
		{
			name:      "invalid cron too many fields",
			interval:  "*/5 * * * * * *",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Interval = tt.interval
			err := v.Struct(cfg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTimezoneValidator(t *testing.T) {
	cfg := &Config{
		RPCUrls: []string{"https://rpc.example.com"},
		Wallets: []string{"0x1234567890123456789012345678901234567890"},
		Tokens: []TokenConfig{
			{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
		},
	}
	v := NewValidator()

	tests := []struct {
		name      string
		timezone  string
		wantError bool
	}{
		{
			name:      "valid UTC",
			timezone:  "UTC",
			wantError: false,
		},
		{
			name:      "valid America/New_York",
			timezone:  "America/New_York",
			wantError: false,
		},
		{
			name:      "valid Europe/Paris",
			timezone:  "Europe/Paris",
			wantError: false,
		},
		{
			name:      "valid Asia/Tokyo",
			timezone:  "Asia/Tokyo",
			wantError: false,
		},
		{
			name:      "empty timezone is valid (defaults to UTC)",
			timezone:  "",
			wantError: false,
		},
		{
			name:      "invalid timezone",
			timezone:  "Invalid/Timezone",
			wantError: true,
		},
		{
			name:      "random string",
			timezone:  "NotATimezone",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Timezone = tt.timezone
			err := v.Struct(cfg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDurationValidator(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		interval  string
		wantError bool
	}{
		{
			name:      "valid 1m",
			interval:  "1m",
			wantError: false,
		},
		{
			name:      "valid 5m",
			interval:  "5m",
			wantError: false,
		},
		{
			name:      "valid 10m",
			interval:  "10m",
			wantError: false,
		},
		{
			name:      "valid 15m",
			interval:  "15m",
			wantError: false,
		},
		{
			name:      "valid 30m",
			interval:  "30m",
			wantError: false,
		},
		{
			name:      "valid 1h",
			interval:  "1h",
			wantError: false,
		},
		{
			name:      "valid 2h",
			interval:  "2h",
			wantError: false,
		},
		{
			name:      "valid 6h",
			interval:  "6h",
			wantError: false,
		},
		{
			name:      "valid 12h",
			interval:  "12h",
			wantError: false,
		},
		{
			name:      "valid 24h",
			interval:  "24h",
			wantError: false,
		},
		{
			name:      "invalid 7m",
			interval:  "7m",
			wantError: true,
		},
		{
			name:      "invalid 13m",
			interval:  "13m",
			wantError: true,
		},
		{
			name:      "invalid 5h",
			interval:  "5h",
			wantError: true,
		},
		{
			name:      "invalid 7h",
			interval:  "7h",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				RPCUrls:  []string{"https://rpc.example.com"},
				Wallets:  []string{"0x1234567890123456789012345678901234567890"},
				Interval: tt.interval,
				Tokens: []TokenConfig{
					{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
				},
			}
			err := v.Struct(cfg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatorCustomTypes(t *testing.T) {
	v := NewValidator()

	t.Run("validates URLs in RPCUrls", func(t *testing.T) {
		cfg := &Config{
			RPCUrls: []string{"https://valid.example.com", "http://another.example.com"},
			Wallets: []string{"0x1234567890123456789012345678901234567890"},
			Tokens: []TokenConfig{
				{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
			},
		}
		err := v.Struct(cfg)
		assert.NoError(t, err)
	})

	t.Run("rejects invalid URLs in RPCUrls", func(t *testing.T) {
		cfg := &Config{
			RPCUrls: []string{"not-a-url"},
			Wallets: []string{"0x1234567890123456789012345678901234567890"},
			Tokens: []TokenConfig{
				{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
			},
		}
		err := v.Struct(cfg)
		assert.Error(t, err)
	})

	t.Run("requires at least one wallet", func(t *testing.T) {
		cfg := &Config{
			RPCUrls: []string{"https://rpc.example.com"},
			Wallets: []string{},
			Tokens: []TokenConfig{
				{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
			},
		}
		err := v.Struct(cfg)
		assert.Error(t, err)
	})

	t.Run("requires at least one token", func(t *testing.T) {
		cfg := &Config{
			RPCUrls: []string{"https://rpc.example.com"},
			Wallets: []string{"0x1234567890123456789012345678901234567890"},
			Tokens:  []TokenConfig{},
		}
		err := v.Struct(cfg)
		assert.Error(t, err)
	})
}

func TestValidatorIntegration(t *testing.T) {
	v := NewValidator()

	t.Run("complete valid config passes all validators", func(t *testing.T) {
		cfg := &Config{
			RPCUrls:  []string{"https://rpc1.example.com", "https://rpc2.example.com"},
			Wallets:  []string{"0x1234567890123456789012345678901234567890", "0x0987654321098765432109876543210987654321"},
			Interval: "5m",
			LogLevel: "debug",
			HTTPPort: 8080,
			Timezone: "America/New_York",
			Tokens: []TokenConfig{
				{Label: "TOKEN1", Address: "0x1111111111111111111111111111111111111111", FallbackDecimals: 18},
				{Label: "TOKEN2", Address: "0x2222222222222222222222222222222222222222", FallbackDecimals: 6},
			},
		}
		err := v.Struct(cfg)
		assert.NoError(t, err)
	})

	t.Run("minimal valid config passes", func(t *testing.T) {
		cfg := &Config{
			RPCUrls: []string{"https://rpc.example.com"},
			Wallets: []string{"0x1234567890123456789012345678901234567890"},
			Tokens: []TokenConfig{
				{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
			},
		}
		err := v.Struct(cfg)
		assert.NoError(t, err)
	})
}
