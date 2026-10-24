package multinode

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestResultFields(t *testing.T) {
	r := Result{
		Args:     []string{"ssh", "10.0.0.1", "uname -r"},
		Stdout:   "5.15.0-generic\n",
		Stderr:   "",
		ExitCode: 0,
		Duration: 150 * time.Millisecond,
	}
	if len(r.Args) != 3 {
		t.Errorf("len(Args) = %d, want 3", len(r.Args))
	}
	if r.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", r.ExitCode)
	}
	if r.Duration != 150*time.Millisecond {
		t.Errorf("Duration = %v, want 150ms", r.Duration)
	}
}

func TestResultWithNonZeroExitCode(t *testing.T) {
	r := Result{
		Args:     []string{"ssh", "host", "false"},
		ExitCode: 1,
		Stderr:   "error occurred\n",
	}
	if r.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", r.ExitCode)
	}
	if !strings.Contains(r.Stderr, "error") {
		t.Errorf("Stderr = %q, want it to contain 'error'", r.Stderr)
	}
}

func TestTailWithFewerLines(t *testing.T) {
	input := "line1\nline2\n"
	got := tail(input, 10)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Errorf("tail() returned %d lines, want 2", len(lines))
	}
}

func TestTailWithExactLines(t *testing.T) {
	input := "a\nb\nc\n"
	got := tail(input, 3)
	if !strings.Contains(got, "a") || !strings.Contains(got, "c") {
		t.Errorf("tail(3) = %q, want all 3 lines", got)
	}
}

func TestTailWithMoreLines(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	input := strings.Join(lines, "\n")
	got := tail(input, 5)
	gotLines := strings.Split(strings.TrimSpace(got), "\n")
	if len(gotLines) != 5 {
		t.Errorf("tail(5) returned %d lines, want 5", len(gotLines))
	}
}

func TestTailEmptyInput(t *testing.T) {
	got := tail("", 5)
	if got != "" {
		t.Errorf("tail('', 5) = %q, want empty", got)
	}
}

func TestExecRunnerCapturesOutput(t *testing.T) {
	if _, err := exec.LookPath("echo"); err != nil {
		t.Skip("echo not available")
	}

	runner := &ExecRunner{}
	result, err := runner.Run(context.Background(), t.TempDir(), "echo", "hello", "world")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("Stdout = %q, want 'hello world'", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestExecRunnerCapturesExitCode(t *testing.T) {
	if _, err := exec.LookPath("false"); err != nil {
		t.Skip("false not available")
	}

	runner := &ExecRunner{}
	result, err := runner.Run(context.Background(), t.TempDir(), "false")
	if err == nil {
		t.Fatal("Run() error = nil, want an error for exit code 1")
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}

func TestExecRunnerReportsContextCancellation(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	runner := &ExecRunner{}
	_, err := runner.Run(ctx, t.TempDir(), "sleep", "10")
	if err == nil {
		t.Fatal("Run() error = nil, want a context error")
	}
}

func TestExecRunnerReportsNonExistentBinary(t *testing.T) {
	runner := &ExecRunner{}
	_, err := runner.Run(context.Background(), t.TempDir(), "/nonexistent/binary")
	if err == nil {
		t.Fatal("Run() error = nil, want an error for missing binary")
	}
}

func TestExecRunnerSetsWorkingDirectory(t *testing.T) {
	if _, err := exec.LookPath("pwd"); err != nil {
		t.Skip("pwd not available")
	}

	dir := t.TempDir()
	runner := &ExecRunner{}
	result, err := runner.Run(context.Background(), dir, "pwd")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(result.Stdout, dir) {
		t.Errorf("Stdout = %q, want it to contain %q", result.Stdout, dir)
	}
}
