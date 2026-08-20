# syntax=docker/dockerfile:1.21
#
# Single-platform build (linux/amd64 only) since #130, so this Dockerfile no
# longer cross-compiles: Go builds for the builder image's native arch.
# If linux/arm64 is ever reintroduced, the cross-compile pattern must come back
# with it -- `--platform=$BUILDPLATFORM` on the builder, TARGETOS/TARGETARCH
# injected by buildx (never defaulted), and the arch guards that caught #124.
#
# Modules are fetched by a dedicated `go mod download` layer that bind-mounts
# only go.mod/go.sum, so it is invalidated by a dependency change and not by
# every source edit, and stores them in a BuildKit cache mount shared with the
# `go build` layer below (same `id=go-mod`). Both mounts must stay in sync:
# the modules live in the cache, not in an image layer, so dropping the mount
# from the build step would silently re-download everything.

FROM golang:1.27-alpine AS builder

# 'local' pins the build to the base image's Go toolchain: no silent mid-build
# download of a different version. Bumping go.mod past that version must now be
# matched by bumping the base image -- the resulting build failure is intended.
ENV GOTOOLCHAIN=local

WORKDIR /app

RUN --mount=type=cache,target=/go/pkg/mod,id=go-mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

COPY . .

# Build args for version info
ARG VERSION=dev
ARG GIT_BRANCH=unknown
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# CGO_ENABLED=0 keeps the binary static so it runs on the bare alpine stage.
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build \
    --mount=type=cache,target=/go/pkg/mod,id=go-mod \
    set -eu; \
    CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w -X github.com/matrixise/rmm-tracker/cmd.Version=${VERSION} -X github.com/matrixise/rmm-tracker/cmd.GitBranch=${GIT_BRANCH} -X github.com/matrixise/rmm-tracker/cmd.GitCommit=${GIT_COMMIT} -X github.com/matrixise/rmm-tracker/cmd.BuildTime=${BUILD_TIME}" \
    -o rmm-tracker .

# Pinned to an explicit Alpine minor. `alpine:latest` silently moves the runtime
# base from one build to the next, which wrecks reproducibility and makes
# "which userland was in the image that broke?" unanswerable after the fact.
# 3.24 is the current stable branch (released 2026-06-09, supported until
# 2028-06-01) and is what `latest` resolves to today, so pinning it changes
# nothing about the image contents -- it only freezes them.
FROM alpine:3.24

# ca-certificates is required: the app makes HTTPS JSON-RPC calls to Gnosis
# Chain and cannot verify TLS without a trust store.
#
# `curl` used to be installed here for exactly one consumer -- the `app`
# healthcheck in docker-compose.yml. BusyBox already ships a `wget` applet that
# covers that need, so the package was pure image size and CVE surface. The
# healthcheck now uses `wget -q -O /dev/null`, which mirrors `curl -f`: BusyBox
# wget exits non-zero on any HTTP >= 400. Note it is a real GET, which matters
# because /health answers 405 to anything that is not a GET.
RUN apk --no-cache add ca-certificates

# Run unprivileged. No runtime code path writes to the filesystem and the health
# server binds :8080, so root buys the process nothing but blast radius.
# UID/GID 65532 is the same numeric id distroless uses for its `nonroot` user,
# so host-side file ownership stays valid if this stage is ever moved there.
RUN addgroup -g 65532 -S nonroot \
    && adduser -u 65532 -S -G nonroot -H -h /nonexistent -s /sbin/nologin nonroot

WORKDIR /app

# Deliberately left owned by root and non-writable by the runtime user: the
# process never needs to modify its own binary or its working directory.
COPY --from=builder /app/rmm-tracker .

USER 65532:65532

ENTRYPOINT ["./rmm-tracker", "run"]
