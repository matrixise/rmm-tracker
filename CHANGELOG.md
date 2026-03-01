# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Wallet detail page made responsive on mobile: address wraps with `break-all`, tables scroll horizontally, padding adapts to screen size (#52)

## [0.1.0] - 2026-03-01

### Added

- Version number displayed in the navigation bar
- Changelog page at `/changelog` rendering this file as HTML
- Yield endpoints registered and covered by tests (#41)
- Wallet address search on the `/wallets` page (#35)
- Current balances section on the wallet detail page (#34)
- In-memory cache for dashboard summary statistics
- Unified `/api/v1/dashboard` endpoint
- Real last-run timestamp tracking in the database
- Git branch name included in build info
- Period yield display on the wallet detail page (#28)
- Web UI migrated to Alpine.js with templ templating, APY panic fixed, and hot reload (#27)
- Configurable log format (text/json) via `--log-format` flag
- `--http` flag now accepts a custom listen address
- Taskfile validation in CI and build info exposed in the health endpoint (#25)
- PostgreSQL integration tests with CI service container (#14)
- Weekly report endpoint for Prefect integration
- APY calculation and configurable weekly period in JSON API (#9)
- `Balance` field migrated from string to `decimal.Decimal` for precision
- Goose database migrations and correct scheduler `RunNow` ordering
- gocron scheduler, GitHub Actions CI/CD, and deployment tooling
- HTTP health check endpoint for daemon mode
- Automatic RPC failover client with retry logic
- Multi-endpoint RPC support with backward compatibility in config
- Multi-architecture Docker builds (AMD64 + ARM64)
- Docker buildx registry cache for faster builds
- `docker:tags` task to list published Docker Hub tags (#47)

### Changed

- Tailwind CSS CDN updated from v3 to v4.2 (#30)
- Binary size reduced by 30% using `-s -w -trimpath` build flags (#29)
- CQRS interfaces (Commander / Querier) introduced in storage layer (#31)
- `--daemon`, `--http`, and `--cron` decoupled as explicit CLI flags
- `SetLastRun` renamed to `SetLastRunStatus` for clarity (#36)
- Database name renamed from `realt_rmm` to `rmm_tracker`
- Project renamed from `realt-rmm` to `rmm-tracker`
- Docker build optimized with `BUILDPLATFORM` and native cross-compilation
- JSON API fields uniformized to snake_case (#8)
- Architecture modernized: Cobra, Viper, pgx, and validator (#26 tooling updated)

### Fixed

- Wallet addresses normalized to lowercase throughout the codebase (#48)
- Go module cache corrected in CI builds (#33)
- Concise error message returned on database connection failure
- `github_token` passed to claude-code-action to enable PR comments
- `latest` Docker tag added to images pushed from the `main` branch
- `config.toml` removed from Dockerfile to avoid leaking credentials
- Docker bridge network made explicit for reliable DNS resolution
- Pull-requests write permission added to the code-review workflow
- deploy.sh and test-env.sh scripts translated to English (#43)
- MD060 table separator spacing fixed in Web UI documentation (#42)

[Unreleased]: https://github.com/matrixise/rmm-tracker/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/matrixise/rmm-tracker/compare/5f17c98...v0.1.0
