-- +goose Up
ALTER TABLE token_balances
  ALTER COLUMN balance TYPE NUMERIC(78, 18)
  USING balance::NUMERIC;

-- +goose Down
ALTER TABLE token_balances
  ALTER COLUMN balance TYPE TEXT
  USING balance::TEXT;
