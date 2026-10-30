package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"log/slog"
)

var ErrAbort = errors.New("retry aborted")

type config struct {
	timeout  time.Duration
	interval time.Duration
}

type Option func(*config)

func Timeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}

func Interval(d time.Duration) Option {
	return func(c *config) {
		c.interval = d
	}
}

func Context(ctx context.Context, fn func(context.Context) error, opts ...Option) error {
	cfg := &config{
		timeout:  2 * time.Minute,
		interval: 3 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := fn(ctx); err != nil {
			if errors.Is(err, ErrAbort) {
				return fmt.Errorf("retry aborted: %w", errors.Unwrap(err))
			}
			lastErr = err
			slog.Debug(fmt.Sprintf("retry: %v", err))
		} else {
			return nil
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("retry timed out, last error: %w", lastErr)
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func Times(ctx context.Context, n int, fn func(context.Context) error) error {
	var lastErr error
	for i := 0; i < n; i++ {
		if err := fn(ctx); err != nil {
			if errors.Is(err, ErrAbort) {
				return fmt.Errorf("retry aborted: %w", errors.Unwrap(err))
			}
			lastErr = err
			slog.Debug(fmt.Sprintf("retry attempt %d/%d: %v", i+1, n, err))
			continue
		}
		return nil
	}
	return fmt.Errorf("retry exhausted after %d attempts, last error: %w", n, lastErr)
}
