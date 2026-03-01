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

## Taskfile Conventions

Strict rules to follow when creating or modifying `Taskfile.yml`.

### YAML

- **Never use `:` in a `desc:` value** without double quotes — it breaks YAML parsing.
  - Wrong    : `desc: DESTRUCTIVE: Drop the local database`
  - Quoted   : `desc: "DESTRUCTIVE: Drop the local database"`
  - Preferred: `desc: Drop the local database (DESTRUCTIVE)`

### Go template vs shell

- **Shell variables** : double the `$` to prevent interpretation by Taskfile's Go template engine.
  - Wrong  : `[ "$confirm" = "yes" ]`
  - Correct: `[ "$$confirm" = "yes" ]`

- **Docker `--format`** : `{{.Xxx}}` strings in shell commands conflict with the Go template engine. Escape them with backticks.
  - Wrong  : `docker ps --format '{{.Names}}'`
  - Correct: `docker ps --format '{{` + "`{{.Names}}`" + `}}'`

### Block scalars for complex commands

- **`{` and `}` in a plain YAML scalar** cause a parse error (YAML interprets `{` as a flow mapping start). Any shell command containing braces (`{ cmd; }`, subshells, etc.) must use a `|` block scalar.
  - Wrong  : `- cmd && fallback || { echo "err" && exit 1; }`
  - Correct:
    ```yaml
    - |
      cmd && fallback || { echo "err" && exit 1; }
    ```

- **Long lines** : any shell command exceeding ~160 characters must be split with `\` inside a `|` block scalar. Never write a command longer than 200 characters on a single YAML line.
  - Always use `|` + `\` for `go build -ldflags`, `docker buildx build`, `ssh ... | docker exec ...`, etc.

- **Validation** : always run both tools in order after any modification — yamllint catches YAML syntax issues, `task --list` catches Go template errors that yamllint cannot see:
  ```bash
  uv tool run yamllint Taskfile.yml  # YAML syntax (Python parser)
  task --list                         # Go template + Taskfile semantics
  ```
  These two parsers are not equivalent. yamllint uses PyYAML and knows nothing about Taskfile's `text/template` layer. Errors like unescaped `{{.Names}}` or wrong `$$var` usage are invisible to yamllint but fatal to Task. The project's `.yamllint.yml` sets the line-length limit to 200 characters.

### Shell portability

- **`read -p`** is bash-only and not POSIX portable. Use `printf` + `read` separately.
  - Wrong  : `read -p "Confirm? " confirm`
  - Correct: `printf "Confirm? " && read confirm`

- **`&&` / `||` precedence** : `A && B || C` triggers `C` if `A` **or** `B` fails, not only when `B` fails. For confirmation prompts, separate `read` with `;` and group the fallback with `{ }`.
  - Wrong  : `printf "..." && read confirm && [ "$$confirm" = "yes" ] || exit 1`
  - Correct: `printf "..." && read confirm; [ "$$confirm" = "yes" ] || { echo "Aborted." && exit 1; }`

### vars with `sh:`

- Vars defined with `sh:` are evaluated **once** at task startup, not before each `cmd`. Do not assume they are re-evaluated dynamically.

## Pre-commit hooks (prek)

This project uses [prek](https://github.com/isak-larsson/prek), a Rust-based drop-in replacement for `pre-commit`. It is **not** `pre-commit` — always use `prek` directly.

```bash
prek install        # Install hooks into .git/hooks/
prek run -a         # Run all hooks on all files
prek run <hook-id>  # Run a specific hook
```

The configuration lives in `.pre-commit-config.yaml` (prek reads this format natively). Hooks include: trailing whitespace, YAML lint, gofmt, golangci-lint, markdownlint-cli2, codespell.

## Worktrees

Always name worktrees at creation time to avoid random names (e.g. `structured-orbiting-duckling`).

```bash
# Via CLI
claude --worktree feat-duckdb
claude -w fix-something
```

The name becomes both the directory name under `.claude/worktrees/` and the git branch name (`worktree-feat-duckdb`).

Worktrees created with a random name cannot be renamed directly — they must be deleted and recreated.

## Go Development Tools

This project has two Go intelligence tools that MUST be used at all times.

### MCP gopls (obligatoire)

The `gopls` MCP server is configured and active. Follow this workflow strictly:

**Read workflow** — before answering any question about Go code:
1. `go_workspace` — always call this at the start of each session
2. `go_vulncheck` — immediately after `go_workspace`
3. `go_search` — to locate symbols before reading files
4. `go_file_context` — MUST be called after reading any `.go` file for the first time
5. `go_package_api` — to understand inter-package contracts (e.g. `storage.Storer`)

**Edit workflow** — before modifying any Go code:
1. `go_symbol_references` — find all usages of a symbol before changing its definition
2. Make edits
3. `go_diagnostics` — MUST be called after every edit, passing the modified files
4. Fix all errors reported before proceeding

### LSP tool (obligatoire)

The built-in LSP tool provides go-to-definition, find references, and hover docs.
Use it for navigation tasks that gopls MCP does not cover (e.g. cross-file jump-to-definition during editing).

### Never skip these tools

Using `Read` alone on a `.go` file without following up with `go_file_context` is insufficient.
Always combine both to understand intra-package dependencies.

## PR Checklist

Before opening any pull request, always:

1. **Update `CHANGELOG.md`** — add an entry under `## [Unreleased]` in the appropriate section (`Added`, `Changed`, or `Fixed`) with a short description and the PR number (e.g. `(#52)`).
2. **Run tests** — `task test`
3. **Squash + auto-merge** — `gh pr merge <n> --squash --auto`

Never skip step 1. The CHANGELOG is the source of truth for release notes.

## Related Documentation

- `README.md` - User-facing quick start
- `docs/MIGRATION.md` - v1 to v2 upgrade guide
- `docs/SCHEDULER-IMPLEMENTATION.md` - Scheduling system details
- `queries.sql` - Example database queries with indexes
