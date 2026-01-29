package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/matrixise/realt-rmm/internal/blockchain"
	"github.com/matrixise/realt-rmm/internal/config"
	"github.com/matrixise/realt-rmm/internal/health"
	"github.com/matrixise/realt-rmm/internal/logger"
	"github.com/matrixise/realt-rmm/internal/scheduler"
	"github.com/matrixise/realt-rmm/internal/storage"
	"github.com/spf13/cobra"
)

var (
	interval string
	once     bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the token balance tracker",
	Long:  `Query token balances from Gnosis Chain and persist results to PostgreSQL.`,
	RunE:  runTracker,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&interval, "interval", "", "run interval - duration (5m, 1h) or cron (\"*/5 * * * *\") - empty for one-time run")
	runCmd.Flags().BoolVar(&once, "once", false, "run once and exit (default)")
}

func runTracker(cmd *cobra.Command, args []string) error {
	// Setup logger (log-level from global flag)
	logger.Setup(logLevel)

	// Context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigChan
		slog.Info("Signal received, graceful shutdown", "signal", sig)
		cancel()
	}()

	// Load config
	cfg, databaseURL, err := config.LoadWithDefaults(cfgFile)
	if err != nil {
		slog.Error("Configuration error", "error", err)
		return err
	}

	// Override log level if set in config
	if cfg.LogLevel != "" {
		logger.Setup(cfg.LogLevel)
	}

	// Use interval from flag if provided, otherwise from config
	runInterval := interval
	if runInterval == "" && cfg.Interval != "" {
		runInterval = cfg.Interval
	}

	slog.Info("Configuration loaded",
		"config_path", cfgFile,
		"wallets", len(cfg.Wallets),
		"tokens", len(cfg.Tokens),
		"interval", runInterval,
	)

	// Connect to PostgreSQL
	store, err := storage.NewStore(ctx, databaseURL)
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		return err
	}
	defer store.Close()
	slog.Info("PostgreSQL connection established")

	// Create schema
	if err := store.CreateSchema(ctx); err != nil {
		slog.Error("Failed to create schema", "error", err)
		return err
	}

	// Connect to blockchain with failover support
	client, err := blockchain.NewClient(cfg.RPCUrls)
	if err != nil {
		slog.Error("Failed to connect to RPC", "error", err)
		return err
	}
	defer client.Close()

	if len(cfg.RPCUrls) == 1 {
		slog.Info("RPC connection established", "endpoint", cfg.RPCUrls[0])
	} else {
		slog.Info("RPC connection established with failover",
			"endpoints", len(cfg.RPCUrls),
			"primary", cfg.RPCUrls[0])
	}

	// Run mode: one-time or daemon
	if runInterval == "" || once {
		// Run once
		return processAllWallets(ctx, cfg, client, store)
	}

	// Daemon mode with scheduler
	slog.Info("Starting daemon mode with scheduler",
		"interval", runInterval,
		"timezone", cfg.GetTimezone().String(),
		"run_immediately", cfg.ShouldRunImmediately())

	// Create scheduler configuration
	schedulerCfg := scheduler.Config{
		Interval:       runInterval,
		Timezone:       cfg.GetTimezone(),
		RunImmediately: cfg.ShouldRunImmediately(),
		Logger:         slog.Default(),
	}

	// Create job function that tracks execution status
	var healthChecker *health.Checker
	jobFunc := func(jobCtx context.Context) error {
		err := processAllWallets(jobCtx, cfg, client, store)
		if healthChecker != nil {
			healthChecker.UpdateLastRun(err == nil)
		}
		return err
	}

	// Create scheduler
	sched, err := scheduler.NewScheduler(ctx, schedulerCfg, jobFunc)
	if err != nil {
		slog.Error("Failed to create scheduler", "error", err)
		return fmt.Errorf("scheduler creation failed: %w", err)
	}
	defer sched.Stop()

	// Determine expected interval for health checker
	expectedInterval, err := sched.GetExpectedInterval()
	if err != nil {
		// Fallback to conservative estimate for irregular cron expressions
		expectedInterval = 5 * time.Minute
		slog.Warn("Could not determine exact interval, using conservative estimate",
			"interval", expectedInterval)
	}

	// Create health checker with scheduler interface
	healthChecker = health.NewChecker(store, client, sched, expectedInterval)

	// Start health check server (daemon mode only)
	httpPort := cfg.HTTPPort
	if httpPort == 0 {
		httpPort = 8080 // Default port
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: http.HandlerFunc(healthChecker.Handler()),
	}

	go func() {
		slog.Info("Health check server starting", "port", httpPort, "endpoint", "/health")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Health server error", "error", err)
		}
	}()

	// Ensure HTTP server shutdown on exit
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Health server shutdown error", "error", err)
		}
	}()

	// Start the scheduler
	if err := sched.Start(); err != nil {
		slog.Error("Failed to start scheduler", "error", err)
		return fmt.Errorf("scheduler start failed: %w", err)
	}

	slog.Info("Daemon mode started with clock-aligned scheduling")

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Shutdown requested, stopping daemon")
	return nil
}

func processAllWallets(ctx context.Context, cfg *config.Config, client *blockchain.Client, store *storage.Store) error {
	for _, walletAddr := range cfg.Wallets {
		// Check for cancellation
		select {
		case <-ctx.Done():
			slog.Info("Shutdown requested, stopping processing")
			return ctx.Err()
		default:
		}

		wallet := common.HexToAddress(walletAddr)
		slog.Info("Processing wallet", "wallet", wallet.Hex())

		// Process tokens in parallel
		results := make(chan storage.TokenBalance, len(cfg.Tokens))
		var wg sync.WaitGroup

		for _, tok := range cfg.Tokens {
			if tok.Address == "" {
				slog.Warn("Token without address ignored", "label", tok.Label)
				continue
			}

			wg.Add(1)
			go func(token config.TokenConfig) {
				defer wg.Done()

				tokenInfo := blockchain.TokenInfo{
					Label:            token.Label,
					Address:          token.Address,
					FallbackDecimals: token.FallbackDecimals,
				}

				result, err := client.GetTokenBalance(ctx, wallet, tokenInfo)
				if err != nil {
					slog.Error("Token query error", "token_address", token.Address, "error", err)
					return
				}

				slog.Info("Balance retrieved",
					"wallet", result.Wallet,
					"symbol", result.Symbol,
					"balance", result.Balance,
					"decimals", result.Decimals,
				)

				results <- result
			}(tok)
		}

		// Wait and collect results
		go func() {
			wg.Wait()
			close(results)
		}()

		var successResults []storage.TokenBalance
		for result := range results {
			successResults = append(successResults, result)
		}

		// Batch insert
		if len(successResults) > 0 {
			if err := store.BatchInsertBalances(ctx, successResults); err != nil {
				slog.Error("Batch insert error", "error", err)
				continue
			}

			slog.Info("Records inserted successfully",
				"wallet", wallet.Hex(),
				"count", len(successResults),
			)
		}
	}

	slog.Info("Processing completed successfully")
	return nil
}
