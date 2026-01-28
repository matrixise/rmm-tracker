# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RealT RMM is a Go application that queries ERC-20 token balances on Gnosis Chain and persists results to PostgreSQL. It monitors RealT RMM (Real Money Market) tokens including armmXDAI, armmUSDC, and their debt variants.

## Build & Run Commands

```bash
# Build
go build -o realt-rmm .

# Run once (requires DATABASE_URL env var)
DATABASE_URL="postgres://user:pass@localhost:5432/realt_rmm?sslmode=disable" ./realt-rmm run

# Run with custom config file
DATABASE_URL="..." ./realt-rmm run --config /path/to/config.toml

# Run in daemon mode (every 5 minutes)
DATABASE_URL="..." ./realt-rmm run --interval 5m

# Validate configuration
DATABASE_URL="..." ./realt-rmm validate-config

# Check version
./realt-rmm version

# View help
./realt-rmm --help
./realt-rmm run --help

# Download dependencies
go mod download

# Docker commands
docker compose build        # Build application image
docker compose up -d        # Start PostgreSQL and run app
docker compose up app       # Run app once (foreground)
docker compose logs app     # View application logs
```

## Configuration

The application uses multi-format configuration via viper (TOML, YAML, JSON):

### Config File Fields

```toml
# Required: RPC endpoints (use multiple for high availability)
rpc_urls = [
    "https://rpc.gnosischain.com",
    "https://gnosis.drpc.org",
    "https://rpc.ankr.com/gnosis"
]

# Or single endpoint (legacy, still supported)
# rpc_url = "https://rpc.gnosischain.com"

wallets = ["0x..."]

[[tokens]]
label = "armmUSDC"
address = "0x..."
fallback_decimals = 6

# Optional
interval = "5m"        # Enable daemon mode
log_level = "info"     # debug, info, warn, error
http_port = 8080       # Health check endpoint port (daemon mode only)
```

### Environment Variables (override config file)

**Recommended format with REALT_RMM_ prefix:**
- `REALT_RMM_RPC_URLS`: Comma-separated RPC endpoints (recommended)
- `REALT_RMM_RPC_URL`: Single RPC endpoint (legacy)
- `REALT_RMM_WALLETS`: Comma-separated wallet addresses
- `REALT_RMM_LOG_LEVEL`: Log level (debug, info, warn, error)
- `REALT_RMM_INTERVAL`: Daemon interval (e.g., "5m", "1h")
- `DATABASE_URL` (required): PostgreSQL connection string

**Legacy format (still supported):**
- `RPC_URL`, `RPC_URLS`, `WALLETS`, `LOG_LEVEL` (no prefix)

See `.env.example` for a template.

## Architecture

Modern modular architecture with cobra CLI, viper config, pgx database, and validator:

### Package Structure

```
realt-rmm/
├── main.go                 # Minimal entry point (calls cmd.Execute)
├── cmd/
│   ├── root.go            # Root cobra command
│   ├── run.go             # Run command (once or daemon)
│   ├── validate.go        # Config validation command
│   └── version.go         # Version command
├── internal/
│   ├── config/
│   │   ├── config.go      # Config structs with validator tags
│   │   └── loader.go      # Viper-based multi-source loader
│   ├── blockchain/
│   │   ├── client.go      # Ethereum RPC client wrapper
│   │   ├── erc20.go       # ERC20 token operations
│   │   └── failover.go    # RPC failover client
│   ├── storage/
│   │   ├── postgres.go    # pgx pool and operations
│   │   └── models.go      # Data models
│   ├── health/
│   │   └── health.go      # Health check endpoint (daemon mode)
│   └── logger/
│       └── logger.go      # Structured logging setup
└── go.mod
```

### Key Technologies

- **CLI**: `github.com/spf13/cobra` for subcommands
- **Config**: `github.com/spf13/viper` for multi-format config (TOML, YAML, JSON) with env var overrides
- **Database**: `github.com/jackc/pgx/v5` with connection pooling (30-50% faster than lib/pq)
- **Validation**: `github.com/go-playground/validator/v10` with custom Ethereum address validator
- **Blockchain**: `github.com/ethereum/go-ethereum` for RPC calls and ERC-20 ABI
- **Logging**: `log/slog` for structured JSON logs

### Core Features

- **Declarative validation**: Config structs with validation tags
- **Connection pooling**: pgxpool with configurable min/max connections
- **Parallelization**: Goroutines + channels for concurrent token queries per wallet
- **RPC Failover**: Automatic failover between multiple RPC endpoints with health tracking
- **Resilience**: Exponential backoff retry (3 attempts, 10s timeout)
- **Batch operations**: pgx.Batch API for efficient bulk inserts
- **Structured logging**: JSON logs with contextual metadata
- **Graceful shutdown**: Signal handling with context cancellation
- **Daemon mode**: Optional interval-based execution

The application iterates over configured wallets, queries tokens in parallel using goroutines, then batch-inserts results per wallet using transactions.

### Health Check Endpoint

In daemon mode, the application exposes an HTTP health check endpoint for monitoring:

**Endpoint:** `GET /health`

**Port:** Configurable via `http_port` in config (default: 8080)

