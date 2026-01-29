package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("loads valid TOML config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_urls = ["https://rpc.example.com"]
wallets = ["0x1234567890123456789012345678901234567890"]
log_level = "debug"

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, []string{"https://rpc.example.com"}, cfg.RPCUrls)
		assert.Equal(t, []string{"0x1234567890123456789012345678901234567890"}, cfg.Wallets)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Len(t, cfg.Tokens, 1)
		assert.Equal(t, "TEST", cfg.Tokens[0].Label)
	})

	t.Run("config from env vars only without config file", func(t *testing.T) {
		// Set all required env vars including tokens
		os.Setenv("RMM_TRACKER_RPC_URLS", "https://rpc.example.com")
		os.Setenv("RMM_TRACKER_WALLETS", "0x1234567890123456789012345678901234567890")
		defer os.Unsetenv("RMM_TRACKER_RPC_URLS")
		defer os.Unsetenv("RMM_TRACKER_WALLETS")

		// Create empty config file (config file found but empty - should load from env vars)
		tmpDir := t.TempDir()
		emptyConfigPath := filepath.Join(tmpDir, "empty.toml")
		err := os.WriteFile(emptyConfigPath, []byte(""), 0644)
		require.NoError(t, err)

		// Note: This will still fail validation because tokens array cannot be set via env vars
		// Tokens must be in config file or provided as structured data
		_, err = Load(emptyConfigPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation")
	})

	t.Run("environment variables override config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_urls = ["https://file-rpc.example.com"]
wallets = ["0x1111111111111111111111111111111111111111"]
log_level = "info"

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		os.Setenv("RMM_TRACKER_LOG_LEVEL", "debug")
		defer os.Unsetenv("RMM_TRACKER_LOG_LEVEL")

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "debug", cfg.LogLevel) // Env var overrides file
	})

	t.Run("comma-separated wallets from env", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_urls = ["https://rpc.example.com"]

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		os.Setenv("RMM_TRACKER_WALLETS", "0x1111111111111111111111111111111111111111, 0x2222222222222222222222222222222222222222")
		defer os.Unsetenv("RMM_TRACKER_WALLETS")

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Len(t, cfg.Wallets, 2)
		assert.Equal(t, "0x1111111111111111111111111111111111111111", cfg.Wallets[0])
		assert.Equal(t, "0x2222222222222222222222222222222222222222", cfg.Wallets[1])
	})

	t.Run("comma-separated RPC URLs from env", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
wallets = ["0x1234567890123456789012345678901234567890"]

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		os.Setenv("RMM_TRACKER_RPC_URLS", "https://rpc1.example.com, https://rpc2.example.com, https://rpc3.example.com")
		defer os.Unsetenv("RMM_TRACKER_RPC_URLS")

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Len(t, cfg.RPCUrls, 3)
		assert.Equal(t, "https://rpc1.example.com", cfg.RPCUrls[0])
		assert.Equal(t, "https://rpc2.example.com", cfg.RPCUrls[1])
		assert.Equal(t, "https://rpc3.example.com", cfg.RPCUrls[2])
	})

	t.Run("validation fails for invalid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_urls = ["https://rpc.example.com"]
wallets = ["invalid-address"]

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, err = Load(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation")
	})

	t.Run("normalization is applied", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_url = "https://single-rpc.example.com"
wallets = ["0x1234567890123456789012345678901234567890"]

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := Load(configPath)
		require.NoError(t, err)

		// Normalization should convert single rpc_url to rpc_urls array
		assert.Empty(t, cfg.RPCUrl)
		assert.Equal(t, []string{"https://single-rpc.example.com"}, cfg.RPCUrls)
	})
}

func TestLoadWithDefaults(t *testing.T) {
	t.Run("loads config with DATABASE_URL", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_urls = ["https://rpc.example.com"]
wallets = ["0x1234567890123456789012345678901234567890"]

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
		defer os.Unsetenv("DATABASE_URL")

		cfg, dbURL, err := LoadWithDefaults(configPath)
		require.NoError(t, err)

		assert.NotNil(t, cfg)
		assert.Equal(t, "postgres://user:pass@localhost:5432/db", dbURL)
	})

	t.Run("fails when DATABASE_URL is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_urls = ["https://rpc.example.com"]
wallets = ["0x1234567890123456789012345678901234567890"]

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		os.Unsetenv("DATABASE_URL")

		_, _, err = LoadWithDefaults(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DATABASE_URL is required")
	})

	t.Run("propagates config load errors", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
		defer os.Unsetenv("DATABASE_URL")

		// Invalid config path with no env vars
		_, _, err := LoadWithDefaults("/nonexistent/invalid.toml")
		assert.Error(t, err)
	})
}

func TestLoadDefaults(t *testing.T) {
	t.Run("defaults are applied", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_urls = ["https://rpc.example.com"]
wallets = ["0x1234567890123456789012345678901234567890"]

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := Load(configPath)
		require.NoError(t, err)

		// Check defaults
		assert.Equal(t, "info", cfg.LogLevel) // Default log level
		assert.Equal(t, 8080, cfg.HTTPPort)   // Default HTTP port
		assert.Equal(t, "UTC", cfg.Timezone)  // Default timezone
	})

	t.Run("explicit values override defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		configContent := `
rpc_urls = ["https://rpc.example.com"]
wallets = ["0x1234567890123456789012345678901234567890"]
log_level = "debug"
http_port = 9090
timezone = "America/New_York"

[[tokens]]
label = "TEST"
address = "0x0000000000000000000000000000000000000000"
fallback_decimals = 18
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, 9090, cfg.HTTPPort)
		assert.Equal(t, "America/New_York", cfg.Timezone)
	})
}
