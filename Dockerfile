# syntax=docker/dockerfile:1.21
#
# Single-platform build (linux/amd64 only) since #130, so this Dockerfile no
# longer cross-compiles: Go builds for the builder image's native arch.
# If linux/arm64 is ever reintroduced, the cross-compile pattern must come back
# with it -- `--platform=$BUILDPLATFORM` on the builder, TARGETOS/TARGETARCH
# injected by buildx (never defaulted), and the arch guards that caught #124.
#
# matrixise/rmm-tracker-builder (built from Dockerfile.builder, see
# .github/workflows/docker-builder.yml) already has the Go modules for the
# current go.mod/go.sum downloaded, so `go mod download` is out of this
# stage's critical path. It's rebuilt/pushed on every go.mod/go.sum change.

FROM matrixise/rmm-tracker-builder:go1.27 AS builder

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

# CGO_ENABLED=0 keeps the binary static so it runs on the bare alpine stage.
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build \
    set -eu; \
    CGO_ENABLED=0 go build \
    -ldflags "-X github.com/matrixise/rmm-tracker/cmd.Version=${VERSION} -X github.com/matrixise/rmm-tracker/cmd.GitBranch=${GIT_BRANCH} -X github.com/matrixise/rmm-tracker/cmd.GitCommit=${GIT_COMMIT} -X github.com/matrixise/rmm-tracker/cmd.BuildTime=${BUILD_TIME}" \
    -o rmm-tracker .

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /app/rmm-tracker .

ENTRYPOINT ["./rmm-tracker", "run"]
