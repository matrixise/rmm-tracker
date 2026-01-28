package cmd

import (
	"log/slog"

	"github.com/matrixise/realt-rmm/internal/config"
	"github.com/matrixise/realt-rmm/internal/logger"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate-config",
	Short: "Validate configuration file",
	Long:  `Validate the configuration file syntax and values without running the application.`,
	RunE:  validateConfig,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func validateConfig(cmd *cobra.Command, args []string) error {
	// Setup logger
	logger.Setup(logLevel)

	// Load config
	cfg, databaseURL, err := config.LoadWithDefaults(cfgFile)
	if err != nil {
		slog.Error("Configuration validation failed", "error", err)
		return err
	}

	slog.Info("✓ Configuration valid",
		"wallets", len(cfg.Wallets),
		"tokens", len(cfg.Tokens),
		"rpc_url", cfg.RPCUrl,
		"interval", cfg.Interval,
		"log_level", cfg.LogLevel,
		"database_url_set", databaseURL != "",
	)

	return nil
}
