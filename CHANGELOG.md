# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `.dockerignore` shrinking the Docker build context (414 kB → 312 kB) and keeping the builder stage's `COPY . .` layer from being invalidated by edits to docs, CI workflows, Taskfile or shell scripts. It also stops a developer's gitignored `config.toml`/`.env` from ever reaching the build context. Deliberately an explicit denylist: `main.go` embeds `CHANGELOG.md` and `internal/storage/migrate.go` embeds `migrations/*.sql`, so a blanket `*.md` rule needs the `!CHANGELOG.md` negation and `queries.sql` must be excluded by exact path rather than a `*.sql` glob — both constraints are documented in the file's header

### Changed

- Published `matrixise/rmm-tracker` images are now `linux/amd64` only, matching the only architecture actually deployed to production. Dropped the QEMU setup step and the `linux/arm64` platform target from `docker-publish.yml` accordingly
- Bumped the builder base image to `golang:1.27-alpine` (from `1.26`) and wired the main `Dockerfile`'s builder stage to build `FROM matrixise/rmm-tracker-builder`, so `go mod download` no longer runs on every image build
- `docker-publish.yml` now rebuilds and pushes the builder image itself (as a `build-builder` job that `build-and-push` explicitly `needs:`) on every deploy, instead of relying on a separately-triggered workflow — removes the race between the two publishing to Docker Hub. `docker-builder.yml` is now manual-only (`workflow_dispatch`), for ad-hoc rebuilds

### Removed

- Cross-compilation machinery from `Dockerfile`, now that only `linux/amd64` is ever built: `--platform=$BUILDPLATFORM` on the builder stage, the buildx-injected `TARGETOS`/`TARGETARCH` ARGs and their non-empty assertions, the explicit `GOOS`/`GOARCH` on `go build`, the `go version -m` build-metadata readback, and the final stage's ELF `e_machine` guard (one fewer layer in the runtime image). Go now builds for the builder image's native architecture and `CGO_ENABLED=0` is unchanged. The issue #124 failure mode these guards protected against — a defaulted platform ARG masking the value buildx injects — cannot occur with a single target platform; a header comment in `Dockerfile` records that the whole pattern must be restored if `linux/arm64` is ever reintroduced

### Fixed

- `docker-publish.yml`'s `build-builder` job tagged the pushed builder image from `go.mod`'s `go` directive (`1.26`) instead of `Dockerfile.builder`'s actual base image (`golang:1.27-alpine`), so it published `rmm-tracker-builder:go1.26` while the main `Dockerfile` pulled `rmm-tracker-builder:go1.27` — a tag that never existed. Both workflows now read the version straight from `Dockerfile.builder`'s `FROM` line, which can't drift from what's actually in the image
- Published `linux/arm64` image no longer contains an amd64 binary: `ARG TARGETARCH=amd64` masked the value injected by buildx, pinning every platform to `GOARCH=amd64`. The final stage now verifies the binary's ELF architecture against the image architecture and fails the build on mismatch (#125)
- Weekly period yield box on the wallet detail page showed the same start/end date instead of the true week range
- Weekly report table now shows one row per consecutive week pair instead of a single row aggregated over the entire requested window
- Weekly report table rows spanned 14 days instead of 7: `week_start`/`week_end` incorrectly stretched from the previous bucket to the current bucket + 7 days instead of describing the current bucket's own week
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
