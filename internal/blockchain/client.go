package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	rpcTimeout    = 10 * time.Second
	maxRetries    = 3
	retryInterval = 500 * time.Millisecond
)

// Client wraps Ethereum RPC client functionality
type Client struct {
	client    *ethclient.Client
	parsedABI abi.ABI
}

// NewClient creates a new blockchain client
func NewClient(rpcURL string) (*Client, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return &Client{
		client:    client,
		parsedABI: parsedABI,
	}, nil
}

// Close closes the RPC client connection
func (c *Client) Close() {
	c.client.Close()
}

// retryWithBackoff executes a function with exponential backoff
func retryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			backoff := retryInterval * time.Duration(1<<uint(attempt-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// HumanBalance converts raw balance to human-readable decimal string
func HumanBalance(rawBalance *big.Int, decimals uint8) string {
	if rawBalance.Sign() == 0 {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)

	intPart := new(big.Int).Div(rawBalance, divisor)
	remainder := new(big.Int).Mod(rawBalance, divisor)

	if remainder.Sign() == 0 {
		return intPart.String()
	}

	fracStr := fmt.Sprintf("%0*s", int(decimals), remainder.String())
	fracStr = strings.TrimRight(fracStr, "0")
	return fmt.Sprintf("%s.%s", intPart.String(), fracStr)
}
