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
	v.SetDefault("log_format", "text")
	v.SetDefault("interval", "") // Run once by default
	v.SetDefault("http_port", 8080)
	v.SetDefault("run_immediately", true)
	v.SetDefault("timezone", "UTC")

	// 2. Configure config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("toml")
		v.AddConfigPath(".")
	}

	// 3. Environment variables
	v.SetEnvPrefix("RMM_TRACKER")
	v.AutomaticEnv()

	// Map environment variables to config keys (RMM_TRACKER_* prefix is set above).
	// BindEnv only fails for an empty key, which is a programming error — panic is appropriate.
	for key, env := range map[string]string{
		"rpc_url":         "RPC_URL",
		"rpc_urls":        "RPC_URLS",
		"wallets":         "WALLETS",
		"log_level":       "LOG_LEVEL",
		"log_format":      "LOG_FORMAT",
		"interval":        "INTERVAL",
		"http_port":       "HTTP_PORT",
		"run_immediately": "RUN_IMMEDIATELY",
		"timezone":        "TIMEZONE",
	} {
		if err := v.BindEnv(key, env); err != nil {
			panic("config: bind env " + key + ": " + err.Error())
		}
	}

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

	// Special handling for comma-separated env vars
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

	// Parse comma-separated RPC_URLS env var
	if rpcURLsEnv := v.GetString("rpc_urls"); rpcURLsEnv != "" {
		if strings.Contains(rpcURLsEnv, ",") {
			urls := strings.Split(rpcURLsEnv, ",")
			for i := range urls {
				urls[i] = strings.TrimSpace(urls[i])
			}
			cfg.RPCUrls = urls
		}
	}

	// 6. Normalize: convert single rpc_url to rpc_urls array
	if err := cfg.Normalize(); err != nil {
		return nil, fmt.Errorf("config normalization failed: %w", err)
	}

	// 7. Validate with validator
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
	if err := v.BindEnv("database_url", "DATABASE_URL"); err != nil {
		panic("config: bind env database_url: " + err.Error())
	}
	databaseURL := v.GetString("database_url")

	if databaseURL == "" {
		return nil, "", fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, databaseURL, nil
}
