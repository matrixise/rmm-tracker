# RMM Tracker - Token Balance Tracker

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

- Go 1.25+ (for local development)
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
export DATABASE_URL="postgres://user:pass@localhost:5432/realt_rmm?sslmode=disable"
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

```bash
# Start PostgreSQL and run app once
docker compose up

# Run in daemon mode
docker compose up -d
docker compose logs -f app

# Stop services
docker compose down
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

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run linter
golangci-lint run

# Build with version info
go build -ldflags "\
  -X github.com/matrixise/rmm-tracker/cmd.Version=1.0.0 \
  -X github.com/matrixise/rmm-tracker/cmd.GitCommit=$(git rev-parse HEAD) \
  -X github.com/matrixise/rmm-tracker/cmd.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o rmm-tracker .
```

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
