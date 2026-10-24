package phase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOpenSSLDateVariousFormats(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantDay int
		wantErr bool
	}{
		{"double-digit day", "Oct 20 12:00:00 2025 GMT", 20, false},
		{"single-digit day space-padded", "Oct  5 12:00:00 2025 GMT", 5, false},
		{"january", "Jan 15 08:30:00 2026 GMT", 15, false},
		{"december", "Dec 31 23:59:59 2025 GMT", 31, false},
		{"first of month", "Mar  1 00:00:00 2025 GMT", 1, false},
		{"ISO format rejected", "2025-10-20T12:00:00Z", 0, true},
		{"empty string", "", 0, true},
		{"unix timestamp rejected", "1729382400", 0, true},
		{"partial date rejected", "Oct 20 2025", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOpenSSLDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDay, got.Day())
		})
	}
}

func TestParseOpenSSLDateReturnsCorrectTimezone(t *testing.T) {
	got, err := parseOpenSSLDate("Oct 20 12:00:00 2025 GMT")
	require.NoError(t, err)
	assert.Equal(t, "GMT", got.Location().String())
}

func TestCertInfoDefaultsToZeroValues(t *testing.T) {
	info := CertInfo{}
	assert.Empty(t, info.Host)
	assert.Empty(t, info.Name)
	assert.Empty(t, info.Subject)
	assert.Empty(t, info.NotAfter)
	assert.Zero(t, info.DaysLeft)
	assert.Empty(t, info.Error)
}

func TestCertInfoDaysLeftCalculation(t *testing.T) {
	// A cert expiring 365 days from now should have ~365 DaysLeft
	future := time.Now().Add(365 * 24 * time.Hour)
	daysLeft := int(time.Until(future).Hours() / 24)
	assert.InDelta(t, 365, daysLeft, 1)
}

func TestEtcdSnapshotFields(t *testing.T) {
	s := EtcdSnapshot{
		Name:    "snap-001",
		Size:    "2.1M",
		Created: "2024-12-18T23:04:50Z",
	}
	assert.Equal(t, "snap-001", s.Name)
	assert.Equal(t, "2.1M", s.Size)
	assert.Equal(t, "2024-12-18T23:04:50Z", s.Created)
}
