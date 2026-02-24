-- +goose NO TRANSACTION

-- +goose Up

-- Stored generated column: UTC day bucket, computable and indexable
ALTER TABLE token_balances
    ADD COLUMN IF NOT EXISTS day_bucket TIMESTAMPTZ
    GENERATED ALWAYS AS (DATE_TRUNC('day', queried_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC') STORED;

-- Covering index for GetDailyBalances and GetDailyReport CTEs
CREATE INDEX IF NOT EXISTS idx_token_balances_wallet_dbucket_symbol
    ON token_balances(wallet, day_bucket DESC, symbol, queried_at DESC);

-- +goose Down

ALTER TABLE token_balances DROP COLUMN IF EXISTS day_bucket;
DROP INDEX IF EXISTS idx_token_balances_wallet_dbucket_symbol;
