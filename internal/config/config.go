package config

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"
)

// Config represents the application configuration
type Config struct {
	// New: Multiple endpoints for high availability
	RPCUrls []string `mapstructure:"rpc_urls" validate:"omitempty,min=1,dive,url"`

	// Legacy: Single endpoint (for backward compatibility)
	RPCUrl string `mapstructure:"rpc_url" validate:"omitempty,url"`

	Wallets  []string      `mapstructure:"wallets" validate:"required,min=1,dive,eth_addr"`
	Tokens   []TokenConfig `mapstructure:"tokens" validate:"required,min=1,dive"`
	Interval string        `mapstructure:"interval" validate:"omitempty,duration"`
	LogLevel string        `mapstructure:"log_level" validate:"omitempty,oneof=debug info warn error"`
	HTTPPort int           `mapstructure:"http_port" validate:"omitempty,min=1024,max=65535"`
}

// Normalize converts single rpc_url to rpc_urls array for backward compatibility
func (cfg *Config) Normalize() error {
	// Case 1: Only rpc_url set -> convert to rpc_urls
	if cfg.RPCUrl != "" && len(cfg.RPCUrls) == 0 {
		cfg.RPCUrls = []string{cfg.RPCUrl}
		cfg.RPCUrl = ""
	}

	// Case 2: Both set -> rpc_urls takes precedence, ignore rpc_url
	if len(cfg.RPCUrls) > 0 {
		cfg.RPCUrl = ""
	}

	// Case 3: Neither set -> error
	if len(cfg.RPCUrls) == 0 {
		return fmt.Errorf("at least one RPC URL is required (rpc_url or rpc_urls)")
	}

	return nil
}

// TokenConfig represents a single token configuration
type TokenConfig struct {
	Label            string `mapstructure:"label" validate:"required,min=1,max=100"`
	Address          string `mapstructure:"address" validate:"required,eth_addr"`
	FallbackDecimals uint8  `mapstructure:"fallback_decimals" validate:"required,min=0,max=255"`
}

// ethAddressValidator validates Ethereum addresses
func ethAddressValidator(fl validator.FieldLevel) bool {
	return common.IsHexAddress(fl.Field().String())
}

// durationValidator validates duration strings
func durationValidator(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true // empty is valid (run once mode)
	}
	_, err := time.ParseDuration(fl.Field().String())
	return err == nil
}

// NewValidator creates a validator with custom validation rules
func NewValidator() *validator.Validate {
	validate := validator.New()
	validate.RegisterValidation("eth_addr", ethAddressValidator)
	validate.RegisterValidation("duration", durationValidator)
	return validate
}
