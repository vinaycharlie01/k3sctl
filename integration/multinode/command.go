package multinode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Result captures the outcome of an external command.
type Result struct {
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// CommandRunner runs an external command.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeCommandRunner -o ./mocks/fake_command_runner.go . CommandRunner
type CommandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (Result, error)
}

// ExecRunner runs commands with os/exec using an explicit argument array, so no
// shell interpolation happens.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	isolateProcessGroup(cmd)
	// Bound how long Wait blocks on output pipes that outlive the command. Without
	// this, a grandchild that inherited the pipes keeps Wait blocked after the
	// command itself has been killed.
	cmd.WaitDelay = time.Second

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	result := Result{
		Args:     append([]string{name}, args...),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	if err != nil && ctx.Err() != nil {
		result.ExitCode = -1
		return result, fmt.Errorf("run %s: %w", strings.Join(result.Args, " "), ctx.Err())
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, fmt.Errorf("%s failed with exit code %d: %s", strings.Join(result.Args, " "), result.ExitCode, tail(result.Stderr, 20))
		}
		result.ExitCode = -1
		return result, fmt.Errorf("run %s: %w", strings.Join(result.Args, " "), err)
	}

	return result, nil
}

// tail returns the last lines of output for compact error messages.
func tail(output string, lines int) string {
	trimmed := strings.TrimRight(output, "\n")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.Join(parts, "\n")
}