**Response Format:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-28T22:30:00Z",
  "checks": {
    "database": {
      "status": "ok",
      "message": "database connection healthy"
    },
    "rpc_endpoints": {
      "status": "ok",
      "message": "all RPC endpoints healthy"
    },
    "daemon": {
      "status": "ok",
      "message": "last executed 45s ago"
    }
  },
  "uptime": "2h15m30s"
}
```

**Status Codes:**
- `200 OK`: All checks passed (status: "healthy")
- `503 Service Unavailable`: One or more critical checks failed (status: "unhealthy")

**Check Types:**
1. **Database**: Verifies PostgreSQL connection with ping (2s timeout)
2. **RPC Endpoints**: Checks at least one RPC endpoint is responding (3s timeout)
   - Status "ok": All endpoints healthy
   - Status "degraded": Some endpoints unhealthy but at least one working
   - Status "error": No healthy endpoints available
3. **Daemon** (daemon mode only): Verifies executions are running on schedule
   - Allows 2× interval grace period before marking degraded
   - Tracks last execution success/failure

**Docker Compose Integration:**
```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 30s
  timeout: 5s
  retries: 3
  start_period: 15s
```

The `-f` flag makes curl return a non-zero exit code on HTTP errors (4xx, 5xx), which Docker uses to mark the container as unhealthy.

**Note:** Health endpoint is only available in daemon mode (when `interval` is set). One-shot execution does not start the HTTP server.

### RPC Endpoint Failover

The application supports multiple RPC endpoints for high availability:

**Configuration:**
```toml
rpc_urls = [
    "https://rpc.gnosischain.com",    # Primary
    "https://gnosis.drpc.org",        # Backup 1
    "https://rpc.ankr.com/gnosis"     # Backup 2
]
```

**Behavior:**
- Automatically fails over to backup endpoints if primary fails
- Unhealthy endpoints are retried after 5-minute cooldown
- Transparent to the application logic - retries include failover attempts
- Logs all failover events for monitoring
- At least one healthy endpoint required at startup

**Backward Compatibility:**
Single `rpc_url` is still supported for simple deployments and automatically converted to `rpc_urls` array internally.

## CLI Commands

### run
Run the token balance tracker once or in daemon mode.

```bash
# Run once (default)
realt-rmm run

# Daemon mode with interval
realt-rmm run --interval 5m

# Custom config
realt-rmm run --config config.toml

# Debug logging
realt-rmm run --log-level debug
```

Flags:
- `--config`: Config file path (default: ./config.toml)
- `--log-level`: Log level (debug, info, warn, error)
- `--interval`: Run interval for daemon mode (e.g., 5m, 1h)
- `--once`: Run once and exit (redundant with default behavior)

### validate-config
Validate configuration without running the application.

```bash
realt-rmm validate-config --config config.toml
```

Useful for CI/CD pipelines and debugging configuration issues.

### version
Display version, git commit, and build time.

```bash
realt-rmm version
```

Build with version info:
```bash
go build -ldflags "-X github.com/matrixise/realt-rmm/cmd.Version=1.0.0 -X github.com/matrixise/realt-rmm/cmd.GitCommit=$(git rev-parse HEAD) -X github.com/matrixise/realt-rmm/cmd.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o realt-rmm .
```

## Quality Assurance

- **Linting**: Optional golangci-lint integration in Dockerfile (`.golangci.yml`)
- **Declarative validation**: Struct tags for automatic validation with clear error messages
- **Address validation**: Custom validator for Ethereum hex addresses
- **Error handling**: Structured errors with retry logic and exponential backoff
- **Observability**: JSON logs ready for ELK, Loki, or other log aggregators
- **Type safety**: Strongly-typed config structs with validation

## Database Schema & Indexes

The `token_balances` table is automatically created with optimized indexes:

```sql
CREATE TABLE token_balances (
    id            BIGSERIAL PRIMARY KEY,
    queried_at    TIMESTAMPTZ NOT NULL,
    wallet        TEXT NOT NULL,
    token_address TEXT NOT NULL,
    symbol        TEXT NOT NULL,
    decimals      SMALLINT NOT NULL,
    raw_balance   TEXT NOT NULL,
    balance       TEXT NOT NULL
);
```

**Indexes:**
- `idx_token_balances_wallet_token_time`: Composite index on `(wallet, token_address, queried_at DESC)` for historical queries
- `idx_token_balances_wallet`: Index on `wallet` for wallet-wide queries
- `idx_token_balances_queried_at`: Index on `queried_at DESC` for time-based queries

**Note:** The `label` field from configuration is used only for identification during setup. The database stores the on-chain `symbol` retrieved from the token contract.

See `queries.sql` for example queries that leverage these indexes.

## Performance Improvements

### pgx vs lib/pq

The upgrade to pgx/v5 provides significant benefits:
- **30-50% faster**: Native PostgreSQL protocol
- **Connection pooling**: Built-in pool with configurable limits
- **Batch API**: `pgx.Batch` for efficient bulk operations
- **Better cancellation**: Full context support
- **Active maintenance**: lib/pq is in maintenance mode

### Validator Benefits

Declarative validation with `github.com/go-playground/validator/v10`:
- Clear struct tags instead of manual validation code
- Detailed error messages with field context
- Custom validators for domain-specific rules (Ethereum addresses, durations)
- Automatic validation on config load

## Migration

For users upgrading from v1, see `MIGRATION.md` for:
- Breaking changes in CLI interface
- New environment variable format (with REALT_RMM_ prefix)
- Backward compatibility notes
- Rollback instructions

## Reference Implementation

`steph.py` contains the original Python implementation using web3.py. The Go version adds:
- PostgreSQL persistence with optimized connection pooling
- Multi-wallet support via external config
- Parallel RPC calls with goroutines
- Retry logic with exponential backoff
- Batch database operations with pgx
- Modern CLI with cobra subcommands
- Multi-format configuration with viper
- Declarative validation with validator
- Daemon mode for continuous monitoring
