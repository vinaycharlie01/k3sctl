// Package download transfers URL sources to a host without repeating work that
// a previous run already did.
//
// A download is remembered by writing down the sha256 checksum and metadata of
// the file at the time it was fetched. A later run checksums the destination and
// skips the transfer when the checksum still matches, which is what keeps a
// re-apply from pulling binaries that are already on the host.
//
// The records live in the login user's cache directory on the host rather than
// beside the downloaded file: the destination is typically a root-owned
// directory, and an unexpected file there has no business being.
package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/k0sproject/rig/v2/remotefs"
	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/retry"
)

var ErrChecksumMismatch = errors.New("checksum mismatch")

var errNoCacheDir = errors.New("no usable user cache directory on the host")

const downloadAttempts = 3

// Downloader executes a URL download on a remote host. Typically backed by
// curl or wget through the host's exec.
type Downloader interface {
	DownloadURL(url, dest string) error
}

// Record describes a file that was downloaded to the host.
type Record struct {
	URLDigest   string `json:"urlDigest"`
	Destination string `json:"destination"`

	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Sha256  string    `json:"sha256,omitempty"`

	DownloadedAt time.Time `json:"downloadedAt"`
}

// Tracker downloads URLs to a single host and remembers what it downloaded.
type Tracker struct {
	files remotefs.FS
	user  remotefs.FS
	dl    Downloader
	label string

	dirOnce sync.Once
	dir     string
}

// NewTracker returns a Tracker for a host. The files filesystem is where the
// downloads land (sudo-enabled); user is the login user's FS for records;
// dl is the downloader (typically the host).
func NewTracker(files, user remotefs.FS, dl Downloader, label string) *Tracker {
	return &Tracker{files: files, user: user, dl: dl, label: label}
}

// Verdict is what Needed concluded about a destination.
type Verdict struct {
	Download bool
	Reason   string
	Adopt    bool
}

// Needed reports whether url has to be transferred to dst.
//
// wantSum is the sha256 sum the configuration expects, or empty when it names
// none. A destination verified against its expected sum is the file that was
// asked for, so the transfer is skipped.
func (t *Tracker) Needed(ctx context.Context, dlURL, dst, wantSum string) Verdict {
	stat, err := t.files.Stat(dst)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Debugf("%s: stat %s failed: %s", t.label, dst, err)
		}
		return Verdict{Download: true, Reason: "not downloaded yet"}
	}

	rec := t.loadRecord(dlURL, dst)
	ok, why := usable(rec, stat, wantSum)
	if ok {
		return Verdict{Reason: "matches recorded download"}
	}

	if wantSum == "" {
		return Verdict{Download: true, Reason: why}
	}

	sum, err := t.files.Sha256(dst)
	if err != nil {
		return Verdict{Download: true, Reason: fmt.Sprintf("can not checksum %s: %s", dst, err)}
	}
	if !strings.EqualFold(sum, wantSum) {
		return Verdict{Download: true, Reason: "checksum does not match the configured sha256"}
	}
	return Verdict{Reason: "checksum matches the configured sha256", Adopt: true}
}

// Remember records a destination that already holds the expected file.
func (t *Tracker) Remember(ctx context.Context, dlURL, dst, sum string) {
	if sum == "" {
		return
	}
	stat, err := t.files.Stat(dst)
	if err != nil {
		log.Debugf("%s: not recording %s: %s", t.label, dst, err)
		return
	}
	t.record(dlURL, dst, stat, sum)
}

// Fetch downloads url to dst on the host and records what was downloaded.
// When wantSum is set the file is verified against it.
func (t *Tracker) Fetch(ctx context.Context, dlURL, dst, wantSum string) error {
	err := retry.Times(ctx, downloadAttempts, func(_ context.Context) error {
		return t.dl.DownloadURL(dlURL, dst)
	})
	if err != nil {
		return fmt.Errorf("download to %s: %w", dst, err)
	}

	if wantSum != "" {
		if err := t.verify(dst, wantSum); err != nil {
			return err
		}
	}

	stat, err := t.files.Stat(dst)
	if err != nil {
		log.Debugf("%s: stat %s after download failed: %s", t.label, dst, err)
	}
	t.record(dlURL, dst, stat, wantSum)
	return nil
}

func (t *Tracker) verify(dst, wantSum string) error {
	sum, err := t.files.Sha256(dst)
	if err != nil {
		_ = t.files.Remove(dst)
		return fmt.Errorf("checksum %s: %w", dst, err)
	}
	if strings.EqualFold(sum, wantSum) {
		return nil
	}
	_ = t.files.Remove(dst)
	return fmt.Errorf("%w: %s: expected sha256 %s, got %s", ErrChecksumMismatch, dst, wantSum, sum)
}

