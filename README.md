# RMM Tracker - Token Balance Monitor

A Go application that monitors ERC-20 token balances on Gnosis Chain and persists results to PostgreSQL. Specifically designed to track RealT RMM (Real Money Market) tokens including armmXDAI, armmUSDC, and their debt variants.

## Features

- 🚀 **Modern CLI**: Cobra-based command structure with subcommands
- 🔄 **Daemon Mode**: Optional continuous monitoring with configurable intervals
- ⚡ **High Performance**: pgx/v5 database driver (30-50% faster than lib/pq)
- 🔁 **Parallel Processing**: Concurrent token queries using goroutines
- 🔀 **RPC Failover**: Automatic failover between multiple RPC endpoints for high availability
- 🛡️ **Resilient**: Exponential backoff retry logic for transient failures
- 📦 **Batch Operations**: Efficient bulk inserts with transactions
- 📊 **Structured Logging**: JSON logs ready for ELK, Loki, or other aggregators
- ✅ **Declarative Validation**: Config validation with detailed error messages
- 🐳 **Docker Ready**: Includes Docker Compose for easy deployment
- 🌍 **Multi-Source Config**: Supports TOML, YAML, JSON, and environment variables

## Quick Start

### Prerequisites

- Go 1.26+ (for local development)
- PostgreSQL 18+ (or use Docker Compose)
- Gnosis Chain RPC endpoint

### Installation

```bash
# Clone repository
git clone https://github.com/matrixise/rmm-tracker.git
cd rmm-tracker

# Build
go build -o rmm-tracker .

# Or use Docker
docker compose build
```

### Configuration

Create `config.toml`:

```toml
# Multiple RPC endpoints for high availability (recommended)
rpc_urls = [
    "https://rpc.gnosischain.com",
    "https://gnosis.drpc.org",
    "https://rpc.ankr.com/gnosis"
]

# Or single endpoint (simpler, less resilient)
# rpc_url = "https://rpc.gnosischain.com"

wallets = [
    "0x1234567890123456789012345678901234567890",
    "0x2345678901234567890123456789012345678901"
]

[[tokens]]
label = "armmUSDC"
address = "0x59cd008D1f5e11Fb751370deDB02eC4fc96EAEaa"
fallback_decimals = 6

[[tokens]]
label = "armmXDAI"
address = "0xA0E6c16C5C8Cff4f9e9e8eC6f0e61eE8D8a8b8c2"
fallback_decimals = 18

# Optional
interval = "5m"        # Enable daemon mode
log_level = "info"     # debug, info, warn, error
```

Set DATABASE_URL:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/rmm_tracker?sslmode=disable"
```

### Usage

```bash
# Run once
DATABASE_URL="..." ./rmm-tracker run

# Run in daemon mode (every 5 minutes)
DATABASE_URL="..." ./rmm-tracker run --interval 5m

# Validate configuration
DATABASE_URL="..." ./rmm-tracker validate-config

# Check version
./rmm-tracker version

# Help
./rmm-tracker --help
./rmm-tracker run --help
```

### Docker Deployment

**Using Docker Compose:**
```bash
# Start PostgreSQL and run app once
docker compose up

# Run in daemon mode
docker compose up -d
docker compose logs -f app

# Stop services
docker compose down
```

**Using Task (recommended):**
```bash
task docker:up                 # Start all services
task docker:logs               # Follow application logs
task docker:down               # Stop all services
```

**Pull from Docker Hub:**
```bash
# Multi-arch images available (amd64 + arm64)
docker pull matrixise/rmm-tracker:latest
docker pull matrixise/rmm-tracker:v1.0.0  # Specific version
```

## Commands

### run
Run the token balance tracker once or in daemon mode.

**Flags:**
- `--config`: Config file path (default: ./config.toml)
- `--log-level`: Log level (debug, info, warn, error)
- `--interval`: Run interval for daemon mode (e.g., 5m, 1h)

**Examples:**
```bash
# Run once
rmm-tracker run

