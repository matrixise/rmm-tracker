package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/matrixise/realt-rmm/internal/blockchain"
	"github.com/matrixise/realt-rmm/internal/config"
	"github.com/matrixise/realt-rmm/internal/logger"
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

	runCmd.Flags().StringVar(&interval, "interval", "", "run interval (e.g., 5m, 1h) - empty for one-time run")
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

	// Connect to blockchain
	client, err := blockchain.NewClient(cfg.RPCUrl)
	if err != nil {
		slog.Error("Failed to connect to RPC", "rpc_url", cfg.RPCUrl, "error", err)
		return err
	}
	defer client.Close()
	slog.Info("RPC connection established", "rpc_url", cfg.RPCUrl)

	// Run mode: one-time or daemon
	if runInterval == "" || once {
		// Run once
		return processAllWallets(ctx, cfg, client, store)
	}

	// Daemon mode with interval
	duration, err := time.ParseDuration(runInterval)
	if err != nil {
		slog.Error("Invalid interval", "interval", runInterval, "error", err)
		return err
	}

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	slog.Info("Daemon mode started", "interval", runInterval)

	// First run immediately
	if err := processAllWallets(ctx, cfg, client, store); err != nil {
		slog.Error("First execution error", "error", err)
	}

	// Then run on interval
	for {
		select {
		case <-ctx.Done():
			slog.Info("Shutdown requested, closing daemon")
			return nil
		case <-ticker.C:
			if err := processAllWallets(ctx, cfg, client, store); err != nil {
				slog.Error("Periodic execution error", "error", err)
			}
		}
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
