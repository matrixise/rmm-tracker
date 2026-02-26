# syntax=docker/dockerfile:1.21
#
# --platform=$BUILDPLATFORM ensures the builder stage always runs natively on
# the CI host (linux/amd64), even when cross-compiling for arm64.  QEMU is
# only needed for the final `apk add` step, not for the Go compilation.

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG ENABLE_LINT=false

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

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
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# CGO_ENABLED=0 + GOOS/GOARCH → pure-Go cross-compilation running natively on
# the build host.  No QEMU emulation needed for the compilation itself.
# Cache key is per-TARGETARCH so amd64 and arm64 caches don't collide.
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build-${TARGETARCH} \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags "-X github.com/matrixise/rmm-tracker/cmd.Version=${VERSION} -X github.com/matrixise/rmm-tracker/cmd.GitBranch=${GIT_BRANCH} -X github.com/matrixise/rmm-tracker/cmd.GitCommit=${GIT_COMMIT} -X github.com/matrixise/rmm-tracker/cmd.BuildTime=${BUILD_TIME}" \
    -o rmm-tracker .

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /app/rmm-tracker .

ENTRYPOINT ["./rmm-tracker", "run"]
