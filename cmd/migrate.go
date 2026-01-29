package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/matrixise/rmm-tracker/internal/logger"
	"github.com/matrixise/rmm-tracker/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long:  `Run, rollback, or check the status of database migrations.`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	RunE:  runMigrateUp,
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the last migration",
	RunE:  runMigrateDown,
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE:  runMigrateStatus,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
}

func getDatabaseURL() (string, error) {
	v := viper.New()
	v.BindEnv("database_url", "DATABASE_URL")
	dsn := v.GetString("database_url")
	if dsn == "" {
		return "", fmt.Errorf("DATABASE_URL is required")
	}
	return dsn, nil
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	logger.Setup(logLevel)

	dsn, err := getDatabaseURL()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := storage.RunMigrations(ctx, dsn); err != nil {
		slog.Error("Migration failed", "error", err)
		return err
	}

	slog.Info("Migrations applied successfully")
	return nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	logger.Setup(logLevel)

	dsn, err := getDatabaseURL()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := storage.MigrateDown(ctx, dsn); err != nil {
		slog.Error("Rollback failed", "error", err)
		return err
	}

	slog.Info("Migration rolled back successfully")
	return nil
}

func runMigrateStatus(cmd *cobra.Command, args []string) error {
	logger.Setup(logLevel)

	dsn, err := getDatabaseURL()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := storage.MigrateStatus(ctx, dsn); err != nil {
		slog.Error("Failed to get migration status", "error", err)
		return err
	}

	return nil
}
