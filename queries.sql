-- Example optimized queries leveraging PostgreSQL indexes

-- ============================================================================
-- Index: idx_token_balances_wallet_token_time
-- Usage: Historical queries by wallet and specific token
-- ============================================================================

-- 1. Latest balance of a token for a wallet
SELECT queried_at, balance, symbol
FROM token_balances
WHERE wallet = '0x1234567890123456789012345678901234567890'
  AND token_address = '0x0cA4f5554Dd9Da6217d62D8df2816c82bba4157b'
ORDER BY queried_at DESC
LIMIT 1;

-- 2. Token evolution over the last 7 days for a wallet
SELECT queried_at, balance, symbol
FROM token_balances
WHERE wallet = '0x1234567890123456789012345678901234567890'
  AND token_address = '0xeD56F76E9cBC6A64b821e9c016eAFbd3db5436D1'
  AND queried_at >= NOW() - INTERVAL '7 days'
ORDER BY queried_at DESC;

-- 3. Compare current balance vs 24h ago for a token
WITH current AS (
  SELECT balance, queried_at
  FROM token_balances
  WHERE wallet = '0x1234567890123456789012345678901234567890'
    AND token_address = '0x0cA4f5554Dd9Da6217d62D8df2816c82bba4157b'
  ORDER BY queried_at DESC
  LIMIT 1
),
yesterday AS (
  SELECT balance, queried_at
  FROM token_balances
  WHERE wallet = '0x1234567890123456789012345678901234567890'
    AND token_address = '0x0cA4f5554Dd9Da6217d62D8df2816c82bba4157b'
    AND queried_at <= NOW() - INTERVAL '24 hours'
  ORDER BY queried_at DESC
  LIMIT 1
)
SELECT
  current.balance AS balance_now,
  yesterday.balance AS balance_24h_ago,
  (current.balance::NUMERIC - yesterday.balance::NUMERIC) AS difference
FROM current, yesterday;

-- ============================================================================
-- Index: idx_token_balances_wallet
-- Usage: Queries by wallet across all tokens
-- ============================================================================

-- 4. Latest balance of all tokens for a wallet
SELECT DISTINCT ON (token_address)
  token_address,
  symbol,
  balance,
  queried_at
FROM token_balances
WHERE wallet = '0x1234567890123456789012345678901234567890'
ORDER BY token_address, queried_at DESC;

-- 5. Total records per wallet
SELECT
  wallet,
  COUNT(*) as total_records,
  MIN(queried_at) as first_query,
  MAX(queried_at) as last_query
FROM token_balances
GROUP BY wallet
ORDER BY wallet;

-- 6. Complete snapshot of a wallet at a given time
SELECT
  symbol,
  balance,
  queried_at
FROM token_balances
WHERE wallet = '0x1234567890123456789012345678901234567890'
  AND queried_at >= '2026-01-28 14:00:00'
  AND queried_at < '2026-01-28 15:00:00'
ORDER BY symbol;

-- ============================================================================
-- Index: idx_token_balances_queried_at
-- Usage: Global time-based queries (all wallets)
-- ============================================================================

-- 7. Latest balances recorded for all wallets
SELECT
  queried_at,
  wallet,
  symbol,
  balance
FROM token_balances
WHERE queried_at >= NOW() - INTERVAL '1 hour'
ORDER BY queried_at DESC;

-- 8. Volume of records per day
SELECT
  DATE(queried_at) as date,
  COUNT(*) as records,
  COUNT(DISTINCT wallet) as unique_wallets
FROM token_balances
WHERE queried_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE(queried_at)
ORDER BY date DESC;

-- 9. Purge old records (keeps 30 days)
-- WARNING: Comment out in production, execute manually
-- DELETE FROM token_balances
-- WHERE queried_at < NOW() - INTERVAL '30 days';

-- ============================================================================
-- Advanced analytics queries
-- ============================================================================

-- 10. Detect significant variations (>10%) over 24h
WITH latest AS (
  SELECT DISTINCT ON (wallet, token_address)
    wallet,
    token_address,
    symbol,
    balance::NUMERIC as current_balance,
    queried_at
  FROM token_balances
  WHERE queried_at >= NOW() - INTERVAL '2 hours'
  ORDER BY wallet, token_address, queried_at DESC
),
previous AS (
  SELECT DISTINCT ON (wallet, token_address)
    wallet,
    token_address,
    balance::NUMERIC as previous_balance,
    queried_at
  FROM token_balances
  WHERE queried_at >= NOW() - INTERVAL '26 hours'
    AND queried_at <= NOW() - INTERVAL '23 hours'
  ORDER BY wallet, token_address, queried_at DESC
)
SELECT
  latest.wallet,
  latest.symbol,
  latest.current_balance,
  previous.previous_balance,
  ROUND(((latest.current_balance - previous.previous_balance) /
         NULLIF(previous.previous_balance, 0)) * 100, 2) as change_percent
FROM latest
LEFT JOIN previous USING (wallet, token_address)
WHERE previous.previous_balance IS NOT NULL
  AND ABS((latest.current_balance - previous.previous_balance) /
          NULLIF(previous.previous_balance, 0)) > 0.1
ORDER BY change_percent DESC;

-- 11. Aggregate total balances by token (all wallets)
SELECT DISTINCT ON (token_address)
  token_address,
  symbol,
  SUM(balance::NUMERIC) OVER (PARTITION BY token_address) as total_balance,
  COUNT(*) OVER (PARTITION BY token_address) as wallet_count,
  MAX(queried_at) OVER (PARTITION BY token_address) as last_update
FROM token_balances
WHERE queried_at >= NOW() - INTERVAL '1 hour'
ORDER BY token_address, queried_at DESC;
