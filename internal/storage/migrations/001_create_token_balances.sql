-- +goose Up
CREATE TABLE IF NOT EXISTS token_balances (
    id            BIGSERIAL PRIMARY KEY,
    queried_at    TIMESTAMPTZ NOT NULL,
    wallet        TEXT NOT NULL,
    token_address TEXT NOT NULL,
    symbol        TEXT NOT NULL,
    decimals      SMALLINT NOT NULL,
    raw_balance   TEXT NOT NULL,
    balance       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_token_balances_wallet_token_time
    ON token_balances(wallet, token_address, queried_at DESC);

CREATE INDEX IF NOT EXISTS idx_token_balances_queried_at
    ON token_balances(queried_at DESC);

CREATE INDEX IF NOT EXISTS idx_token_balances_wallet
    ON token_balances(wallet);

-- +goose Down
DROP TABLE IF EXISTS token_balances;
