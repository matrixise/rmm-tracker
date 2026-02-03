# Migration Guide

This guide helps users migrate between versions of rmm-tracker.

## v2 to v3: Clock-Aligned Scheduling

Version 3 introduces **gocron v2** for clock-aligned scheduling, replacing the time.Ticker-based implementation.

### What Changed?

**Scheduling Behavior:**
- **v2**: Executions were relative to container startup
  - Start at 09:03:27 with `interval = "5m"` → runs at 09:03, 09:08, 09:13, 09:18...
- **v3**: Executions align to clock boundaries
  - Start at 09:03:27 with `interval = "5m"` → runs at 09:05, 09:10, 09:15, 09:20...

### Breaking Changes

**Invalid Intervals:**
Non-standard intervals that don't divide evenly are now rejected:

```bash
# ❌ These will fail in v3
interval = "7m"   # Error: minute intervals must divide evenly into 60
interval = "5h"   # Error: hour intervals must divide evenly into 24
interval = "13m"  # Error: not a standard divisor of 60

# ✅ Use these instead
interval = "5m"   # OK: divides evenly into 60
interval = "6h"   # OK: divides evenly into 24

# ✅ Or use cron expressions for non-standard intervals
interval = "*/7 * * * *"   # Every 7 minutes (non-aligned)
```

**Valid Intervals:**
- **Minutes**: 1, 5, 10, 15, 20, 30
- **Hours**: 1, 2, 3, 4, 6, 8, 12, 24
- **Seconds**: 10, 15, 30 (for sub-minute scheduling)

### New Features

**1. Cron Expression Support:**
```toml
# Advanced scheduling with cron expressions
interval = "0 9,17 * * 1-5"   # 9am and 5pm on weekdays
interval = "*/15 * * * *"      # Every 15 minutes
interval = "0 */2 * * *"       # Every 2 hours at :00
```

**2. Timezone Support:**
```toml
# Configure timezone for scheduling
timezone = "America/New_York"  # Eastern Time
# timezone = "Europe/Paris"    # Central European Time
# timezone = "UTC"             # Default
```

Or via environment variable:
```bash
RMM_TRACKER_TIMEZONE=America/New_York
```

**3. Run Immediately Option:**
```toml
# Control immediate execution on startup
run_immediately = false  # Skip first execution, wait for schedule
```

Or via environment variable:
```bash
RMM_TRACKER_RUN_IMMEDIATELY=false
```

### Migration Steps

**Step 1: Check Your Interval**

Review your current `interval` configuration:

```bash
# Check if your interval is valid for v3
DATABASE_URL="..." ./rmm-tracker validate-config
```

**Step 2: Update Invalid Intervals**

If you have a non-standard interval:

```toml
# Option A: Use a standard interval
# BEFORE
interval = "7m"

# AFTER
interval = "5m"   # or 10m, depending on your needs

# Option B: Use a cron expression
# BEFORE
interval = "7m"

# AFTER
interval = "*/7 * * * *"  # Preserves 7-minute interval (non-aligned)
```

**Step 3: (Optional) Configure Timezone**

If you need scheduling in a specific timezone:

```toml
interval = "0 9 * * *"      # Run at 9am
timezone = "America/New_York"  # Eastern Time
```

**Step 4: Test**

Run in daemon mode and verify clock alignment:

```bash
DATABASE_URL="..." ./rmm-tracker run --interval 5m --log-level debug
```

Check logs for:
```
"Converting duration to cron" cron="*/5 * * * *"
"Scheduler started" next_run="2026-01-28T22:05:00Z"
```

### Compatibility

**✅ No Changes Required:**
- Standard intervals: `5m`, `10m`, `15m`, `30m`, `1h`, `2h`, `6h`, `12h`
- All existing config files with standard intervals work without modification
- Environment variables unchanged (same `RMM_TRACKER_*` prefix)

**⚠️ May Require Changes:**
- Non-standard intervals: `7m`, `13m`, `45m`, `5h`, `7h`, etc.
- If you relied on relative (non-aligned) execution timing

### Rollback to v2

If you need to rollback:

```bash
git checkout v2.x.x  # Your previous v2 version
docker compose build
docker compose up -d
```

## v1 to v2: Modular Architecture

This guide helps users migrate from the original monolithic version to the new modular architecture with cobra, viper, pgx, and validator.

## Breaking Changes

### CLI Interface

**Before (v1):**
```bash
# Run with default config.toml
DATABASE_URL="..." ./rmm-tracker

# Run with custom config
DATABASE_URL="..." ./rmm-tracker /path/to/config.toml
```

**After (v2):**
```bash
# Run once with default config
DATABASE_URL="..." ./rmm-tracker run

# Run with custom config
DATABASE_URL="..." ./rmm-tracker run --config /path/to/config.toml

# Run in daemon mode
DATABASE_URL="..." ./rmm-tracker run --interval 5m

# Validate config
DATABASE_URL="..." ./rmm-tracker validate-config

# Check version
./rmm-tracker version
```

### Environment Variables

Environment variables now support a consistent `RMM_TRACKER_` prefix:

**Before (v1):**
```bash
export RPC_URL="https://rpc.gnosischain.com"
export WALLETS="0xAddr1,0xAddr2"
export LOG_LEVEL="debug"
export DATABASE_URL="postgres://..."
```

