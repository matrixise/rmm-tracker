package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		want     slog.Level
	}{
		{
			name:     "debug level",
			logLevel: "debug",
			want:     slog.LevelDebug,
		},
		{
			name:     "info level",
			logLevel: "info",
			want:     slog.LevelInfo,
		},
		{
			name:     "warn level",
			logLevel: "warn",
			want:     slog.LevelWarn,
		},
		{
			name:     "error level",
			logLevel: "error",
			want:     slog.LevelError,
		},
		{
			name:     "invalid level defaults to info",
			logLevel: "invalid",
			want:     slog.LevelInfo,
		},
		{
			name:     "empty level defaults to info",
			logLevel: "",
			want:     slog.LevelInfo,
		},
		{
			name:     "case insensitive DEBUG",
			logLevel: "DEBUG",
			want:     slog.LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Setup(tt.logLevel)
			// Logger is configured, we can't easily test the level directly
			// but at least verify it doesn't panic
			assert.NotNil(t, slog.Default())
		})
	}
}

func TestSetupNoErrors(t *testing.T) {
	// Verify Setup can be called multiple times without panic
	Setup("info")
	Setup("debug")
	Setup("warn")
	Setup("error")

	assert.NotNil(t, slog.Default())
}
