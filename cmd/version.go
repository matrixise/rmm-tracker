package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version information (set via ldflags at build time)
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display version, git commit, and build time information.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("rmm-tracker %s\n", Version)
		fmt.Printf("Commit: %s\n", GitCommit)
		fmt.Printf("Built: %s\n", BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
