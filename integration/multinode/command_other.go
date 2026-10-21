//go:build !unix

package multinode

import "os/exec"

// isolateProcessGroup is a no-op on platforms without process groups. The
// default cancellation still kills the direct child, and Cmd.WaitDelay bounds
// how long Wait blocks on pipes held open by any remaining children.
func isolateProcessGroup(*exec.Cmd) {}
