package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// JobFunc is the function signature for scheduled jobs
type JobFunc func(ctx context.Context) error

// Scheduler wraps gocron v2 and provides clock-aligned scheduling
type Scheduler struct {
	gocronScheduler gocron.Scheduler
	job             gocron.Job
	interval        string
	timezone        *time.Location
	runImmediately  bool
	logger          *slog.Logger
}

// Config holds scheduler configuration
type Config struct {
	Interval       string         // Duration (e.g., "5m") or cron expression (e.g., "*/5 * * * *")
	Timezone       *time.Location // Timezone for cron expressions (default: UTC)
	RunImmediately bool           // Execute immediately on start (default: true)
	Logger         *slog.Logger   // Logger for scheduler events
}

var (
	// cronPattern matches cron expressions (5 or 6 fields)
	cronPattern = regexp.MustCompile(`^(\S+\s+){4,5}\S+$`)

	// validMinuteIntervals are minute intervals that divide evenly into 60
	validMinuteIntervals = map[int]bool{
		1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 10: true, 12: true,
		15: true, 20: true, 30: true,
	}

	// validHourIntervals are hour intervals that divide evenly into 24
	validHourIntervals = map[int]bool{
		1: true, 2: true, 3: true, 4: true, 6: true, 8: true, 12: true, 24: true,
	}

	// validSecondIntervals are second intervals that divide evenly into 60
	validSecondIntervals = map[int]bool{
		1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 10: true, 12: true,
		15: true, 20: true, 30: true,
	}
)

