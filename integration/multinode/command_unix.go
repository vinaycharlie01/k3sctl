//go:build unix

package multinode

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// isolateProcessGroup puts the command in its own process group so that
// cancellation terminates the whole tree. Killing only the direct child leaves
// grandchildren running, and any that inherited the output pipes keep Wait
// blocked until they exit on their own.
func isolateProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}
