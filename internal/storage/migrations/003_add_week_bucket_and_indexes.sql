-- +goose NO TRANSACTION

-- +goose Up

-- Stored generated column: UTC week bucket, computable and indexable
ALTER TABLE token_balances
    ADD COLUMN IF NOT EXISTS week_bucket TIMESTAMPTZ
    GENERATED ALWAYS AS (DATE_TRUNC('week', queried_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC') STORED;

-- Covering index for GetWeeklyBalances and GetWeeklyReport CTEs
-- Allows DISTINCT ON (week_bucket, symbol) without a full sort
CREATE INDEX IF NOT EXISTS idx_token_balances_wallet_wbucket_symbol
    ON token_balances(wallet, week_bucket DESC, symbol, queried_at DESC);

-- Covering index for GetBalances with wallet+symbol filter
CREATE INDEX IF NOT EXISTS idx_token_balances_wallet_symbol_time
    ON token_balances(wallet, symbol, queried_at DESC);

-- Planner tuning: default values are calibrated for spinning HDDs.
-- Set to SSD-appropriate values so the planner prefers index scans.
ALTER DATABASE rmm_tracker SET random_page_cost = 1.1;
ALTER DATABASE rmm_tracker SET work_mem = '16MB';

-- +goose Down

ALTER TABLE token_balances DROP COLUMN IF EXISTS week_bucket;
DROP INDEX IF EXISTS idx_token_balances_wallet_wbucket_symbol;
DROP INDEX IF EXISTS idx_token_balances_wallet_symbol_time;
ALTER DATABASE rmm_tracker RESET random_page_cost;
ALTER DATABASE rmm_tracker RESET work_mem;
