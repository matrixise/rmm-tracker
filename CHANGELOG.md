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
- `Dockerfile`'s builder stage now builds with `-trimpath` and `-ldflags "-s -w"`, stripping the symbol table and DWARF debug info and removing absolute build-path prefixes from the binary. Measured on `linux/amd64`: 33 706 211 → 23 769 248 bytes (32.14 → 22.66 MiB, -29.5%). The four `-X` version injections (`Version`/`GitBranch`/`GitCommit`/`BuildTime`) are unaffected — `-s -w` strips debug data, not the linker-set string variables `cmd/version.go` reads
- `Dockerfile`'s builder stage sets `GOTOOLCHAIN=local` instead of `auto`. `auto` let Go silently download and switch to a different toolchain mid-build, a network dependency and a reproducibility hole in an image build; `local` pins the build to the base image's Go. A `go.mod` bump past the base image's Go version now fails the build loudly instead, and must be matched by bumping the base image
- `Dockerfile`'s builder stage now builds `FROM golang:1.27-alpine` directly instead of the custom `matrixise/rmm-tracker-builder` image. Modules are fetched by a dedicated `RUN go mod download` layer that bind-mounts only `go.mod`/`go.sum` (so it is invalidated by a dependency change, not by every source edit) into a BuildKit cache mount shared with the `go build` layer via the same `id=go-mod`. Same "no `go mod download` on the critical path" effect, with no registry round-trip and no cross-workflow coordination: the publish workflow no longer serialises an extra job that pushes an image to Docker Hub just so the next job can pull it back down, and the fragile `grep 'FROM' Dockerfile.builder` tag-derivation shell snippet that already broke once (#129) is gone. The `go mod verify` that `Dockerfile.builder` ran is not carried over: `go mod download` already checks every module against `go.sum`, so it only added build time

- `docker-publish.yml` now records why its `type=gha` cache is not paired with `reproducible-containers/buildkit-cache-dance`. `type=gha` exports image layers only, so the `Dockerfile`'s `id=go-mod`/`id=go-build` cache mounts do not survive between runs and `go build` — whose layer `COPY . .` invalidates on every commit — always compiles cold. The dance was measured rather than assumed, and rejected: it works (auto-discovery derives both mounts from their `id=`, and a build on the injected builder went from 70.6s to 1.7s locally), but the mounts are 263 MB/2878 files (`go-build`) and 393 MB/11 596 files (`go-mod`), and moving them costs 76s of inject+extract for both or 31s for `go-build` alone — against a `go build` that takes only 28.1s on a real runner. `skip-extraction` cannot avoid that cost, since it only fires on an exact cache-key hit and the whole premise is that the source changed. The comment records the numbers and the ~90s `go build` threshold at which the trade would flip

- `Dockerfile`'s runtime stage pins `alpine:3.24` instead of `alpine:latest`. An unpinned `latest` silently moves the runtime userland from one build to the next, which defeats reproducibility and makes "which base was in the image that broke?" unanswerable after the fact. 3.24 is the current stable branch (released 2026-06-09, supported to 2028-06-01) and is what `latest` resolved to at the time of the change (3.24.1), so the pin froze the contents rather than changing them

- `docker-compose.yml`'s `app` healthcheck runs BusyBox `wget` instead of `curl`: `wget -q -O /dev/null http://localhost:8080/health`. This matches `curl -f` — BusyBox wget exits non-zero on any HTTP >= 400 — and it was verified to issue a real `GET`, which matters because `/health` answers 405 to anything that is not a GET (BusyBox's `--spider` also uses GET, unlike GNU wget's HEAD, but the explicit `-O /dev/null` form avoids depending on that). Errors still print under `-q`, so `docker inspect`'s healthcheck log shows the failing status line. Verified on all four paths: 200 → exit 0, 404 → exit 1, 503 → exit 1, connection refused → exit 1

### Removed

