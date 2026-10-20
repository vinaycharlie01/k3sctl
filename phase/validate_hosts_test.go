package phase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMedianDuration(t *testing.T) {
	tests := []struct {
		name   string
		values []time.Duration
		want   time.Duration
	}{
		{"single", []time.Duration{5 * time.Second}, 5 * time.Second},
		{"odd count", []time.Duration{3 * time.Second, time.Second, 2 * time.Second}, 2 * time.Second},
		{"even count uses upper median", []time.Duration{4 * time.Second, time.Second, 3 * time.Second, 2 * time.Second}, 3 * time.Second},
		{"negative values", []time.Duration{-3 * time.Second, -time.Second, -2 * time.Second}, -2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, medianDuration(tt.values))
		})
	}
}

func TestClockSkewDeviations(t *testing.T) {
	t.Run("no skew", func(t *testing.T) {
		assert.Equal(t,
			[]time.Duration{0, 0, 0},
			clockSkewDeviations([]time.Duration{time.Second, time.Second, time.Second}),
		)
	})

	t.Run("deviations preserve input order", func(t *testing.T) {
		got := clockSkewDeviations([]time.Duration{
			0,
			10 * time.Second,
			2 * time.Minute,
		})

		// median is 10s
		assert.Equal(t, []time.Duration{10 * time.Second, 0, 110 * time.Second}, got)
	})
}
