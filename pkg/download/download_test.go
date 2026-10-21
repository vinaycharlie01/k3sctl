package download

import (
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubStat struct {
	size    int64
	modTime time.Time
}

func (s stubStat) Name() string       { return "stub" }
func (s stubStat) Size() int64        { return s.size }
func (s stubStat) Mode() fs.FileMode  { return 0o644 }
func (s stubStat) ModTime() time.Time { return s.modTime }
func (s stubStat) IsDir() bool        { return false }
func (s stubStat) Sys() any           { return nil }

func TestUsable(t *testing.T) {
	const sum = "1111111111111111111111111111111111111111111111111111111111111111"
	written := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	onDisk := stubStat{size: 100, modTime: written}
	rec := func(mod func(*Record)) *Record {
		r := &Record{
			URLDigest:   urlDigest("https://example.com/f"),
			Destination: "/tmp/f",
			Size:        100,
			ModTime:     written,
			Sha256:      sum,
		}
		if mod != nil {
			mod(r)
		}
		return r
	}

	t.Run("no record", func(t *testing.T) {
		ok, why := usable(nil, onDisk, "")
		require.False(t, ok)
		require.Equal(t, "no record of an earlier download", why)
	})

	t.Run("matching record", func(t *testing.T) {
		ok, _ := usable(rec(nil), onDisk, sum)
		require.True(t, ok)
	})

	t.Run("file was written to after the download", func(t *testing.T) {
		ok, why := usable(rec(nil), stubStat{size: 120, modTime: written}, "")
		require.False(t, ok)
		require.Contains(t, why, "does not match the downloaded")
	})

	t.Run("file was replaced with content of the same length", func(t *testing.T) {
		ok, why := usable(rec(nil), stubStat{size: 100, modTime: written.Add(time.Minute)}, "")
		require.False(t, ok)
		require.Contains(t, why, "was modified at")
	})

	t.Run("modification time in another zone is the same instant", func(t *testing.T) {
		berlin := time.FixedZone("CEST", 2*60*60)
		ok, _ := usable(rec(nil), stubStat{size: 100, modTime: written.In(berlin)}, sum)
		require.True(t, ok)
	})

	t.Run("record without a modification time", func(t *testing.T) {
		ok, why := usable(rec(func(r *Record) { r.ModTime = time.Time{} }), onDisk, "")
		require.False(t, ok)
		require.Contains(t, why, "without a modification time")
	})

	t.Run("configured sum changed", func(t *testing.T) {
		ok, why := usable(rec(nil), onDisk, "2222222222222222222222222222222222222222222222222222222222222222")
		require.False(t, ok)
		require.Contains(t, why, "configured sha256 differs")
	})

	t.Run("sum configured after the download", func(t *testing.T) {
		ok, _ := usable(rec(func(r *Record) { r.Sha256 = "" }), onDisk, sum)
		require.False(t, ok)
	})

	t.Run("configured sum in a different case", func(t *testing.T) {
		ok, _ := usable(rec(func(r *Record) { r.Sha256 = "AAAA" }), onDisk, "aaaa")
		require.True(t, ok)
	})

	t.Run("no sum configured or recorded", func(t *testing.T) {
		ok, _ := usable(rec(func(r *Record) { r.Sha256 = "" }), onDisk, "")
		require.True(t, ok)
	})
}

func TestRecordRoundTrip(t *testing.T) {
	rec := Record{
		URLDigest:    urlDigest("https://example.com/bundle.tar"),
		Destination:  "/var/lib/rancher/k3s/images/bundle.tar",
		Size:         1234,
		ModTime:      time.Date(2026, 9, 1, 8, 29, 0, 0, time.UTC),
		Sha256:       "1111111111111111111111111111111111111111111111111111111111111111",
		DownloadedAt: time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(rec)
	require.NoError(t, err)

	var got Record
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, rec, got)
}

func TestIsAbs(t *testing.T) {
	for path, want := range map[string]bool{
		"/home/user/.cache":           true,
		`C:\Users\user\AppData\Local`: true,
		"C:/Users/user/AppData/Local": true,
		`\\server\share`:              true,
		".cache":                      false,
		"":                            false,
		"home/user/.cache":            false,
		"C:":                          false,
		"~/.cache":                    false,
	} {
		require.Equal(t, want, isAbs(path), path)
	}
}

func TestRecordKeepsNoURL(t *testing.T) {
	const secret = "https://example.com/bundle.tar?X-Amz-Signature=deadbeefcafe"

	rec := Record{URLDigest: urlDigest(secret), Destination: "/tmp/bundle.tar"}
	data, err := json.Marshal(rec)
	require.NoError(t, err)

	require.NotContains(t, string(data), "deadbeefcafe")
	require.NotContains(t, string(data), "example.com")
	require.Equal(t, urlDigest(secret), rec.URLDigest)
	require.NotEqual(t, urlDigest(secret), urlDigest(secret+"x"))
}

func TestURLDigest(t *testing.T) {
	d1 := urlDigest("https://example.com/a")
	d2 := urlDigest("https://example.com/b")
	require.NotEqual(t, d1, d2)
	require.Len(t, d1, 64)
	require.Equal(t, d1, urlDigest("https://example.com/a"))
}