# Daemon mode
rmm-tracker run --interval 5m

# Debug logging
rmm-tracker run --log-level debug

# Custom config
rmm-tracker run --config production.toml
```

### validate-config
Validate configuration file without running the application.

**Example:**
```bash
DATABASE_URL="..." rmm-tracker validate-config --config config.toml
```

### version
Display version, git commit, and build time.

**Example:**
```bash
rmm-tracker version
```

## Configuration

### Environment Variables

**Recommended format (with RMM_TRACKER_ prefix):**
```bash
export DATABASE_URL="postgres://..."                    # Required
export RMM_TRACKER_RPC_URL="https://rpc.gnosischain.com"
export RMM_TRACKER_WALLETS="0xAddr1,0xAddr2"
export RMM_TRACKER_LOG_LEVEL="info"
export RMM_TRACKER_INTERVAL="5m"
```

**Legacy format (still supported):**
```bash
export RPC_URL="https://rpc.gnosischain.com"
export WALLETS="0xAddr1,0xAddr2"
export LOG_LEVEL="info"
```

See `.env.example` for a complete template.

### Configuration Precedence

1. Command-line flags (highest priority)
2. Environment variables with `RMM_TRACKER_` prefix
3. Environment variables without prefix (legacy)
4. Configuration file
5. Default values (lowest priority)

## Database Schema

The application automatically creates the `token_balances` table with optimized indexes:

```sql
CREATE TABLE token_balances (
    id            BIGSERIAL PRIMARY KEY,
    queried_at    TIMESTAMPTZ NOT NULL,
    wallet        TEXT NOT NULL,
    token_address TEXT NOT NULL,
    label         TEXT NOT NULL,
    symbol        TEXT NOT NULL,
    decimals      SMALLINT NOT NULL,
    raw_balance   TEXT NOT NULL,
    balance       TEXT NOT NULL
);

-- Optimized indexes
CREATE INDEX idx_token_balances_wallet_token_time
    ON token_balances(wallet, token_address, queried_at DESC);
CREATE INDEX idx_token_balances_queried_at
    ON token_balances(queried_at DESC);
CREATE INDEX idx_token_balances_wallet
    ON token_balances(wallet);
```

See `queries.sql` for example queries.

## Architecture

Modern modular architecture:

```
rmm-tracker/
├── cmd/                   # Cobra commands
│   ├── root.go
│   ├── run.go
│   ├── validate.go
│   └── version.go
├── internal/
│   ├── config/           # Viper config + validator
│   ├── storage/          # pgx database layer
│   ├── blockchain/       # Ethereum RPC client
│   └── logger/           # Structured logging
└── main.go               # Entry point
```

**Key Technologies:**
- CLI: cobra
- Config: viper (TOML, YAML, JSON)
- Database: pgx/v5 with connection pooling
- Validation: validator/v10
- Blockchain: go-ethereum
- Logging: log/slog (JSON)

## HTTP API

Available in daemon mode only (port 8080).

### Health

```
GET /health
```

Returns the overall health status of the application. HTTP 200 if healthy, 503 if any check is in error.

```json
{
  "status": "ok",
  "timestamp": "2025-01-01T00:00:00Z",
  "uptime": "2h34m5s",
  "build": {
    "version": "1.0.0",
    "git_commit": "abc1234",
    "build_time": "2025-01-01T00:00:00Z"
  },
  "checks": {
    "database":      { "status": "ok", "message": "database connection healthy" },
    "rpc_endpoints": { "status": "ok", "message": "all RPC endpoints healthy" },
    "daemon":        { "status": "ok", "message": "last executed 4m32s ago, next run in 28s" }
  }
}
```

Status values: `ok`, `degraded`, `error`.

### Balances

```
GET /api/v1/balances?wallet=0x...&symbol=armmUSDC&limit=100
```

Query parameters (all optional):
- `wallet`: filter by wallet address
- `symbol`: filter by token symbol
- `limit`: number of results (default: 100)

### Wallets

```
GET /api/v1/wallets
```

Returns the list of tracked wallet addresses.

### Weekly balances

```
GET /api/v1/wallets/{wallet}/balances/weekly
```

Returns one balance snapshot per week for the given wallet.

### Weekly report

```
GET /api/v1/wallets/{wallet}/report/weekly?weeks=4
```

Returns a week-over-week comparison report. `weeks` parameter: integer between 2 and 52 (default: 2).

## Performance

### pgx Benefits
- 30-50% faster than lib/pq
- Native PostgreSQL protocol
- Advanced connection pooling
- Batch API for bulk operations

### Optimization Features
- Parallel token queries per wallet
- Exponential backoff retry (3 attempts)
- Batch inserts with transactions
- Connection pooling (2-10 connections)

## Migration from v1

If upgrading from the original monolithic version, see [`docs/MIGRATION.md`](docs/MIGRATION.md) for:
- CLI command changes
- Environment variable updates
- Backward compatibility notes
- Rollback instructions

## Development

This project uses [Task](https://taskfile.dev/) for build automation. Run `task --list` to see all available tasks.

### Common Tasks

```bash
# Build
task build                     # Build binary with version info

