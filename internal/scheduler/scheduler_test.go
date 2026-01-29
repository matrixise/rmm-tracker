package scheduler

import (
	"testing"
	"time"
)

func TestValidateScheduleInterval(t *testing.T) {
	tests := []struct {
		name      string
		interval  string
		wantError bool
	}{
		// Valid durations
		{"valid 5m", "5m", false},
		{"valid 10m", "10m", false},
		{"valid 15m", "15m", false},
		{"valid 30m", "30m", false},
		{"valid 1h", "1h", false},
		{"valid 2h", "2h", false},
		{"valid 6h", "6h", false},
		{"valid 12h", "12h", false},
		{"empty interval", "", false},

		// Invalid durations
		{"invalid 7m", "7m", true},
		{"invalid 13m", "13m", true},
		{"invalid 5h", "5h", true},
		{"invalid 7h", "7h", true},

		// Valid cron expressions
		{"cron every 5 min", "*/5 * * * *", false},
		{"cron every 2 hours", "0 */2 * * *", false},
		{"cron complex", "0 9,17 * * 1-5", false},

		// Invalid cron expressions
		{"cron too few fields", "*/5 * * *", true},
		{"cron too many fields", "*/5 * * * * * *", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScheduleInterval(tt.interval)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateScheduleInterval(%q) error = %v, wantError %v", tt.interval, err, tt.wantError)
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
		{"5 minutes", "5m", "*/5 * * * *", false},
		{"10 minutes", "10m", "*/10 * * * *", false},
		{"30 minutes", "30m", "*/30 * * * *", false},
		{"1 hour", "1h", "0 */1 * * *", false},
		{"2 hours", "2h", "0 */2 * * *", false},
		{"6 hours", "6h", "0 */6 * * *", false},
		{"30 seconds", "30s", "*/30 * * * * *", false},

		// Invalid
		{"7 minutes", "7m", "", true},
		{"5 hours", "5h", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := durationToCron(tt.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("durationToCron(%q) error = %v, wantErr %v", tt.duration, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("durationToCron(%q) = %q, want %q", tt.duration, got, tt.want)
			}
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
			if got := isCronExpression(tt.input); got != tt.want {
				t.Errorf("isCronExpression(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDescribeSchedule(t *testing.T) {
	utc := time.UTC
	ny, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name     string
		interval string
		timezone *time.Location
		want     string
	}{
		{"5m UTC", "5m", utc, "every 5m0s (aligned to clock, cron: */5 * * * *, UTC)"},
		{"1h NYC", "1h", ny, "every 1h0m0s (aligned to clock, cron: 0 */1 * * *, America/New_York)"},
		{"cron UTC", "*/5 * * * *", utc, "cron: */5 * * * * (UTC)"},
		{"invalid 7m", "7m", utc, "duration: 7m (non-aligned)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DescribeSchedule(tt.interval, tt.timezone)
			if got != tt.want {
				t.Errorf("DescribeSchedule(%q, %s) = %q, want %q", tt.interval, tt.timezone, got, tt.want)
			}
		})
	}
}