- Optional golangci-lint pass from `Dockerfile`'s builder stage, along with its `ARG ENABLE_LINT=false` and the `RUN` layer that executed an `echo` on every build when disabled. It was redundant with the two places that already lint — the `lint` job in `.github/workflows/test.yml` and the `golangci-lint` pre-commit hook, both pinned to `v2.10.1` — and worse, it installed `golangci-lint@latest` unpinned, so an `ENABLE_LINT=true` build could enforce a different version than CI. `docs/IMPROVEMENTS.md` updated to point at `prek run -a` and the CI job instead of the removed build arg
- Cross-compilation machinery from `Dockerfile`, now that only `linux/amd64` is ever built: `--platform=$BUILDPLATFORM` on the builder stage, the buildx-injected `TARGETOS`/`TARGETARCH` ARGs and their non-empty assertions, the explicit `GOOS`/`GOARCH` on `go build`, the `go version -m` build-metadata readback, and the final stage's ELF `e_machine` guard (one fewer layer in the runtime image). Go now builds for the builder image's native architecture and `CGO_ENABLED=0` is unchanged. The issue #124 failure mode these guards protected against — a defaulted platform ARG masking the value buildx injects — cannot occur with a single target platform; a header comment in `Dockerfile` records that the whole pattern must be restored if `linux/arm64` is ever reintroduced
- `Dockerfile.builder` and the `matrixise/rmm-tracker-builder` image it produced, the `docker-builder.yml` workflow that rebuilt it on demand, the `build-builder` job in `docker-publish.yml` (and the `needs:`/`BUILDER_IMAGE` plumbing around it — `build-and-push`'s own `if:` gate already stands alone, so removing the dependency does not change when the publish runs), and the `docker:builder:build`/`docker:builder:push` tasks plus the now-unused `DOCKER_BUILDER_IMAGE` and `GO_VERSION` vars from `Taskfile.yml`. All replaced by the BuildKit cache/bind mounts described above

- `curl` from `Dockerfile`'s runtime stage. Its only in-image consumer was the `app` healthcheck in `docker-compose.yml`, which now uses the `wget` applet BusyBox already ships; the remaining `curl` invocations in `Makefile`, `Taskfile.yml`, `deploy.sh` and `test-health.sh` all run on the host, not in the container. Dropping the package takes the `apk add` layer from 6.54 MB to 1.22 MB and the whole runtime image from 39.42 MB to 34.13 MB of uncompressed layers (-13.4%), and removes libcurl, nghttp2, brotli, c-ares, libidn2 and zstd from the image's CVE surface. `ca-certificates` is deliberately kept — the app makes HTTPS JSON-RPC calls to Gnosis Chain and cannot verify TLS without a trust store

### Fixed

- `docker-compose.yml`'s `app` healthcheck could never pass, and had been failing since it was added. The image ENTRYPOINT is `./rmm-tracker run`, and `run`'s `--http` flag defaults to `""`, so the HTTP server never started and nothing ever listened on :8080 — the check failed with connection refused on every interval (reproduced against the pre-change image: `curl -f` exited 7). The service now passes `--http :8080 --interval 5m` as `command:`. The interval has to move from `RMM_TRACKER_INTERVAL` to the flag because `cmd/run.go` deliberately ignores the config/env interval once `--http` is set, so flags are the only way to get the scheduler and the health server simultaneously

- `docker-publish.yml`'s `build-builder` job tagged the pushed builder image from `go.mod`'s `go` directive (`1.26`) instead of `Dockerfile.builder`'s actual base image (`golang:1.27-alpine`), so it published `rmm-tracker-builder:go1.26` while the main `Dockerfile` pulled `rmm-tracker-builder:go1.27` — a tag that never existed. Both workflows now read the version straight from `Dockerfile.builder`'s `FROM` line, which can't drift from what's actually in the image
- Published `linux/arm64` image no longer contains an amd64 binary: `ARG TARGETARCH=amd64` masked the value injected by buildx, pinning every platform to `GOARCH=amd64`. The final stage now verifies the binary's ELF architecture against the image architecture and fails the build on mismatch (#125)
- Weekly period yield box on the wallet detail page showed the same start/end date instead of the true week range
- Weekly report table now shows one row per consecutive week pair instead of a single row aggregated over the entire requested window
- Weekly report table rows spanned 14 days instead of 7: `week_start`/`week_end` incorrectly stretched from the previous bucket to the current bucket + 7 days instead of describing the current bucket's own week
- Wallet detail page made responsive on mobile: address wraps with `break-all`, tables scroll horizontally, padding adapts to screen size (#52)

### Security

- `Dockerfile`'s runtime stage no longer runs as root. It creates a system user and switches to `USER 65532:65532` — the same numeric id distroless uses for `nonroot`, chosen so host-side file ownership stays valid if the stage is ever moved there. Nothing in the runtime code path writes to the filesystem and the health server binds :8080, so root bought the process nothing but blast radius; the binary is deliberately left root-owned and non-writable so the process cannot modify itself. Note the operational consequence for `docker-compose.yml`'s `./config.toml:/app/config.toml:ro` bind mount: the host file must be world-readable (0644) or owned by uid 65532, otherwise the app exits at startup with `failed to read config: permission denied` — a 0600 config.toml, a natural mode for a file that may carry RPC credentials, will not start

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
