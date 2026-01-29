package scheduler

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateScheduleInterval(t *testing.T) {
	tests := []struct {
		name      string
		interval  string
		wantError bool
	}{
		// Empty interval
		{"empty interval", "", false},

		// Valid minute durations
		{"valid 1m", "1m", false},
		{"valid 5m", "5m", false},
		{"valid 10m", "10m", false},
		{"valid 15m", "15m", false},
		{"valid 20m", "20m", false},
		{"valid 30m", "30m", false},

		// Valid hour durations
		{"valid 1h", "1h", false},
		{"valid 2h", "2h", false},
		{"valid 3h", "3h", false},
		{"valid 4h", "4h", false},
		{"valid 6h", "6h", false},
		{"valid 8h", "8h", false},
		{"valid 12h", "12h", false},
		{"valid 24h", "24h", false},

		// Valid second durations
		{"valid 1s", "1s", false},
		{"valid 5s", "5s", false},
		{"valid 10s", "10s", false},
		{"valid 15s", "15s", false},
		{"valid 30s", "30s", false},

		// Invalid minute durations
		{"invalid 7m", "7m", true},
		{"invalid 13m", "13m", true},
		{"invalid 45m", "45m", true},

		// Invalid hour durations
		{"invalid 5h", "5h", true},
		{"invalid 7h", "7h", true},
		{"invalid 11h", "11h", true},

		// Invalid second durations
		{"invalid 7s", "7s", true},
		{"invalid 13s", "13s", true},

		// Valid cron expressions (5 fields)
		{"cron every 5 min", "*/5 * * * *", false},
		{"cron every 2 hours", "0 */2 * * *", false},
		{"cron complex", "0 9,17 * * 1-5", false},
		{"cron at midnight", "0 0 * * *", false},

		// Valid cron expressions (6 fields with seconds)
		{"cron 6 fields", "*/30 * * * * *", false},
		{"cron with seconds at hour start", "0 0 * * * *", false},

		// Invalid cron expressions
		{"cron too few fields", "*/5 * * *", true},
		{"cron too many fields", "*/5 * * * * * *", true},
		{"cron 1 field", "*/5", true},
		{"cron 2 fields", "*/5 *", true},
		{"cron 3 fields", "*/5 * *", true},

		// Invalid format
		{"non-duration non-cron", "invalid", true},
		{"mixed units", "1h30m", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScheduleInterval(tt.interval)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDurationToCron(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     string
		wantErr  bool
	}{
		// Minutes
		{"5 minutes", "5m", "*/5 * * * *", false},
		{"10 minutes", "10m", "*/10 * * * *", false},
		{"30 minutes", "30m", "*/30 * * * *", false},
		{"1 minute", "1m", "*/1 * * * *", false},
		{"15 minutes", "15m", "*/15 * * * *", false},
		{"20 minutes", "20m", "*/20 * * * *", false},

		// Hours
		{"1 hour", "1h", "0 */1 * * *", false},
		{"2 hours", "2h", "0 */2 * * *", false},
		{"3 hours", "3h", "0 */3 * * *", false},
		{"4 hours", "4h", "0 */4 * * *", false},
		{"6 hours", "6h", "0 */6 * * *", false},
		{"8 hours", "8h", "0 */8 * * *", false},
		{"12 hours", "12h", "0 */12 * * *", false},
		{"24 hours", "24h", "0 */24 * * *", false},

		// Seconds
		{"30 seconds", "30s", "*/30 * * * * *", false},
		{"15 seconds", "15s", "*/15 * * * * *", false},
		{"10 seconds", "10s", "*/10 * * * * *", false},
		{"5 seconds", "5s", "*/5 * * * * *", false},
		{"1 second", "1s", "*/1 * * * * *", false},

		// Invalid minutes
		{"7 minutes", "7m", "", true},
		{"13 minutes", "13m", "", true},

		// Invalid hours
		{"5 hours", "5h", "", true},
		{"7 hours", "7h", "", true},
		{"11 hours", "11h", "", true},

		// Invalid seconds
		{"7 seconds", "7s", "", true},
		{"13 seconds", "13s", "", true},

		// Invalid format
		{"non-duration", "invalid", "", true},
		{"mixed units", "1h30m", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := durationToCron(tt.duration)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCronExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"5-field cron", "*/5 * * * *", true},
		{"6-field cron", "*/30 * * * * *", true},
		{"duration 5m", "5m", false},
		{"duration 1h", "1h", false},
		{"invalid", "not a cron", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isCronExpression(tt.input))
		})
	}
}

