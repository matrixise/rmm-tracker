package cmd

import (
	"github.com/spf13/cobra"
)

var (
	cfgFile  string
	logLevel string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "rmm-tracker",
	Short: "RMM token balance tracker",
	Long: `rmm-tracker monitors ERC-20 token balances on Gnosis Chain and persists
results to PostgreSQL. It tracks RealT RMM (Real Money Market) tokens including
armmXDAI, armmUSDC, and their debt variants.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.toml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
}
