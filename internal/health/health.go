package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/matrixise/realt-rmm/internal/blockchain"
	"github.com/matrixise/realt-rmm/internal/storage"
)

// Checker performs health checks on application dependencies
type Checker struct {
	store          *storage.Store
	client         *blockchain.Client
	lastRunTime    time.Time
	lastRunSuccess bool
	interval       time.Duration
	mu             sync.RWMutex
}

// NewChecker creates a new health checker
func NewChecker(store *storage.Store, client *blockchain.Client, interval time.Duration) *Checker {
	return &Checker{
		store:    store,
		client:   client,
		interval: interval,
	}
}

// UpdateLastRun updates the timestamp and status of the last execution
func (c *Checker) UpdateLastRun(success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastRunTime = time.Now()
	c.lastRunSuccess = success
}

// CheckStatus represents the health status of a component
type CheckStatus string

const (
	StatusOK       CheckStatus = "ok"
	StatusDegraded CheckStatus = "degraded"
	StatusError    CheckStatus = "error"
)

// HealthResponse is the JSON response structure
type HealthResponse struct {
	Status    CheckStatus            `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckDetail `json:"checks"`
	Uptime    string                 `json:"uptime,omitempty"`
}

// CheckDetail contains details about a specific health check
type CheckDetail struct {
	Status  CheckStatus `json:"status"`
	Message string      `json:"message,omitempty"`
}

var startTime = time.Now()

// Check performs all health checks and returns the aggregated status
func (c *Checker) Check(ctx context.Context) HealthResponse {
	checks := make(map[string]CheckDetail)
	overallStatus := StatusOK

	// Check 1: Database connectivity
	dbCheck := c.checkDatabase(ctx)
	checks["database"] = dbCheck
	if dbCheck.Status != StatusOK {
		overallStatus = StatusError
	}

	// Check 2: RPC endpoint availability
	rpcCheck := c.checkRPC(ctx)
	checks["rpc_endpoints"] = rpcCheck
	if rpcCheck.Status == StatusError {
		overallStatus = StatusError
	} else if rpcCheck.Status == StatusDegraded && overallStatus == StatusOK {
		overallStatus = StatusDegraded
	}

	// Check 3: Daemon execution (if in daemon mode)
	if c.interval > 0 {
		daemonCheck := c.checkDaemon()
		checks["daemon"] = daemonCheck
		if daemonCheck.Status != StatusOK && overallStatus == StatusOK {
			overallStatus = StatusDegraded
		}
	}

	return HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    checks,
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	}
}

// checkDatabase verifies PostgreSQL connectivity
func (c *Checker) checkDatabase(ctx context.Context) CheckDetail {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.store.Ping(ctx); err != nil {
		slog.Error("Health check: database ping failed", "error", err)
		return CheckDetail{
			Status:  StatusError,
			Message: "database unreachable: " + err.Error(),
		}
	}

	return CheckDetail{
		Status:  StatusOK,
		Message: "database connection healthy",
	}
}

// checkRPC verifies that at least one RPC endpoint is available
func (c *Checker) checkRPC(ctx context.Context) CheckDetail {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	client, url, err := c.client.GetHealthyEndpoint()
	if err != nil {
		slog.Error("Health check: no healthy RPC endpoints", "error", err)
		return CheckDetail{
			Status:  StatusError,
			Message: "no healthy RPC endpoints available",
		}
	}

	// Quick health check: get chain ID
	if _, err := client.ChainID(ctx); err != nil {
		slog.Error("Health check: RPC endpoint failed", "url", url, "error", err)
		return CheckDetail{
			Status:  StatusError,
			Message: "RPC endpoint not responding: " + err.Error(),
		}
	}

	healthStatus := c.client.GetEndpointsHealth()
	healthyCount := 0
	totalCount := len(healthStatus)

	for _, healthy := range healthStatus {
		if healthy {
			healthyCount++
		}
	}

	if healthyCount == totalCount {
		return CheckDetail{
			Status:  StatusOK,
			Message: "all RPC endpoints healthy",
		}
	}

	return CheckDetail{
		Status:  StatusDegraded,
		Message: fmt.Sprintf("%d/%d RPC endpoints healthy", healthyCount, totalCount),
	}
}

// checkDaemon verifies the daemon is executing at expected intervals
func (c *Checker) checkDaemon() CheckDetail {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If we've never run, that's OK (might be starting up)
	if c.lastRunTime.IsZero() {
		return CheckDetail{
			Status:  StatusOK,
			Message: "daemon not yet executed (startup)",
		}
	}

	// Check if last run was successful
	if !c.lastRunSuccess {
		return CheckDetail{
			Status:  StatusDegraded,
			Message: "last execution failed",
		}
	}

	// Check if we're running on schedule (allow 2x interval grace period)
	timeSinceLastRun := time.Since(c.lastRunTime)
	graceThreshold := c.interval * 2

	if timeSinceLastRun > graceThreshold {
		return CheckDetail{
			Status:  StatusDegraded,
			Message: fmt.Sprintf("no execution in %s (expected every %s)", timeSinceLastRun.Round(time.Second), c.interval),
		}
	}

	return CheckDetail{
		Status:  StatusOK,
		Message: fmt.Sprintf("last executed %s ago", timeSinceLastRun.Round(time.Second)),
	}
}

// Handler returns an http.HandlerFunc for the health endpoint
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only support GET
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		status := c.Check(ctx)

		// Set status code based on health
		statusCode := http.StatusOK
		if status.Status == StatusError {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(status); err != nil {
			slog.Error("Failed to encode health response", "error", err)
		}
	}
}
