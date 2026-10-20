package version

import (
	"strings"

	"github.com/carlmjohnson/versioninfo"
)

var (
	Version     = versioninfo.Version
	GitCommit   = versioninfo.Revision
	Environment = "development"
)

func IsPre() bool {
	return strings.Contains(Version, "-")
}