# Testing
task test                      # Run all tests
task test:coverage:html        # Coverage report in browser

# Database
task migrate:up                # Apply migrations
task migrate:status            # Check migration state

# Docker (local)
task docker:up                 # Start PostgreSQL + app
task docker:logs               # Follow logs
task docker:buildx:build       # Build linux/amd64 image locally
task docker:buildx:push        # Build multi-arch (amd64+arm64) and push to Docker Hub
task docker:cache:clear        # Clear Docker build cache

# Development
task run                       # Run tracker once
task run:daemon                # Run in daemon mode (5m interval)
task clean                     # Remove build artifacts
```

### Multi-Architecture Docker Builds

The project supports building Docker images for multiple architectures using buildx:

```bash
# Build for local testing (linux/amd64 only, loaded into Docker)
task docker:buildx:build

# Build and push multi-arch images (linux/amd64 + linux/arm64)
task docker:buildx:push
```

**Build Cache Optimization:**
- Local builds use inline cache for faster rebuilds
- Push builds use Docker Hub registry cache (`:buildcache` tag)
- Cache is shared between local builds and CI/CD
- Go modules and build artifacts are cached in Docker layers

### Manual Build with Version Info

```bash
go build -ldflags "\
  -X github.com/matrixise/rmm-tracker/cmd.Version=1.0.0 \
  -X github.com/matrixise/rmm-tracker/cmd.GitCommit=$(git rev-parse HEAD) \
  -X github.com/matrixise/rmm-tracker/cmd.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o rmm-tracker .
```

## CI/CD

The project uses GitHub Actions for continuous integration and deployment:

- **Multi-arch Docker builds**: Automatically builds for `linux/amd64` and `linux/arm64`
- **Docker Hub publishing**: Pushes images on tags (`v*`) and main branch commits
- **Automatic README sync**: Docker Hub description is automatically updated from this README
- **Build cache**: Uses Docker Hub registry cache for faster builds

**Image tags:**
- `latest`: Latest commit on main branch
- `v1.0.0`: Semantic version tags
- `main`: Main branch (same as latest)
- `<git-sha>`: Specific commit SHA

## Documentation

- **[docs/CLAUDE.md](docs/CLAUDE.md)**: Architecture and development guide for Claude Code
- **[docs/MIGRATION.md](docs/MIGRATION.md)**: Migration guide from v1 to v2
- **[docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md)**: Performance improvements and optimization notes
- **[docs/SCHEDULER-IMPLEMENTATION.md](docs/SCHEDULER-IMPLEMENTATION.md)**: Scheduler implementation details
- **queries.sql**: Example database queries
- **.env.example**: Environment variable template

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]

## Support

For issues or questions, please open a GitHub issue.
