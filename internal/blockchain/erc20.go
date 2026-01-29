package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/matrixise/rmm-tracker/internal/storage"
)

const erc20ABI = `[
	{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"}
]`

// TokenInfo represents basic token configuration
type TokenInfo struct {
	Label            string
	Address          string
	FallbackDecimals uint8
}

// GetTokenBalance retrieves balance for a specific token and wallet
func (c *Client) GetTokenBalance(ctx context.Context, wallet common.Address, token TokenInfo) (storage.TokenBalance, error) {
	// Get healthy client with automatic failover
	ethClient, _, err := c.failoverClient.GetClient()
	if err != nil {
		return storage.TokenBalance{}, fmt.Errorf("no RPC endpoint available: %w", err)
	}

	// Context with timeout
	rpcCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	tokenAddr := common.HexToAddress(token.Address)
	contract := bind.NewBoundContract(tokenAddr, c.parsedABI, ethClient, ethClient, ethClient)

	result := storage.TokenBalance{
		QueriedAt:    time.Now().UTC(),
		Wallet:       wallet.Hex(),
		TokenAddress: tokenAddr.Hex(),
	}

	// Get balanceOf with retry
	var balanceResult []any
	err = c.retryWithBackoff(rpcCtx, func() error {
		return contract.Call(&bind.CallOpts{Context: rpcCtx}, &balanceResult, "balanceOf", wallet)
	})
	if err != nil {
		return result, fmt.Errorf("balanceOf: %w", err)
	}
	result.RawBalance = balanceResult[0].(*big.Int)

	// Get decimals with retry (use fallback if fails)
	result.Decimals = token.FallbackDecimals
	var decimalsResult []any
	err = c.retryWithBackoff(rpcCtx, func() error {
		return contract.Call(&bind.CallOpts{Context: rpcCtx}, &decimalsResult, "decimals")
	})
	if err == nil {
		result.Decimals = decimalsResult[0].(uint8)
	}

	// Get symbol with retry
	var symbolResult []any
	err = c.retryWithBackoff(rpcCtx, func() error {
		return contract.Call(&bind.CallOpts{Context: rpcCtx}, &symbolResult, "symbol")
	})
	if err != nil {
		return result, fmt.Errorf("symbol: %w", err)
	}
	result.Symbol = symbolResult[0].(string)

	// Convert to human-readable balance
	result.Balance = HumanBalance(result.RawBalance, result.Decimals)

	return result, nil
}