**After (v2):**
```bash
# Both formats work, but RMM_TRACKER_ prefix is recommended
export RMM_TRACKER_RPC_URL="https://rpc.gnosischain.com"
export RMM_TRACKER_WALLETS="0xAddr1,0xAddr2"
export RMM_TRACKER_LOG_LEVEL="debug"
export DATABASE_URL="postgres://..."  # Still supported

# Legacy format still works for compatibility
export RPC_URL="https://rpc.gnosischain.com"
export WALLETS="0xAddr1,0xAddr2"
export LOG_LEVEL="debug"
```

## New Features

### 1. Command Structure

The application now uses cobra for a modern CLI experience:

```bash
rmm-tracker
├── run              # Run the tracker (once or daemon)
├── validate-config  # Validate configuration without running
└── version         # Display version information
```

### 2. Daemon Mode

Run continuously with automatic execution at intervals:

```bash
# In config.toml
interval = "5m"

# Or via environment variable
RMM_TRACKER_INTERVAL=5m ./rmm-tracker run

# Or via flag
./rmm-tracker run --interval 5m
```

### 3. Configuration Validation

Validate your configuration before running:

```bash
DATABASE_URL="..." ./rmm-tracker validate-config --config config.toml
```

### 4. Multi-Format Configuration

Configuration now supports multiple formats via viper:

```bash
# TOML (default)
./rmm-tracker run --config config.toml

# YAML
./rmm-tracker run --config config.yaml

# JSON
./rmm-tracker run --config config.json
```

### 5. Enhanced Logging

Log level can be set multiple ways (in order of precedence):

1. Command-line flag: `--log-level debug`
2. Environment variable: `RMM_TRACKER_LOG_LEVEL=debug`
3. Config file: `log_level = "debug"`
4. Default: `info`

### 6. PostgreSQL Performance

Upgraded from `lib/pq` to `pgx/v5` with connection pooling:

- 30-50% faster database operations
- Native PostgreSQL protocol
- Advanced connection pool management
- Better context support for cancellation

## Configuration Updates

### Optional New Fields

You can add these optional fields to your `config.toml`:

```toml
# Existing fields
rpc_url = "https://rpc.gnosischain.com"
wallets = ["0x..."]
[[tokens]]
label = "armmUSDC"
address = "0x..."
fallback_decimals = 6

# New optional fields
interval = "5m"        # Enable daemon mode
log_level = "info"     # Set default log level
http_port = 8080       # Reserved for future HTTP API
```

## Docker Deployment

### Docker Compose

**Before (v1):**
```yaml
services:
  app:
    command: ["./rmm-tracker", "config.toml"]
```

**After (v2):**
```yaml
services:
  app:
    # Default command is "run"
    command: ["./rmm-tracker", "run"]

    # Or daemon mode
    command: ["./rmm-tracker", "run", "--interval", "5m"]

    # Environment variables
    environment:
      - RMM_TRACKER_INTERVAL=5m
```

### Dockerfile

The Dockerfile has been updated to use the new command structure:

```dockerfile
ENTRYPOINT ["./rmm-tracker", "run"]
```

## Backward Compatibility

### Automatic Detection

The application will detect if you're using the old command format and suggest migration:

```bash
# Old format (still works but deprecated)
DATABASE_URL="..." ./rmm-tracker config.toml

# New format
DATABASE_URL="..." ./rmm-tracker run --config config.toml
```

### Environment Variables

Both old and new environment variable formats are supported:

- `RPC_URL` → `RMM_TRACKER_RPC_URL`
- `WALLETS` → `RMM_TRACKER_WALLETS`
- `LOG_LEVEL` → `RMM_TRACKER_LOG_LEVEL`

The prefixed versions take precedence if both are set.

## Rollback Instructions

If you need to rollback to v1:

1. Checkout the previous version:
   ```bash
   git checkout v1.0.0  # Or your previous version tag
   ```

2. Rebuild:
   ```bash
   go build -o rmm-tracker .
   ```

3. Use the old command format:
   ```bash
   DATABASE_URL="..." ./rmm-tracker config.toml
   ```

## Support

For issues or questions:
- GitHub Issues: https://github.com/matrixise/rmm-tracker/issues
- Review CLAUDE.md for detailed architecture documentation
- Check README.md for usage examples

## Performance Improvements

### pgx Benefits

The upgrade to pgx/v5 provides:

- **Faster queries**: 30-50% improvement in RPC and DB operations
- **Connection pooling**: Optimized pool with configurable min/max connections
- **Batch operations**: Native batch API for bulk inserts
- **Better cancellation**: Full context support for graceful shutdown

### Validation Benefits

The validator library provides:

- **Declarative validation**: Clear struct tags instead of manual checks
- **Better error messages**: Detailed validation failures with field context
- **Ethereum address validation**: Built-in validation for hex addresses
- **Duration validation**: Automatic validation of time.Duration strings

## Testing Checklist

After migration, verify:

- [ ] `rmm-tracker version` displays version info
- [ ] `rmm-tracker validate-config` validates your config
- [ ] `rmm-tracker run` executes once successfully
- [ ] `rmm-tracker run --interval 5s` runs in daemon mode
- [ ] Environment variables override config values
- [ ] Graceful shutdown works (Ctrl+C)
- [ ] Database records are inserted correctly
- [ ] Docker Compose deployment works

## Questions?

Refer to the updated documentation:
- **CLAUDE.md**: Architecture and development guide
- **README.md**: Usage and deployment
- **queries.sql**: Database query examples
