package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextSuccess(t *testing.T) {
	calls := 0
	err := Context(context.Background(), func(_ context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	}, Timeout(5*time.Second), Interval(10*time.Millisecond))
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestContextTimeout(t *testing.T) {
	err := Context(context.Background(), func(_ context.Context) error {
		return errors.New("always fails")
	}, Timeout(100*time.Millisecond), Interval(10*time.Millisecond))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry timed out")
}

func TestContextAbort(t *testing.T) {
	err := Context(context.Background(), func(_ context.Context) error {
		return errors.Join(ErrAbort, errors.New("fatal error"))
	}, Timeout(5*time.Second), Interval(10*time.Millisecond))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry aborted")
}

func TestTimesSuccess(t *testing.T) {
	calls := 0
	err := Times(context.Background(), 5, func(_ context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestTimesExhausted(t *testing.T) {
	err := Times(context.Background(), 3, func(_ context.Context) error {
		return errors.New("always fails")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry exhausted after 3 attempts")
}
