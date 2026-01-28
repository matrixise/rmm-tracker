package blockchain

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	unhealthyDuration  = 5 * time.Minute // Cooldown before retry
	healthCheckTimeout = 5 * time.Second
)

type endpointStatus struct {
	url           string
	client        *ethclient.Client
	healthy       bool
	lastError     error
	lastErrorTime time.Time
	mu            sync.RWMutex
}

// FailoverClient manages multiple RPC endpoints with automatic failover
type FailoverClient struct {
	endpoints    []*endpointStatus
	currentIndex int
	mu           sync.RWMutex
}

// NewFailoverClient creates a new failover client with multiple endpoints
func NewFailoverClient(urls []string) (*FailoverClient, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("at least one RPC URL is required")
	}

	fc := &FailoverClient{
		endpoints:    make([]*endpointStatus, 0, len(urls)),
		currentIndex: 0,
	}

	// Initialize all endpoints
	healthyCount := 0
	for _, url := range urls {
		client, err := ethclient.Dial(url)

		// Verify connection with test call
		var chainIDErr error
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
			_, chainIDErr = client.ChainID(ctx)
			cancel()

			if chainIDErr != nil {
				client.Close()
				client = nil
				err = chainIDErr
			}
		}

		ep := &endpointStatus{
			url:           url,
			client:        client,
			healthy:       err == nil,
			lastError:     err,
			lastErrorTime: time.Now(),
		}

		fc.endpoints = append(fc.endpoints, ep)

		if err == nil {
			healthyCount++
			slog.Info("Connected to RPC endpoint", "url", url)
		} else {
			slog.Warn("Failed to connect to RPC endpoint, will retry later", "url", url, "error", err)
		}
	}

	// At least one endpoint must be healthy
	if healthyCount == 0 {
		return nil, fmt.Errorf("no healthy RPC endpoints available")
	}

	return fc, nil
}

// GetClient returns a healthy client, automatically failing over if needed
func (fc *FailoverClient) GetClient() (*ethclient.Client, string, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	startIndex := fc.currentIndex

	// Try all endpoints in round-robin
	for i := 0; i < len(fc.endpoints); i++ {
		idx := (startIndex + i) % len(fc.endpoints)
		ep := fc.endpoints[idx]

		ep.mu.RLock()
		healthy := ep.healthy
		client := ep.client
		url := ep.url
		canRetry := time.Since(ep.lastErrorTime) > unhealthyDuration
		ep.mu.RUnlock()

		// Use healthy endpoint
		if healthy && client != nil {
			fc.currentIndex = idx
			return client, url, nil
		}

		// Try to reconnect unhealthy endpoint if cooldown expired
		if !healthy && canRetry {
			if newClient, err := ethclient.Dial(ep.url); err == nil {
				// Verify with a test call
				ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
				_, chainErr := newClient.ChainID(ctx)
				cancel()

				if chainErr == nil {
					ep.mu.Lock()
					if ep.client != nil {
						ep.client.Close()
					}
					ep.client = newClient
					ep.healthy = true
					ep.lastError = nil
					ep.mu.Unlock()

					fc.currentIndex = idx
					slog.Info("Reconnected to RPC endpoint", "url", ep.url)
					return newClient, url, nil
				} else {
					newClient.Close()
				}
			}
		}
	}

	return nil, "", fmt.Errorf("no healthy RPC endpoints available")
}

// MarkUnhealthy marks an endpoint as unhealthy and closes its connection
func (fc *FailoverClient) MarkUnhealthy(url string, err error) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	for _, ep := range fc.endpoints {
		if ep.url == url {
			ep.mu.Lock()
			ep.healthy = false
			ep.lastError = err
			ep.lastErrorTime = time.Now()
			if ep.client != nil {
				ep.client.Close()
				ep.client = nil
			}
			ep.mu.Unlock()

			slog.Warn("Marked RPC endpoint as unhealthy, will retry after cooldown",
				"url", url,
				"error", err,
				"retry_after", unhealthyDuration)
			return
		}
	}
}

// Close closes all endpoint connections
func (fc *FailoverClient) Close() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	for _, ep := range fc.endpoints {
		ep.mu.Lock()
		if ep.client != nil {
			ep.client.Close()
			ep.client = nil
		}
		ep.mu.Unlock()
	}
}
