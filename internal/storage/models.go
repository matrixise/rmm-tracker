package storage

import (
	"math/big"
	"time"

	"github.com/shopspring/decimal"
)

// TokenBalance represents a token balance record
type TokenBalance struct {
	ID           int64
	QueriedAt    time.Time
	Wallet       string
	TokenAddress string
	Symbol       string
	Decimals     uint8
	RawBalance   *big.Int
	Balance      decimal.Decimal
}