func usable(rec *Record, stat fs.FileInfo, wantSum string) (bool, string) {
	switch {
	case rec == nil:
		return false, "no record of an earlier download"
	case rec.Size != stat.Size():
		return false, fmt.Sprintf("file size %d does not match the downloaded %d", stat.Size(), rec.Size)
	case rec.ModTime.IsZero():
		return false, "the earlier download was recorded without a modification time"
	case !rec.ModTime.Equal(stat.ModTime()):
		return false, fmt.Sprintf("file was modified at %s, the download left it at %s", stat.ModTime().UTC(), rec.ModTime.UTC())
	case wantSum != "" && !strings.EqualFold(rec.Sha256, wantSum):
		return false, "configured sha256 differs from the downloaded file's sum"
	}
	return true, ""
}

func (t *Tracker) record(dlURL, dst string, stat fs.FileInfo, sum string) {
	rec := Record{
		URLDigest:    urlDigest(dlURL),
		Destination:  dst,
		Size:         -1,
		Sha256:       sum,
		DownloadedAt: time.Now().UTC(),
	}
	if stat != nil {
		rec.Size = stat.Size()
		rec.ModTime = stat.ModTime()
	}
	if rec.ModTime.IsZero() {
		log.Debugf("%s: not recording the download of %s: the destination could not be measured", t.label, dst)
		return
	}
	if sum == "" {
		log.Debugf("%s: not recording the download of %s: nothing to compare it by later", t.label, dst)
		return
	}

	data, err := json.Marshal(rec)
	if err != nil {
		log.Debugf("%s: failed to marshal the download record for %s: %s", t.label, dst, err)
		return
	}
	dir, err := t.recordsDir()
	if err != nil {
		log.Debugf("%s: not recording the download of %s: %s", t.label, dst, err)
		return
	}
	if err := t.user.MkdirAll(dir, 0o700); err != nil {
		log.Debugf("%s: failed to create %s: %s", t.label, dir, err)
		return
	}
	path := t.user.Join(dir, recordName(dlURL, dst))
	if err := t.user.WriteFile(path, data, 0o600); err != nil {
		log.Debugf("%s: failed to write the download record %s: %s", t.label, path, err)
		return
	}
	log.Debugf("%s: recorded the download of %s in %s", t.label, dst, path)
}

func (t *Tracker) forget(dlURL, dst string) {
	dir, err := t.recordsDir()
	if err != nil {
		return
	}
	path := t.user.Join(dir, recordName(dlURL, dst))
	if err := t.user.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Debugf("%s: failed to remove the download record %s: %s", t.label, path, err)
	}
}

func (t *Tracker) loadRecord(dlURL, dst string) *Record {
	dir, err := t.recordsDir()
	if err != nil {
		log.Debugf("%s: no download records available: %s", t.label, err)
		return nil
	}
	path := t.user.Join(dir, recordName(dlURL, dst))
	data, err := t.user.ReadFile(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Debugf("%s: failed to read the download record %s: %s", t.label, path, err)
		}
		return nil
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		log.Debugf("%s: failed to parse the download record %s: %s", t.label, path, err)
		return nil
	}
	if rec.URLDigest != urlDigest(dlURL) || rec.Destination != dst {
		log.Debugf("%s: the download record %s is for a different download (%s), ignoring it", t.label, path, rec.Destination)
		return nil
	}
	return &rec
}

func (t *Tracker) recordsDir() (string, error) {
	t.dirOnce.Do(func() {
		cache := t.user.UserCacheDir()
		if !isAbs(cache) {
			log.Debugf("%s: no usable user cache directory (%q), download records are disabled", t.label, cache)
			return
		}
		t.dir = t.user.Join(cache, "k3sctl", "downloads")
	})
	if t.dir == "" {
		return "", errNoCacheDir
	}
	return t.dir, nil
}

func urlDigest(dlURL string) string {
	sum := sha256.Sum256([]byte(dlURL))
	return hex.EncodeToString(sum[:])
}

func recordName(dlURL, dst string) string {
	sum := sha256.Sum256([]byte(dlURL + "\x00" + dst))
	return filepath.Base(dst) + "-" + hex.EncodeToString(sum[:8]) + ".json"
}

func isAbs(path string) bool {
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) {
		return true
	}
	if len(path) > 2 && path[1] == ':' && (path[2] == '/' || path[2] == '\\') {
		return true
	}
	return false
}
