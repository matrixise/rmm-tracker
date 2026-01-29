package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigNormalize(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantError bool
		check     func(*Config)
	}{
		{
			name: "single rpc_url converts to rpc_urls",
			cfg: &Config{
				RPCUrl:  "https://rpc1.example.com",
				RPCUrls: nil,
			},
			wantError: false,
			check: func(c *Config) {
				assert.Empty(t, c.RPCUrl)
				assert.Equal(t, []string{"https://rpc1.example.com"}, c.RPCUrls)
			},
		},
		{
			name: "rpc_urls takes precedence over rpc_url",
			cfg: &Config{
				RPCUrl:  "https://rpc1.example.com",
				RPCUrls: []string{"https://rpc2.example.com", "https://rpc3.example.com"},
			},
			wantError: false,
			check: func(c *Config) {
				assert.Empty(t, c.RPCUrl)
				assert.Equal(t, []string{"https://rpc2.example.com", "https://rpc3.example.com"}, c.RPCUrls)
			},
		},
		{
			name: "empty rpc_urls with non-empty rpc_url still converts",
			cfg: &Config{
				RPCUrl:  "https://rpc1.example.com",
				RPCUrls: []string{},
			},
			wantError: false,
			check: func(c *Config) {
				assert.Empty(t, c.RPCUrl)
				assert.Equal(t, []string{"https://rpc1.example.com"}, c.RPCUrls)
			},
		},
		{
			name: "both empty rpc_url and rpc_urls returns error",
			cfg: &Config{
				RPCUrl:  "",
				RPCUrls: nil,
			},
			wantError: true,
		},
		{
			name: "rpc_urls already set, no change",
			cfg: &Config{
				RPCUrl:  "",
				RPCUrls: []string{"https://rpc1.example.com"},
			},
			wantError: false,
			check: func(c *Config) {
				assert.Empty(t, c.RPCUrl)
				assert.Equal(t, []string{"https://rpc1.example.com"}, c.RPCUrls)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Normalize()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(tt.cfg)
				}
			}
		})
	}
}

func TestConfigGetTimezone(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		wantName string
	}{
		{
			name: "UTC timezone",
			cfg: &Config{
				Timezone: "UTC",
			},
			wantName: "UTC",
		},
		{
			name: "empty timezone defaults to UTC",
			cfg: &Config{
				Timezone: "",
			},
			wantName: "UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tz := tt.cfg.GetTimezone()
			assert.Equal(t, tt.wantName, tz.String())
		})
	}
}

func TestConfigShouldRunImmediately(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *Config
		wantRun  bool
	}{
		{
			name: "true when explicitly set",
			cfg: &Config{
				RunImmediately: &trueVal,
			},
			wantRun: true,
		},
		{
			name: "false when explicitly disabled",
			cfg: &Config{
				RunImmediately: &falseVal,
			},
			wantRun: false,
		},
		{
			name: "nil pointer defaults to true",
			cfg: &Config{
				RunImmediately: nil,
			},
			wantRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantRun, tt.cfg.ShouldRunImmediately())
		})
	}
}

func TestConfigIsCronExpression(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected bool
	}{
		{
			name: "duration is not cron",
			cfg: &Config{
				Interval: "5m",
			},
			expected: false,
		},
		{
			name: "cron expression detected",
			cfg: &Config{
				Interval: "*/5 * * * *",
			},
			expected: true,
		},
		{
			name: "six-field cron with seconds",
			cfg: &Config{
				Interval: "*/30 * * * * *",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.IsCronExpression()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	assert.NotNil(t, validator)

	// Test that validator has custom validators registered
	t.Run("eth_addr validator registered", func(t *testing.T) {
		cfg := &Config{
			RPCUrls: []string{"https://rpc.example.com"},
			Wallets: []string{"0x1234567890123456789012345678901234567890"},
			Tokens: []TokenConfig{
				{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
			},
		}
		err := validator.Struct(cfg)
		assert.NoError(t, err)
	})

	t.Run("schedule validator registered", func(t *testing.T) {
		cfg := &Config{
			RPCUrls:  []string{"https://rpc.example.com"},
			Wallets:  []string{"0x1234567890123456789012345678901234567890"},
			Interval: "5m",
			Tokens: []TokenConfig{
				{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
			},
		}
		err := validator.Struct(cfg)
		assert.NoError(t, err)
	})

	t.Run("timezone validator registered", func(t *testing.T) {
		cfg := &Config{
			RPCUrls:  []string{"https://rpc.example.com"},
			Wallets:  []string{"0x1234567890123456789012345678901234567890"},
			Timezone: "America/New_York",
			Tokens: []TokenConfig{
				{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
			},
		}
		err := validator.Struct(cfg)
		assert.NoError(t, err)
	})
}

func TestTokenConfigValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name      string
		token     TokenConfig
		wantError bool
	}{
		{
			name: "valid token config",
			token: TokenConfig{
				Label:            "TEST",
				Address:          "0x0000000000000000000000000000000000000000",
				FallbackDecimals: 18,
			},
			wantError: false,
		},
		{
			name: "missing label",
			token: TokenConfig{
				Address:          "0x0000000000000000000000000000000000000000",
				FallbackDecimals: 18,
			},
			wantError: true,
		},
		{
			name: "invalid address",
			token: TokenConfig{
				Label:            "TEST",
				Address:          "invalid",
				FallbackDecimals: 18,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				RPCUrls: []string{"https://rpc.example.com"},
				Wallets: []string{"0x1234567890123456789012345678901234567890"},
				Tokens:  []TokenConfig{tt.token},
			}
			err := validator.Struct(cfg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigHTTPPortValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name      string
		httpPort  int
		wantError bool
	}{
		{
			name:      "valid port 8080",
			httpPort:  8080,
			wantError: false,
		},
		{
			name:      "valid port 9090",
			httpPort:  9090,
			wantError: false,
		},
		{
			name:      "port too low (1023)",
			httpPort:  1023,
			wantError: true,
		},
		{
			name:      "port too high (65536)",
			httpPort:  65536,
			wantError: true,
		},
		{
			name:      "minimum valid port (1024)",
			httpPort:  1024,
			wantError: false,
		},
		{
			name:      "maximum valid port (65535)",
			httpPort:  65535,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				RPCUrls:  []string{"https://rpc.example.com"},
				Wallets:  []string{"0x1234567890123456789012345678901234567890"},
				HTTPPort: tt.httpPort,
				Tokens: []TokenConfig{
					{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
				},
			}
			err := validator.Struct(cfg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigLogLevelValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name      string
		logLevel  string
		wantError bool
	}{
		{
			name:      "valid debug",
			logLevel:  "debug",
			wantError: false,
		},
		{
			name:      "valid info",
			logLevel:  "info",
			wantError: false,
		},
		{
			name:      "valid warn",
			logLevel:  "warn",
			wantError: false,
		},
		{
			name:      "valid error",
			logLevel:  "error",
			wantError: false,
		},
		{
			name:      "invalid level",
			logLevel:  "invalid",
			wantError: true,
		},
		{
			name:      "empty is valid (uses default)",
			logLevel:  "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				RPCUrls:  []string{"https://rpc.example.com"},
				Wallets:  []string{"0x1234567890123456789012345678901234567890"},
				LogLevel: tt.logLevel,
				Tokens: []TokenConfig{
					{Label: "TEST", Address: "0x0000000000000000000000000000000000000000", FallbackDecimals: 18},
				},
			}
			err := validator.Struct(cfg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
