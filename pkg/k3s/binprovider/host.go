package binprovider

import (
	"fmt"
	"io/fs"
	"time"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/remotefs"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeHost -o ./fake_host_test.go . Host

type Host interface {
	fmt.Stringer

	FS() remotefs.FS
	Sudo() *rig.Client

	Arch() (string, error)

	InstalledK3sVersion() string
	RunningK3sVersion() string

	Dir(path string) (string, error)
	OSKind() (string, error)
	DownloadURL(url, dest string) error
	Touch(path string, modTime time.Time) error
	DeleteFile(path string) error
	SetFileMode(path string, mode fs.FileMode) error
}
