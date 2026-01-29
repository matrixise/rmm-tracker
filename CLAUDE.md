# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

rmm-tracker is a Go application that queries ERC-20 token balances on Gnosis Chain and persists results to PostgreSQL. It monitors RealT RMM (Real Money Market) tokens including armmXDAI, armmUSDC, and their debt variants.

## Build & Run Commands

The project uses [Taskfile](https://taskfile.dev/) as task runner. Run `task --list` to see all available tasks.

```bash
# Build & quality
task build                  # Build binary with version info
task test                   # Run all tests
task test:verbose           # Run tests with verbose output
task test:coverage          # Run tests with coverage report
task test:coverage:html     # Open coverage in browser
task vet                    # Run go vet
task clean                  # Remove build artifacts

# Database migrations
task migrate:up             # Apply pending migrations
task migrate:down           # Rollback last migration
task migrate:status         # Show migration status

# Docker
task docker:build           # Build Docker image
task docker:up              # Start PostgreSQL + app
task docker:down            # Stop all services
task docker:logs            # Follow app logs
task docker:ps              # Show container status

# Run
task run                    # Run tracker once
task run:daemon             # Run in daemon mode (5m)
```

### Direct commands (without Taskfile)

```bash
# Build
go build -o rmm-tracker .

# Run once (requires DATABASE_URL env var)
DATABASE_URL="postgres://user:pass@localhost:5432/realt_rmm?sslmode=disable" ./rmm-tracker run

# Run with custom config file
DATABASE_URL="..." ./rmm-tracker run --config /path/to/config.toml

# Run in daemon mode (every 5 minutes)
DATABASE_URL="..." ./rmm-tracker run --interval 5m

# Validate configuration
DATABASE_URL="..." ./rmm-tracker validate-config

# Check version
./rmm-tracker version

# View help
./rmm-tracker --help
./rmm-tracker run --help

# Download dependencies
go mod download
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

# Optional: Scheduler configuration
# Option 1: Duration (automatically converted to clock-aligned cron)
interval = "5m"        # Runs at :00, :05, :10, :15, :20, :25, etc.

# Option 2: Cron expression for advanced scheduling
# interval = "*/5 * * * *"      # Every 5 minutes
# interval = "0 */2 * * *"      # Every 2 hours at :00
# interval = "0 9,17 * * 1-5"   # 9am and 5pm on weekdays

# Scheduler options
run_immediately = true  # Execute immediately on startup (default: true)
timezone = "UTC"        # Timezone for scheduling (default: UTC)

# Other options
log_level = "info"      # debug, info, warn, error
http_port = 8080        # Health check endpoint port (daemon mode only)
```

### Environment Variables (override config file)

**Recommended format with RMM_TRACKER_ prefix:**
- `RMM_TRACKER_RPC_URLS`: Comma-separated RPC endpoints (recommended)
- `RMM_TRACKER_RPC_URL`: Single RPC endpoint (legacy)
- `RMM_TRACKER_WALLETS`: Comma-separated wallet addresses
- `RMM_TRACKER_LOG_LEVEL`: Log level (debug, info, warn, error)
- `RMM_TRACKER_INTERVAL`: Schedule interval - duration (e.g., "5m", "1h") or cron expression (e.g., "*/5 * * * *")
- `RMM_TRACKER_RUN_IMMEDIATELY`: Execute immediately on startup (true/false, default: true)
- `RMM_TRACKER_TIMEZONE`: Timezone for scheduling (e.g., "UTC", "America/New_York")
- `DATABASE_URL` (required): PostgreSQL connection string

**Legacy format (still supported):**
- `RPC_URL`, `RPC_URLS`, `WALLETS`, `LOG_LEVEL` (no prefix)

See `.env.example` for a template.

## Architecture

Modern modular architecture with cobra CLI, viper config, pgx database, and validator:

### Package Structure

```
rmm-tracker/
├── main.go                 # Minimal entry point (calls cmd.Execute)
├── Taskfile.yml            # Task runner (build, test, docker, migrate)
├── cmd/
│   ├── root.go            # Root cobra command
│   ├── run.go             # Run command (once or daemon)
│   ├── migrate.go         # Database migration commands
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
│   │   ├── migrate.go     # Goose migration runner (embedded SQL)
│   │   ├── migrations/    # SQL migration files
│   │   │   └── 001_create_token_balances.sql
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
- **Migrations**: `github.com/pressly/goose/v3` for versioned SQL migrations (embedded in binary)
- **Testing**: `github.com/stretchr/testify` for assertions (`assert`, `require`)
- **Task runner**: [Taskfile](https://taskfile.dev/) for development workflows (`Taskfile.yml`)
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
- **Clock-aligned scheduling**: gocron v2 for precise, predictable execution timing
- **Daemon mode**: Optional interval-based or cron-based scheduling

The application iterates over configured wallets, queries tokens in parallel using goroutines, then batch-inserts results per wallet using transactions.

### Scheduling System

The daemon mode uses **gocron v2** for clock-aligned scheduling:

**Duration-based scheduling (automatic clock alignment):**
- `5m` → Executes at :00, :05, :10, :15, :20, etc. (not relative to startup)
- `1h` → Executes at :00 every hour
- `30m` → Executes at :00 and :30 every hour

**Cron expression support:**
- `"*/5 * * * *"` → Every 5 minutes at clock boundaries
- `"0 */2 * * *"` → Every 2 hours at :00
- `"0 9,17 * * 1-5"` → 9am and 5pm on weekdays
- `"30 */6 * * *"` → Every 6 hours at :30

**Features:**
- **Clock alignment**: Duration intervals like `5m` automatically align to clock boundaries (not relative to container start)
- **Timezone support**: Configure timezone for cron expressions (default: UTC)
- **Run immediately**: Optional immediate execution on startup (default: enabled)
- **Validation**: Only accepts durations that divide evenly into 60 (minutes) or 24 (hours)
- **Health monitoring**: Health endpoint shows next scheduled run time

**Valid intervals:** 1m, 5m, 10m, 15m, 20m, 30m, 1h, 2h, 3h, 4h, 6h, 8h, 12h, 24h

**Invalid intervals:** 7m, 13m, 45m, 5h, 7h (use cron expressions for non-standard intervals)

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
rmm-tracker run

# Daemon mode with interval
rmm-tracker run --interval 5m

# Custom config
rmm-tracker run --config config.toml

# Debug logging
rmm-tracker run --log-level debug
```

Flags:
- `--config`: Config file path (default: ./config.toml)
- `--log-level`: Log level (debug, info, warn, error)
- `--interval`: Run interval for daemon mode (e.g., 5m, 1h)
- `--once`: Run once and exit (redundant with default behavior)

### migrate
Manage database migrations manually.

```bash
# Apply all pending migrations
DATABASE_URL="..." rmm-tracker migrate up

# Rollback the last migration
DATABASE_URL="..." rmm-tracker migrate down

# Show migration status
DATABASE_URL="..." rmm-tracker migrate status
```

**Note:** Migrations are also applied automatically on `rmm-tracker run` startup. The `migrate` command is useful for manual management without running the tracker.

### validate-config
Validate configuration without running the application.

```bash
rmm-tracker validate-config --config config.toml
```

Useful for CI/CD pipelines and debugging configuration issues.

### version
Display version, git commit, and build time.

```bash
rmm-tracker version
```

Build with version info:
```bash
go build -ldflags "-X github.com/matrixise/rmm-tracker/cmd.Version=1.0.0 -X github.com/matrixise/rmm-tracker/cmd.GitCommit=$(git rev-parse HEAD) -X github.com/matrixise/rmm-tracker/cmd.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o rmm-tracker .
```

## Quality Assurance

- **Linting**: Optional golangci-lint integration in Dockerfile (`.golangci.yml`)
- **Declarative validation**: Struct tags for automatic validation with clear error messages
- **Address validation**: Custom validator for Ethereum hex addresses
- **Error handling**: Structured errors with retry logic and exponential backoff
- **Observability**: JSON logs ready for ELK, Loki, or other log aggregators
- **Type safety**: Strongly-typed config structs with validation

## Database Schema & Migrations

The database schema is managed by [goose](https://github.com/pressly/goose) versioned migrations. Migration SQL files are embedded in the binary via `go:embed`, so no external files are needed at runtime.

Migrations run automatically on `rmm-tracker run` startup, or manually via `rmm-tracker migrate up`. Goose tracks applied migrations in a `goose_db_version` table.

The initial migration uses `IF NOT EXISTS` for backward compatibility with existing databases.

### token_balances table

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
- New environment variable format (with RMM_TRACKER_ prefix)
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
