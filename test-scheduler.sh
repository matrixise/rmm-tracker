#!/bin/bash
set -e

echo "=== Scheduler Integration Tests ==="
echo

# Test 1: Valid duration
echo "Test 1: Valid 5m duration"
DATABASE_URL="postgres://user:pass@localhost:5432/test" ./realt-rmm validate-config --config config.toml
echo "✅ PASS"
echo

# Test 2: Invalid duration (7m)
echo "Test 2: Invalid 7m duration (should fail)"
echo 'interval = "7m"' > /tmp/test-invalid.toml
cat config.toml | grep -v '^interval' | grep -v '^#' | grep -v 'run_immediately' | grep -v 'timezone' >> /tmp/test-invalid.toml
if DATABASE_URL="postgres://user:pass@localhost:5432/test" ./realt-rmm validate-config --config /tmp/test-invalid.toml 2>&1 | grep -q "failed on the 'schedule' tag"; then
    echo "✅ PASS - Correctly rejected invalid interval"
else
    echo "❌ FAIL - Should have rejected 7m interval"
    exit 1
fi
echo

# Test 3: Cron expression
echo "Test 3: Cron expression validation"
echo 'interval = "*/5 * * * *"' > /tmp/test-cron.toml
cat config.toml | grep -v '^interval' | grep -v '^#' | grep -v 'run_immediately' | grep -v 'timezone' >> /tmp/test-cron.toml
DATABASE_URL="postgres://user:pass@localhost:5432/test" ./realt-rmm validate-config --config /tmp/test-cron.toml
echo "✅ PASS"
echo

# Test 4: Complex cron expression
echo "Test 4: Complex cron expression (9am and 5pm on weekdays)"
echo 'interval = "0 9,17 * * 1-5"' > /tmp/test-complex-cron.toml
cat config.toml | grep -v '^interval' | grep -v '^#' | grep -v 'run_immediately' | grep -v 'timezone' >> /tmp/test-complex-cron.toml
DATABASE_URL="postgres://user:pass@localhost:5432/test" ./realt-rmm validate-config --config /tmp/test-complex-cron.toml
echo "✅ PASS"
echo

# Test 5: Timezone configuration
echo "Test 5: Timezone configuration"
cat > /tmp/test-timezone.toml <<EOF
rpc_urls = ["https://rpc.gnosischain.com"]
interval = "5m"
timezone = "America/New_York"
wallets = ["0x1234567890123456789012345678901234567890"]

[[tokens]]
label = "test"
address = "0x0cA4f5554Dd9Da6217d62D8df2816c82bba4157b"
fallback_decimals = 18
EOF
DATABASE_URL="postgres://user:pass@localhost:5432/test" ./realt-rmm validate-config --config /tmp/test-timezone.toml
echo "✅ PASS"
echo

# Test 6: Invalid timezone
echo "Test 6: Invalid timezone (should fail)"
cat > /tmp/test-invalid-tz.toml <<EOF
rpc_urls = ["https://rpc.gnosischain.com"]
interval = "5m"
timezone = "Invalid/Timezone"
wallets = ["0x1234567890123456789012345678901234567890"]

[[tokens]]
label = "test"
address = "0x0cA4f5554Dd9Da6217d62D8df2816c82bba4157b"
fallback_decimals = 18
EOF
if DATABASE_URL="postgres://user:pass@localhost:5432/test" ./realt-rmm validate-config --config /tmp/test-invalid-tz.toml 2>&1 | grep -q "failed on the 'timezone' tag"; then
    echo "✅ PASS - Correctly rejected invalid timezone"
else
    echo "❌ FAIL - Should have rejected invalid timezone"
    exit 1
fi
echo

echo "=== All Tests Passed ==="
