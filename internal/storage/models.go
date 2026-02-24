package storage

import (
	"math/big"
	"time"

	"github.com/shopspring/decimal"
)

// TokenBalance represents a token balance record
type TokenBalance struct {
	ID           int64           `json:"id"`
	QueriedAt    time.Time       `json:"queried_at"`
	Wallet       string          `json:"wallet"`
	TokenAddress string          `json:"token_address"`
	Symbol       string          `json:"symbol"`
	Decimals     uint8           `json:"decimals"`
	RawBalance   *big.Int        `json:"-"`
	Balance      decimal.Decimal `json:"balance"`
}

// WeeklyBalance represents the last recorded balance for a (week, symbol) pair.
type WeeklyBalance struct {
	Week         time.Time       `json:"week"`
	Wallet       string          `json:"wallet"`
	TokenAddress string          `json:"token_address"`
	Symbol       string          `json:"symbol"`
	Decimals     uint8           `json:"decimals"`
	Balance      decimal.Decimal `json:"balance"`
	QueriedAt    time.Time       `json:"queried_at"`
}

// DailyBalance represents the last recorded balance for a (day, symbol) pair.
type DailyBalance struct {
	Day          time.Time       `json:"day"`
	Wallet       string          `json:"wallet"`
	TokenAddress string          `json:"token_address"`
	Symbol       string          `json:"symbol"`
	Decimals     uint8           `json:"decimals"`
	Balance      decimal.Decimal `json:"balance"`
	QueriedAt    time.Time       `json:"queried_at"`
}

// DailyReport represents the balance comparison between a day and the previous day for a token.
type DailyReport struct {
	Symbol          string          `json:"symbol"`
	TokenAddress    string          `json:"token_address"`
	Day             time.Time       `json:"day"`
	CurrentBalance  decimal.Decimal `json:"current_balance"`
	PreviousBalance decimal.Decimal `json:"previous_balance"`
	Change          decimal.Decimal `json:"change"`
	ChangePercent   decimal.Decimal `json:"change_percent"`
	APY             decimal.Decimal `json:"apy"`
}

// WeeklyReport represents the balance comparison between current and previous week for a token.
type WeeklyReport struct {
	Symbol          string          `json:"symbol"`
	TokenAddress    string          `json:"token_address"`
	WeekStart       time.Time       `json:"week_start"`
	WeekEnd         time.Time       `json:"week_end"`
	CurrentBalance  decimal.Decimal `json:"current_balance"`
	PreviousBalance decimal.Decimal `json:"previous_balance"`
	Change          decimal.Decimal `json:"change"`
	ChangePercent   decimal.Decimal `json:"change_percent"`
	DailyAvgChange  decimal.Decimal `json:"daily_avg_change"`
	APY             decimal.Decimal `json:"apy"`
}