// NewScheduler creates a new scheduler instance
func NewScheduler(ctx context.Context, cfg Config, jobFunc JobFunc) (*Scheduler, error) {
	if cfg.Timezone == nil {
		cfg.Timezone = time.UTC
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	s := &Scheduler{
		interval:       cfg.Interval,
		timezone:       cfg.Timezone,
		runImmediately: cfg.RunImmediately,
		logger:         cfg.Logger,
	}

	// Create gocron scheduler
	gocronScheduler, err := gocron.NewScheduler(
		gocron.WithLocation(cfg.Timezone),
		gocron.WithLogger(newGocronLoggerAdapter(cfg.Logger)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gocron scheduler: %w", err)
	}
	s.gocronScheduler = gocronScheduler

	// Determine if interval is duration or cron expression
	isCron := isCronExpression(cfg.Interval)

	var job gocron.Job
	if isCron {
		// Use cron expression directly
		s.logger.Info("Using cron expression", "cron", cfg.Interval, "timezone", cfg.Timezone.String())
		job, err = gocronScheduler.NewJob(
			gocron.CronJob(cfg.Interval, true), // withSeconds = true for 6-field cron
			gocron.NewTask(func() {
				if err := jobFunc(ctx); err != nil {
					s.logger.Error("Job execution failed", "error", err)
				}
			}),
		)
	} else {
		// Convert duration to clock-aligned cron expression
		cronExpr, err := durationToCron(cfg.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %w", err)
		}

		s.logger.Info("Converting duration to cron", "duration", cfg.Interval, "cron", cronExpr, "timezone", cfg.Timezone.String())

		job, err = gocronScheduler.NewJob(
			gocron.CronJob(cronExpr, strings.Count(cronExpr, " ") == 5), // withSeconds if 6 fields
			gocron.NewTask(func() {
				if err := jobFunc(ctx); err != nil {
					s.logger.Error("Job execution failed", "error", err)
				}
			}),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create scheduled job: %w", err)
	}

	s.job = job

	return s, nil
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	// Run immediately if configured
	if s.runImmediately {
		s.logger.Info("Executing job immediately before starting scheduler")
		// Execute the job's task once (gocron handles this internally when job is created)
		if err := s.job.RunNow(); err != nil {
			s.logger.Error("Immediate execution failed", "error", err)
			// Don't return error, continue with scheduled execution
		}
	}

	// Start the scheduler
	s.gocronScheduler.Start()

	nextRun, err := s.NextRun()
	if err == nil {
		s.logger.Info("Scheduler started", "next_run", nextRun.Format(time.RFC3339), "timezone", s.timezone.String())
	} else {
		s.logger.Info("Scheduler started")
	}

	return nil
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() error {
	s.logger.Info("Stopping scheduler")
	return s.gocronScheduler.Shutdown()
}

// NextRun returns the next scheduled run time
func (s *Scheduler) NextRun() (time.Time, error) {
	nextRun, err := s.job.NextRun()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get next run: %w", err)
	}
	return nextRun, nil
}

// LastRun returns the last run time
func (s *Scheduler) LastRun() (time.Time, error) {
	lastRun, err := s.job.LastRun()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last run: %w", err)
	}
	return lastRun, nil
}

// GetExpectedInterval calculates the expected interval between executions
// This is used by the health checker to determine if executions are on schedule
func (s *Scheduler) GetExpectedInterval() (time.Duration, error) {
	// Try to parse as duration first
	if duration, err := time.ParseDuration(s.interval); err == nil {
		return duration, nil
	}

	// For cron expressions, we cannot easily determine the interval
	// since it may be irregular (e.g., "0 9,17 * * *" runs at 9am and 5pm)
	// The health checker should use NextRun() for precise monitoring instead

	// Return a conservative default for health check grace periods
	return 5 * time.Minute, nil
}

// isCronExpression checks if a string is a cron expression (vs duration)
func isCronExpression(s string) bool {
	// Cron expressions have 5 or 6 space-separated fields
	return cronPattern.MatchString(s)
}

// durationToCron converts a duration string to a clock-aligned cron expression
// Examples:
//   "5m" -> "*/5 * * * *"
//   "1h" -> "0 */1 * * *"
//   "30s" -> "*/30 * * * * *"
func durationToCron(durationStr string) (string, error) {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return "", fmt.Errorf("invalid duration format: %w", err)
	}

	// Convert duration to appropriate cron expression based on magnitude
	switch {
	case duration < time.Minute:
		// Seconds-based cron (6 fields)
		seconds := int(duration.Seconds())
		if seconds == 0 || 60%seconds != 0 {
			return "", fmt.Errorf("second intervals must divide evenly into 60 (got %ds)", seconds)
		}
		if !validSecondIntervals[seconds] {
			return "", fmt.Errorf("second interval %ds is not a standard divisor of 60", seconds)
		}
		return fmt.Sprintf("*/%d * * * * *", seconds), nil

	case duration < time.Hour:
		// Minutes-based cron (5 fields)
		minutes := int(duration.Minutes())
		if minutes == 0 || 60%minutes != 0 {
			return "", fmt.Errorf("minute intervals must divide evenly into 60 (got %dm)", minutes)
		}
		if !validMinuteIntervals[minutes] {
			return "", fmt.Errorf("minute interval %dm is not a standard divisor of 60", minutes)
		}
		return fmt.Sprintf("*/%d * * * *", minutes), nil

	case duration%time.Hour == 0:
		// Hour-based cron (5 fields)
		hours := int(duration.Hours())
		if hours == 0 || 24%hours != 0 {
			return "", fmt.Errorf("hour intervals must divide evenly into 24 (got %dh)", hours)
		}
		if !validHourIntervals[hours] {
			return "", fmt.Errorf("hour interval %dh is not a standard divisor of 24", hours)
		}
		return fmt.Sprintf("0 */%d * * *", hours), nil

	default:
		return "", fmt.Errorf("duration must be whole seconds, minutes, or hours (got %s)", durationStr)
	}
}

// ValidateScheduleInterval validates a schedule interval (duration or cron)
func ValidateScheduleInterval(interval string) error {
	if interval == "" {
		return nil // Empty is valid (one-shot mode)
	}

	// Check if it's a cron expression
	if isCronExpression(interval) {
		// Basic validation - gocron will do deeper validation
		fields := strings.Fields(interval)
		if len(fields) != 5 && len(fields) != 6 {
			return errors.New("cron expression must have 5 or 6 fields")
		}
		return nil
	}

	// Validate as duration
	_, err := durationToCron(interval)
	return err
}

// gocronLoggerAdapter adapts slog.Logger to gocron.Logger interface
type gocronLoggerAdapter struct {
	logger *slog.Logger
}

func newGocronLoggerAdapter(logger *slog.Logger) gocron.Logger {
	return &gocronLoggerAdapter{logger: logger}
}

func (a *gocronLoggerAdapter) Debug(msg string, args ...any) {
	a.logger.Debug(msg, args...)
}

func (a *gocronLoggerAdapter) Info(msg string, args ...any) {
	a.logger.Info(msg, args...)
}

func (a *gocronLoggerAdapter) Warn(msg string, args ...any) {
	a.logger.Warn(msg, args...)
}

func (a *gocronLoggerAdapter) Error(msg string, args ...any) {
	a.logger.Error(msg, args...)
}

// ParseCronFields extracts interval information from cron expression
// This is a helper for documentation/debugging purposes
func ParseCronFields(cronExpr string) map[string]string {
	fields := strings.Fields(cronExpr)
	if len(fields) == 5 {
		return map[string]string{
			"minute":     fields[0],
			"hour":       fields[1],
			"dayOfMonth": fields[2],
			"month":      fields[3],
			"dayOfWeek":  fields[4],
		}
	} else if len(fields) == 6 {
		return map[string]string{
			"second":     fields[0],
			"minute":     fields[1],
			"hour":       fields[2],
			"dayOfMonth": fields[3],
			"month":      fields[4],
			"dayOfWeek":  fields[5],
		}
	}
	return nil
}

// DescribeSchedule provides a human-readable description of the schedule
func DescribeSchedule(interval string, timezone *time.Location) string {
	if timezone == nil {
		timezone = time.UTC
	}

	if isCronExpression(interval) {
		return fmt.Sprintf("cron: %s (%s)", interval, timezone.String())
	}

	duration, err := time.ParseDuration(interval)
	if err != nil {
		return fmt.Sprintf("invalid: %s", interval)
	}

	cronExpr, err := durationToCron(interval)
	if err != nil {
		return fmt.Sprintf("duration: %s (non-aligned)", interval)
	}

	return fmt.Sprintf("every %s (aligned to clock, cron: %s, %s)", duration, cronExpr, timezone.String())
}
