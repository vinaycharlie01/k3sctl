# 01: Entry Point, Module Setup, and Version

## Files to Create

### `go.mod`
- **Source**: `k0sctl/go.mod`
- **Module**: `github.com/vinaycharlie01/k3sctl`
- **Go version**: 1.27.0 (match k0sctl)
- **Key dependencies** (same as k0sctl):
  - `github.com/urfave/cli/v2` — CLI framework
  - `github.com/k0sproject/rig/v2` — SSH remote execution
  - `github.com/sirupsen/logrus` — logging
  - `github.com/creasty/defaults` — struct defaults
  - `github.com/jellydator/validation` — validation
  - `github.com/k0sproject/dig` — dynamic YAML mapping
  - `github.com/k0sproject/version` — semver comparison
  - `github.com/logrusorgru/aurora` — colored output
  - `github.com/AlecAivazis/survey/v2` — interactive prompts
  - `github.com/carlmjohnson/versioninfo` — build version info
  - `k8s.io/client-go` — kubeconfig manipulation
  - `gopkg.in/yaml.v2` — YAML parsing
  - `github.com/mattn/go-isatty` — terminal detection
  - `github.com/a8m/envsubst` — env var substitution
  - `github.com/adrg/xdg` — XDG paths
  - `github.com/bmatcuk/doublestar/v4` — glob patterns
  - `github.com/sergi/go-diff` — diff output
  - `github.com/shiena/ansicolor` — Windows ANSI colors
  - `github.com/stretchr/testify` — test assertions
  - `github.com/samber/slog-logrus/v2` — slog bridge for rig
- **Dropped deps**: None of the k0sctl deps are Windows-only; keep them all

### `main.go`
- **Source**: `k0sctl/main.go` (21 lines)
- **Changes**:
  - Import `github.com/vinaycharlie01/k3sctl/cmd`
  - Import `github.com/vinaycharlie01/k3sctl/version` (blank)
  - `cmd.NewK3sctl(os.Stdin, os.Stdout, os.Stderr)`
  - Otherwise identical structure

### `version/version.go`
- **Source**: `k0sctl/version/version.go` (22 lines)
- **Changes**: None — identical structure. Version/GitCommit/Environment injected at build time via ldflags.
- Variables: `Version`, `GitCommit`, `Environment`
- Function: `IsPre() bool` — checks for prerelease

## k0sctl Source Analysis

```go
// main.go (k0sctl) - 21 lines
package main
import (
    "os"
    "github.com/k0sproject/k0sctl/cmd"
    _ "github.com/carlmjohnson/versioninfo"
    _ "github.com/k0sproject/k0sctl/version"
)
func main() {
    k0sctl := cmd.NewK0sctl(os.Stdin, os.Stdout, os.Stderr)
    if err := k0sctl.Run(os.Args); err != nil { log.Fatal(err) }
}
```

```go
// version/version.go (k0sctl) - 22 lines
package version
var (
    Version     = versioninfo.Version
    GitCommit   = versioninfo.Revision
    Environment = "development"
)
func IsPre() bool { return strings.Contains(Version, "-") }
```
