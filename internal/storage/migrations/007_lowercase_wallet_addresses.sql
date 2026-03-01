-- +goose Up

-- Normalize existing wallet addresses to lowercase.
-- Ethereum addresses are case-insensitive; storing them in a consistent
-- case ensures that index lookups always hit regardless of the input case.
UPDATE token_balances
    SET wallet = LOWER(wallet)
    WHERE wallet <> LOWER(wallet);

-- +goose Down

-- Lowercase is a one-way normalization: the original case is not recoverable.
-- This down migration is intentionally a no-op.
