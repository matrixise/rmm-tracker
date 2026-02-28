# RMM Tracker

A Go application that monitors ERC-20 token balances on Gnosis Chain and persists results to PostgreSQL. Designed for RealT RMM (Real Money Market) tokens: armmXDAI, armmUSDC, and their debt variants.

## Features

- **Web UI**: Dashboard, wallet list with search, wallet detail with current balances
- **REST API**: JSON endpoints for balances, reports, and yield analytics
- **Daemon mode**: Clock-aligned scheduling (e.g. `5m` runs at :00, :05, :10…)
- **RPC failover**: Automatic failover between multiple Gnosis Chain RPC endpoints
- **Parallel processing**: Concurrent token queries per wallet using goroutines
- **Structured logging**: JSON logs compatible with ELK, Loki, and similar stacks
- **Docker ready**: Multi-arch images (amd64 + arm64) published to Docker Hub

## Quick Start

### Prerequisites

- Go 1.26+
- PostgreSQL 18+ (or Docker Compose)
- Gnosis Chain RPC endpoint

### Installation

```bash
git clone https://github.com/matrixise/rmm-tracker.git
cd rmm-tracker
task build
```

> `task build` injects version, git branch, commit, and build time into the binary via `-ldflags`.
> These values are exposed by `rmm-tracker version` and the `/health` endpoint.
> A plain `go build` will leave them empty.

### Configuration

Create `config.toml` from the provided example:

```bash
cp config.toml.example config.toml
```

Minimal configuration:

```toml
rpc_urls = [
    "https://rpc.gnosischain.com",
    "https://gnosis.drpc.org",
]

wallets = [
    "0x1234567890123456789012345678901234567890",
]

[[tokens]]
label = "armmUSDC"
address = "0x59cd008D1f5e11Fb751370deDB02eC4fc96EAEaa"
fallback_decimals = 6

[[tokens]]
label = "armmXDAI"
address = "0xA0E6c16C5C8Cff4f9e9e8eC6f0e61eE8D8a8b8c2"
fallback_decimals = 18

# Optional: enable daemon mode
interval = "5m"
```

Set the database URL:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/rmm_tracker?sslmode=disable"
```

### Usage

```bash
# Run once
DATABASE_URL="..." ./rmm-tracker run

# Daemon mode (every 5 minutes, clock-aligned)
DATABASE_URL="..." ./rmm-tracker run --interval 5m

# Validate configuration
DATABASE_URL="..." ./rmm-tracker validate-config

# Apply database migrations
./rmm-tracker migrate up

# Check version
./rmm-tracker version
```

### Docker

```bash
# Start PostgreSQL + app
docker compose up

# Daemon mode
docker compose up -d
docker compose logs -f app

# Pull from Docker Hub (multi-arch)
docker pull matrixise/rmm-tracker:latest
```

Or with Task:

```bash
task docker:up       # Start all services
task docker:logs     # Follow application logs
task docker:down     # Stop all services
```

## Web UI

Available in daemon mode at `http://localhost:8080`.

| Path | Description |
|---|---|
| `/` | Dashboard — global summary |
| `/wallets` | Wallet list with address search |
| `/wallets/{wallet}` | Wallet detail — current balances per token |

## REST API

All endpoints are prefixed with `/api/v1`.

### Dashboard

```http
GET /api/v1/dashboard
```

Global summary: total wallets, tokens tracked, latest run status.

### Balances

```http
GET /api/v1/balances?wallet=0x...&symbol=armmUSDC&limit=100
```

Historical balance records. All query parameters are optional.

```http
GET /api/v1/wallets/{wallet}/balances/latest
```

Latest balance for each token of a given wallet.

```http
GET /api/v1/wallets/{wallet}/balances/weekly
GET /api/v1/wallets/{wallet}/balances/daily
```

One balance snapshot per week or per day.

### Reports

```http
GET /api/v1/wallets/{wallet}/report/weekly?weeks=2
GET /api/v1/wallets/{wallet}/report/daily?days=31
```

Week-over-week or day-over-day comparison. `weeks`: 2–52 (default 2). `days`: 2–365 (default 31).

### Wallets

```http
GET /api/v1/wallets
```

List of tracked wallet addresses.

### Health

```http
GET /health
```

Returns HTTP 200 if healthy, 503 otherwise. Checks database connection, RPC endpoints, and scheduler status.

## Architecture

```text
rmm-tracker/
├── cmd/                   # Cobra commands (run, migrate, validate-config, version)
├── internal/
│   ├── api/               # REST API handlers + chi router
│   ├── blockchain/        # ERC-20 queries via go-ethereum, RPC failover
│   ├── config/            # Viper config loader + struct tag validation
│   ├── health/            # Health check endpoint
│   ├── logger/            # Structured logging (log/slog, JSON)
│   ├── scheduler/         # gocron v2, clock-aligned scheduling
│   ├── storage/           # pgx/v5, goose migrations (embedded SQL)
│   └── web/               # Web UI using templ templates
└── main.go
```

**Key technologies:**

| Concern | Library |
| --- | --- |
| CLI | cobra |
| Config | viper |
| Database | pgx/v5 + goose |
| HTTP | chi |
| Scheduler | gocron v2 |
| Templates | templ |
| Blockchain | go-ethereum |
| Decimals | shopspring/decimal |
| Validation | validator/v10 |

## Configuration Reference

### Environment Variables

```bash
DATABASE_URL="postgres://..."           # Required
RMM_TRACKER_RPC_URLS="url1,url2"
RMM_TRACKER_WALLETS="0xAddr1,0xAddr2"
RMM_TRACKER_INTERVAL="5m"
RMM_TRACKER_LOG_LEVEL="info"           # debug, info, warn, error
RMM_TRACKER_TIMEZONE="Europe/Brussels" # default: UTC
```

### Scheduling

The scheduler aligns to clock boundaries — `5m` runs at :00, :05, :10, not relative to startup.

Valid duration intervals: `1m`, `5m`, `10m`, `15m`, `20m`, `30m`, `1h`, `2h`, `3h`, `4h`, `6h`, `8h`, `12h`.

For non-standard schedules, use cron expressions:

```toml
interval = "0 9,17 * * 1-5"   # 9am and 5pm, weekdays only
interval = "*/7 * * * *"       # every 7 minutes (non-aligned)
```

## Development

This project uses [Task](https://taskfile.dev/). Run `task --list` for all available tasks.

```bash
task build                  # Build binary with version info
task test                   # Run unit tests
task test:coverage:html     # Coverage report in browser
task migrate:up             # Apply migrations
task migrate:status         # Check migration state
task docker:buildx:push     # Build multi-arch image and push to Docker Hub
```

## CI/CD

GitHub Actions workflows:

| Workflow | Trigger |
| --- | --- |
| **Test** (lint, test, integration) | Go/Taskfile changes on push or PR |
| **Markdown** (lint, spell check) | `.md` changes on PR |
| **Docker** | Push to `main` or `v*` tags |
| **Claude Code Review** | Every PR |
| **Claude Code** (`@claude`) | Comments mentioning `@claude` |

## Documentation

- [`docs/MIGRATION.md`](docs/MIGRATION.md) — upgrade guide between versions
- [`docs/SCHEDULER-IMPLEMENTATION.md`](docs/SCHEDULER-IMPLEMENTATION.md) — scheduling system details
- [`config.toml.example`](config.toml.example) — annotated configuration template
- [`queries.sql`](queries.sql) — example database queries

## License

[Add your license here]
