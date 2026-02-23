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

// WeeklyBalance represents the last recorded balance for a (week, symbol) pair.
type WeeklyBalance struct {
	Week         time.Time
	Wallet       string
	TokenAddress string
	Symbol       string
	Decimals     uint8
	Balance      decimal.Decimal
	QueriedAt    time.Time
}

// WeeklyReport represents the balance comparison between current and previous week for a token.
type WeeklyReport struct {
	Symbol          string          `json:"symbol"`
	TokenAddress    string          `json:"token_address"`
	CurrentBalance  decimal.Decimal `json:"current_balance"`
	PreviousBalance decimal.Decimal `json:"previous_balance"`
	Change          decimal.Decimal `json:"change"`
	ChangePercent   decimal.Decimal `json:"change_percent"`
	DailyAvgChange  decimal.Decimal `json:"daily_avg_change"`
}