func TestDescribeSchedule(t *testing.T) {
	utc := time.UTC
	ny, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	tests := []struct {
		name     string
		interval string
		timezone *time.Location
		want     string
	}{
		// Duration-based with UTC
		{"5m UTC", "5m", utc, "every 5m0s (aligned to clock, cron: */5 * * * *, UTC)"},
		{"1h UTC", "1h", utc, "every 1h0m0s (aligned to clock, cron: 0 */1 * * *, UTC)"},
		{"30s UTC", "30s", utc, "every 30s (aligned to clock, cron: */30 * * * * *, UTC)"},
		{"10m UTC", "10m", utc, "every 10m0s (aligned to clock, cron: */10 * * * *, UTC)"},

		// Duration-based with different timezone
		{"1h NYC", "1h", ny, "every 1h0m0s (aligned to clock, cron: 0 */1 * * *, America/New_York)"},
		{"5m NYC", "5m", ny, "every 5m0s (aligned to clock, cron: */5 * * * *, America/New_York)"},

		// Cron expressions with UTC
		{"cron 5 fields UTC", "*/5 * * * *", utc, "cron: */5 * * * * (UTC)"},
		{"cron complex UTC", "0 9,17 * * 1-5", utc, "cron: 0 9,17 * * 1-5 (UTC)"},
		{"cron midnight UTC", "0 0 * * *", utc, "cron: 0 0 * * * (UTC)"},

		// Cron expressions with different timezone
		{"cron NYC", "*/5 * * * *", ny, "cron: */5 * * * * (America/New_York)"},

		// Cron with 6 fields (seconds)
		{"cron 6 fields UTC", "*/30 * * * * *", utc, "cron: */30 * * * * * (UTC)"},

		// Invalid durations (non-aligned)
		{"invalid 7m", "7m", utc, "duration: 7m (non-aligned)"},
		{"invalid 13m", "13m", utc, "duration: 13m (non-aligned)"},
		{"invalid 5h", "5h", utc, "duration: 5h (non-aligned)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DescribeSchedule(tt.interval, tt.timezone))
		})
	}
}

func TestParseCronFields(t *testing.T) {
	tests := []struct {
		name     string
		cronExpr string
		wantLen  int
		checkKey string
		wantVal  string
	}{
		{
			name:     "5-field cron expression",
			cronExpr: "*/5 * * * *",
			wantLen:  5,
			checkKey: "minute",
			wantVal:  "*/5",
		},
		{
			name:     "6-field cron with seconds",
			cronExpr: "*/30 * * * * *",
			wantLen:  6,
			checkKey: "second",
			wantVal:  "*/30",
		},
		{
			name:     "complex 5-field cron",
			cronExpr: "0 9,17 * * 1-5",
			wantLen:  5,
			checkKey: "hour",
			wantVal:  "9,17",
		},
		{
			name:     "invalid cron returns nil",
			cronExpr: "invalid",
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCronFields(tt.cronExpr)
			if tt.wantLen == 0 {
				assert.Nil(t, result)
			} else {
				assert.Len(t, result, tt.wantLen)
				if tt.checkKey != "" {
					assert.Equal(t, tt.wantVal, result[tt.checkKey])
				}
			}
		})
	}
}

func TestDescribeScheduleWithNilTimezone(t *testing.T) {
	result := DescribeSchedule("5m", nil)
	assert.Contains(t, result, "UTC") // Should default to UTC
}

func TestGocronLoggerAdapter(t *testing.T) {
	logger := slog.Default()
	adapter := newGocronLoggerAdapter(logger)

	// Test that adapter methods don't panic
	t.Run("log methods work", func(t *testing.T) {
		adapter.Debug("test debug", "key", "value")
		adapter.Info("test info", "key", "value")
		adapter.Warn("test warn", "key", "value")
		adapter.Error("test error", "key", "value")
		// If we got here without panic, test passes
	})
}
