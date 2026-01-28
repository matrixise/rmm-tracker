package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 1. Set defaults
	v.SetDefault("log_level", "info")
	v.SetDefault("interval", "")    // Run once by default
	v.SetDefault("http_port", 8080)

	// 2. Configure config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("toml")
		v.AddConfigPath(".")
	}

	// 3. Environment variables
	v.SetEnvPrefix("REALT_RMM")
	v.AutomaticEnv()

	// Map environment variables to config keys
	// REALT_RMM_RPC_URL -> rpc_url
	v.BindEnv("rpc_url", "RPC_URL")
	v.BindEnv("wallets", "WALLETS")
	v.BindEnv("log_level", "LOG_LEVEL")
	v.BindEnv("interval", "INTERVAL")
	v.BindEnv("http_port", "HTTP_PORT")

	// 4. Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	// 5. Unmarshal into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Special handling for WALLETS env var (comma-separated)
	if walletsEnv := v.GetString("wallets"); walletsEnv != "" {
		// Check if it's a comma-separated string (from env var)
		if strings.Contains(walletsEnv, ",") {
			wallets := strings.Split(walletsEnv, ",")
			for i := range wallets {
				wallets[i] = strings.TrimSpace(wallets[i])
			}
			cfg.Wallets = wallets
		}
	}

	// 6. Validate with validator
	validate := NewValidator()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// LoadWithDefaults loads config with DATABASE_URL from environment
func LoadWithDefaults(configPath string) (*Config, string, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, "", err
	}

	// DATABASE_URL is required
	v := viper.New()
	v.BindEnv("database_url", "DATABASE_URL")
	databaseURL := v.GetString("database_url")

	if databaseURL == "" {
		return nil, "", fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, databaseURL, nil
}
