# syntax=docker/dockerfile:1.21
#
# --platform=$BUILDPLATFORM ensures the builder stage always runs natively on
# the CI host (linux/amd64), even when cross-compiling for arm64.  QEMU is
# only needed for the final `apk add` step, not for the Go compilation.
#
# matrixise/rmm-tracker-builder (built from Dockerfile.builder, see
# .github/workflows/docker-builder.yml) already has the Go modules for the
# current go.mod/go.sum downloaded, so `go mod download` is out of this
# stage's critical path. It's rebuilt/pushed on every go.mod/go.sum change
# and is amd64-only, matching $BUILDPLATFORM on the CI host.

FROM --platform=$BUILDPLATFORM matrixise/rmm-tracker-builder:go1.27 AS builder

ARG ENABLE_LINT=false

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY . .

# Optional linting (enabled with --build-arg ENABLE_LINT=true)
RUN if [ "$ENABLE_LINT" = "true" ]; then \
        apk add --no-cache git && \
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
        /go/bin/golangci-lint run --timeout=5m; \
    else \
        echo "Linting disabled (use --build-arg ENABLE_LINT=true to enable)"; \
    fi

# Build args for version info
ARG VERSION=dev
ARG GIT_BRANCH=unknown
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Injected by Docker buildx: TARGETOS=linux, TARGETARCH=amd64|arm64
#
# These ARGs MUST stay without a default value.  Giving a predefined platform
# ARG a default masks the value buildx injects: the stage then sees the literal
# default for every target platform.  That is what shipped an amd64 binary
# inside the arm64 image (see issue #124) -- `ARG TARGETARCH=amd64` pinned the
# arm64 build to GOARCH=amd64 while the final stage was still tagged arm64.
ARG TARGETOS
ARG TARGETARCH

# CGO_ENABLED=0 + GOOS/GOARCH → pure-Go cross-compilation running natively on
# the build host.  No QEMU emulation needed for the compilation itself.
# Cache key is per-TARGETARCH so amd64 and arm64 caches don't collide.
#
# The trailing check is a hard guard: it reads GOOS/GOARCH back out of the
# binary's own build metadata and fails the build on any mismatch, so a
# wrong-architecture image can never be published silently again.
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build-${TARGETARCH} \
    set -eu; \
    : "${TARGETOS:?not injected by buildx -- build with docker buildx}"; \
    : "${TARGETARCH:?not injected by buildx -- build with docker buildx}"; \
    echo "==> target ${TARGETOS}/${TARGETARCH} (build host $(go env GOHOSTOS)/$(go env GOHOSTARCH))"; \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags "-X github.com/matrixise/rmm-tracker/cmd.Version=${VERSION} -X github.com/matrixise/rmm-tracker/cmd.GitBranch=${GIT_BRANCH} -X github.com/matrixise/rmm-tracker/cmd.GitCommit=${GIT_COMMIT} -X github.com/matrixise/rmm-tracker/cmd.BuildTime=${BUILD_TIME}" \
    -o rmm-tracker .; \
    built_os="$(go version -m rmm-tracker | awk '$1=="build" && $2 ~ /^GOOS=/ {print substr($2, 6)}')"; \
    built_arch="$(go version -m rmm-tracker | awk '$1=="build" && $2 ~ /^GOARCH=/ {print substr($2, 8)}')"; \
    echo "==> built ${built_os}/${built_arch}"; \
    if [ "${built_os}/${built_arch}" != "${TARGETOS}/${TARGETARCH}" ]; then \
        echo "FATAL: binary is ${built_os}/${built_arch} but target is ${TARGETOS}/${TARGETARCH}" >&2; \
        exit 1; \
    fi

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /app/rmm-tracker .

# Authoritative architecture guard.
#
# This stage is NOT pinned to $BUILDPLATFORM, so its architecture comes from
# buildx's --platform directly rather than from an ARG.  That makes it the only
# place that can detect the issue #124 failure mode, where the ARG itself was
# wrong and every ARG-based check agreed with it.
#
# e_machine is a little-endian 2-byte field at offset 18 of the ELF header:
# 0x3e -> x86-64, 0xb7 -> aarch64.
RUN set -eu; \
    image_arch="$(apk --print-arch)"; \
    machine="$(od -An -tx1 -j18 -N2 rmm-tracker | tr -d ' \n')"; \
    case "${machine}" in \
        3e00) binary_arch="x86_64" ;; \
        b700) binary_arch="aarch64" ;; \
        *)    binary_arch="unknown(0x${machine})" ;; \
    esac; \
    echo "==> image arch ${image_arch}, binary arch ${binary_arch}"; \
    if [ "${binary_arch}" != "${image_arch}" ]; then \
        echo "FATAL: ${image_arch} image contains a ${binary_arch} binary" >&2; \
        exit 1; \
    fi

ENTRYPOINT ["./rmm-tracker", "run"]
