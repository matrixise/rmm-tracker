-- +goose Up
-- Backfill last_run_at from the actual blockchain query timestamps when it is
-- still NULL (e.g. databases created before migration 005 was introduced).
UPDATE tracker_metadata
SET last_run_at = (SELECT MAX(queried_at) FROM token_balances),
    succeeded   = true
WHERE id = 1
  AND last_run_at IS NULL
  AND EXISTS (SELECT 1 FROM token_balances LIMIT 1);

-- +goose Down
UPDATE tracker_metadata SET last_run_at = NULL, succeeded = false WHERE id = 1;
