# syntax=docker/dockerfile:1.21

FROM golang:1.25-alpine AS builder

ARG ENABLE_LINT=false

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Optional linting (enabled with --build-arg ENABLE_LINT=true)
RUN if [ "$ENABLE_LINT" = "true" ]; then \
        apk add --no-cache git && \
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
        /go/bin/golangci-lint run --timeout=5m; \
    else \
        echo "Linting disabled (use --build-arg ENABLE_LINT=true to enable)"; \
    fi

# Build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o realt-rmm .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/realt-rmm .
COPY config.toml .

ENTRYPOINT ["./realt-rmm", "run"]
