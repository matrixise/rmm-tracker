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
	"github.com/matrixise/rmm-tracker/internal/api"
	"github.com/matrixise/rmm-tracker/internal/blockchain"
	"github.com/matrixise/rmm-tracker/internal/config"
	"github.com/matrixise/rmm-tracker/internal/health"
	"github.com/matrixise/rmm-tracker/internal/logger"
	"github.com/matrixise/rmm-tracker/internal/scheduler"
	"github.com/matrixise/rmm-tracker/internal/storage"
	"github.com/spf13/cobra"
)

var (
	interval     string
	cronExpr     string
	enableHTTP   bool
	enableDaemon bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the token balance tracker",
	Long:  `Query token balances from Gnosis Chain and persist results to PostgreSQL.`,
	RunE:  runTracker,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&interval, "interval", "", "run interval as Go duration (5m, 1h, 6h) - clock-aligned")
	runCmd.Flags().StringVar(&cronExpr, "cron", "", "run interval as cron expression (\"*/5 * * * *\")")
	runCmd.Flags().BoolVar(&enableHTTP, "http", false, "start HTTP server (/health and API endpoints)")
	runCmd.Flags().BoolVar(&enableDaemon, "daemon", false, "start scheduler (requires --interval or --cron)")
}

func runTracker(cmd *cobra.Command, args []string) error {
	// Setup logger (log-level from global flag)
	logger.Setup(logLevel)

	// Validate mutually exclusive flags
	if interval != "" && cronExpr != "" {
		return fmt.Errorf("use either --interval or --cron, not both")
	}

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

	// Resolve effective run interval: flag > config
	runInterval := interval
	if runInterval == "" && cronExpr != "" {
		runInterval = cronExpr
	}
	if runInterval == "" && cfg.Interval != "" {
		runInterval = cfg.Interval
	}

	// --daemon requires an interval
	if enableDaemon && runInterval == "" {
		return fmt.Errorf("daemon mode requires --interval or --cron")
	}

	slog.Info("Configuration loaded",
		"config_path", cfgFile,
		"wallets", len(cfg.Wallets),
		"tokens", len(cfg.Tokens),
		"interval", runInterval,
	)

	// Run database migrations
	if err := storage.RunMigrations(ctx, databaseURL); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		return err
	}
	slog.Info("Database migrations applied")

	// Connect to PostgreSQL
	store, err := storage.NewStore(ctx, databaseURL)
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		return err
	}
	defer store.Close()
	slog.Info("PostgreSQL connection established")

	// One-shot mode: neither --http nor --daemon
	if !enableHTTP && !enableDaemon {
		client, err := blockchain.NewClient(cfg.RPCUrls)
		if err != nil {
			slog.Error("Failed to connect to RPC", "error", err)
			return err
		}
		defer client.Close()
		logRPCConnection(cfg.RPCUrls)
		return processAllWallets(ctx, cfg, client, store)
	}

	// Connect to blockchain only when daemon mode is active
	var client *blockchain.Client
	if enableDaemon {
		client, err = blockchain.NewClient(cfg.RPCUrls)
		if err != nil {
			slog.Error("Failed to connect to RPC", "error", err)
			return err
		}
		defer client.Close()
		logRPCConnection(cfg.RPCUrls)
	}

	buildInfo := health.BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
	}

	var healthChecker *health.Checker

	if enableDaemon {
		slog.Info("Starting daemon mode with scheduler",
			"interval", runInterval,
			"timezone", cfg.GetTimezone().String(),
			"run_immediately", cfg.ShouldRunImmediately())

		schedulerCfg := scheduler.Config{
			Interval:       runInterval,
			Timezone:       cfg.GetTimezone(),
			RunImmediately: cfg.ShouldRunImmediately(),
			Logger:         slog.Default(),
		}

		// jobFunc references healthChecker which is set after scheduler creation
		jobFunc := func(jobCtx context.Context) error {
			err := processAllWallets(jobCtx, cfg, client, store)
			if healthChecker != nil {
				healthChecker.UpdateLastRun(err == nil)
			}
			return err
		}

		sched, err := scheduler.NewScheduler(ctx, schedulerCfg, jobFunc)
		if err != nil {
			slog.Error("Failed to create scheduler", "error", err)
			return fmt.Errorf("scheduler creation failed: %w", err)
		}
		defer func() { _ = sched.Stop() }()

		expectedInterval, err := sched.GetExpectedInterval()
		if err != nil {
			expectedInterval = 5 * time.Minute
			slog.Warn("Could not determine exact interval, using conservative estimate",
				"interval", expectedInterval)
		}

		healthChecker = health.NewChecker(store, client, sched, expectedInterval, buildInfo)

		if err := sched.Start(); err != nil {
			slog.Error("Failed to start scheduler", "error", err)
			return fmt.Errorf("scheduler start failed: %w", err)
		}

		slog.Info("Daemon mode started with clock-aligned scheduling")
	}

	if enableHTTP && !enableDaemon {
		// HTTP-only mode: health checker without scheduler
		healthChecker = health.NewChecker(store, client, nil, 0, buildInfo)
	}

	if enableHTTP {
		httpPort := cfg.HTTPPort
		if httpPort == 0 {
			httpPort = 8080
		}

		apiHandler := api.NewHandler(store)
		router := api.NewRouter(healthChecker.Handler(), apiHandler)

		httpServer := &http.Server{
			Addr:              fmt.Sprintf(":%d", httpPort),
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		}

		go func() {
			slog.Info("HTTP server starting", "port", httpPort, "endpoint", "/health")
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTP server error", "error", err)
			}
		}()

		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				slog.Error("HTTP server shutdown error", "error", err)
			}
		}()
	}

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Shutdown requested, stopping")
	return nil
}

func logRPCConnection(rpcURLs []string) {
	if len(rpcURLs) == 1 {
		slog.Info("RPC connection established", "endpoint", rpcURLs[0])
	} else {
		slog.Info("RPC connection established with failover",
			"endpoints", len(rpcURLs),
			"primary", rpcURLs[0])
	}
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
					"balance", result.Balance.String(),
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
