package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

const (
	rpcTimeout    = 10 * time.Second
	maxRetries    = 3
	retryInterval = 500 * time.Millisecond
)

// Client wraps Ethereum RPC client functionality with failover support
type Client struct {
	failoverClient *FailoverClient
	parsedABI      abi.ABI
}

// NewClient creates a new blockchain client with failover support
func NewClient(rpcURLs []string) (*Client, error) {
	failoverClient, err := NewFailoverClient(rpcURLs)
	if err != nil {
		return nil, err
	}

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return &Client{
		failoverClient: failoverClient,
		parsedABI:      parsedABI,
	}, nil
}

// Close closes all RPC client connections
func (c *Client) Close() {
	c.failoverClient.Close()
}

// GetHealthyEndpoint returns a healthy RPC client and its URL
func (c *Client) GetHealthyEndpoint() (*ethclient.Client, string, error) {
	return c.failoverClient.GetClient()
}

// GetEndpointsHealth returns the health status of all RPC endpoints
func (c *Client) GetEndpointsHealth() map[string]bool {
	return c.failoverClient.GetEndpointsHealth()
}

// retryWithBackoff executes a function with exponential backoff and automatic failover
func (c *Client) retryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	var currentURL string
	var previousURL string

	for attempt := range maxRetries {
		if attempt > 0 {
			backoff := retryInterval * time.Duration(1<<uint(attempt-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Get current RPC URL
		_, currentURL, _ = c.failoverClient.GetClient()

		if err := fn(); err != nil {
			lastErr = err

			// Mark endpoint unhealthy after first failure
			if previousURL != currentURL {
				previousURL = currentURL
			}
			c.failoverClient.MarkUnhealthy(currentURL, err)

			// Try to get a different healthy endpoint
			if _, newURL, getErr := c.failoverClient.GetClient(); getErr == nil {
				if newURL != currentURL {
					// Successfully failed over to a different endpoint
					// Continue with remaining retries on new endpoint
					continue
				}
			}

			// No healthy endpoints available or still on same endpoint
			continue
		}
		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// HumanBalance converts raw balance to human-readable decimal
func HumanBalance(rawBalance *big.Int, decimals uint8) decimal.Decimal {
	if rawBalance.Sign() == 0 {
		return decimal.Zero
	}

	// Use NewFromBigInt with negative exponent to represent division by 10^decimals
	// This is equivalent to rawBalance / 10^decimals
	return decimal.NewFromBigInt(rawBalance, -int32(decimals))
}
