# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

rmm-tracker monitors ERC-20 token balances on Gnosis Chain and stores them in PostgreSQL. Built in Go with Cobra CLI, it tracks RealT RMM tokens (armmXDAI, armmUSDC, and debt variants).

## Essential Commands

Uses [Taskfile](https://taskfile.dev/) - run `task --list` for all tasks.

```bash
# Development
task build                  # Build with version info
task test                   # Run tests
task test:coverage:html     # Coverage report in browser

# Database
task migrate:up             # Apply migrations
task migrate:status         # Check migration state

# Docker
task docker:up              # PostgreSQL + app
task docker:logs            # Follow logs

# Run
DATABASE_URL="postgres://..." ./rmm-tracker run                # Once
DATABASE_URL="postgres://..." ./rmm-tracker run --interval 5m  # Daemon mode
```

## Architecture

**Entry point:** `main.go` → `cmd.Execute()`

**Core packages:**
- `cmd/` - Cobra commands (run, migrate, validate-config, version)
- `internal/config/` - Viper config loader + validator tags
- `internal/blockchain/` - ERC20 queries via go-ethereum + RPC failover
- `internal/storage/` - pgx connection pool + goose migrations (embedded SQL)
- `internal/health/` - HTTP health endpoint for daemon mode

**Data flow:**
1. Load config (TOML/YAML/JSON) + env vars → validate with struct tags
2. For each wallet: query all tokens in parallel (goroutines + channels)
3. Batch insert per wallet using pgx transactions

**Key design choices:**
- **pgx/v5** instead of lib/pq (30-50% faster, better pooling)
- **Declarative validation** via struct tags (github.com/go-playground/validator)
- **RPC failover** for resilience (automatic backup endpoint switching)
- **Clock-aligned scheduling** with gocron v2 (5m → :00, :05, :10, not relative to start)

## Configuration

`config.toml` is gitignored. Use `config.toml.example` as template.

Environment variables override config:
- `DATABASE_URL` (required)
- `RMM_TRACKER_RPC_URLS` (comma-separated)
- `RMM_TRACKER_WALLETS` (comma-separated)
- `RMM_TRACKER_INTERVAL` (duration like "5m" or cron expression)

## Database

Migrations in `internal/storage/migrations/*.sql` are embedded via `go:embed`.

Auto-run on `rmm-tracker run` startup or manual via `rmm-tracker migrate up`.

Schema: `token_balances` table with composite indexes for historical queries.

## Testing

Tests use `github.com/stretchr/testify` for assertions.

Run single test:
```bash
go test ./internal/config -run TestLoadConfig
```

## Docker

Multi-arch builds (AMD64 + ARM64) via buildx:
```bash
task docker:buildx:push  # Build and push to Docker Hub
```

Health check endpoint: `GET /health` (daemon mode only, port 8080).

## Important Notes

- Scheduler uses **clock alignment**: `5m` interval runs at :00, :05, :10 (not relative to startup)
- Only durations dividing evenly into 60min or 24h are valid (e.g., 1m, 5m, 15m, 1h, 6h)
- For custom schedules, use cron expressions: `"0 9,17 * * 1-5"` (9am & 5pm weekdays)
- RPC failover is transparent: retries automatically cycle through healthy endpoints
- Migrations use `IF NOT EXISTS` for backward compatibility

## Related Documentation

- `README.md` - User-facing quick start
- `docs/MIGRATION.md` - v1 to v2 upgrade guide
- `docs/SCHEDULER-IMPLEMENTATION.md` - Scheduling system details
- `queries.sql` - Example database queries with indexes
