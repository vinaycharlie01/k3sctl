package multinode

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestExecRunnerCapturesStdout(t *testing.T) {
	result, err := ExecRunner{}.Run(context.Background(), "", "sh", "-c", "echo hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("Stdout = %q, want it to contain %q", result.Stdout, "hello")
	}
	if !slices.Equal(result.Args, []string{"sh", "-c", "echo hello"}) {
		t.Errorf("Args = %v, want the full argument vector", result.Args)
	}
	if result.Duration <= 0 {
		t.Errorf("Duration = %v, want a positive duration", result.Duration)
	}
}

func TestExecRunnerCapturesStderrAndExitCode(t *testing.T) {
	result, err := ExecRunner{}.Run(context.Background(), "", "sh", "-c", "echo boom >&2; exit 3")
	if err == nil {
		t.Fatal("Run() error = nil, want a non-zero exit error")
	}
	if result.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", result.ExitCode)
	}
	if !strings.Contains(result.Stderr, "boom") {
		t.Errorf("Stderr = %q, want it to contain %q", result.Stderr, "boom")
	}
	if !strings.Contains(err.Error(), "exit code 3") {
		t.Errorf("error = %v, want it to mention the exit code", err)
	}
}

func TestExecRunnerReportsStartFailure(t *testing.T) {
	result, err := ExecRunner{}.Run(context.Background(), "", "k3sctl-missing-binary")
	if err == nil {
		t.Fatal("Run() error = nil, want a start failure")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestExecRunnerHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := ExecRunner{}.Run(ctx, "", "sh", "-c", "sleep 5")
	if err == nil {
		t.Fatal("Run() error = nil, want the command to be cancelled")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
	if result.Duration > 3*time.Second {
		t.Errorf("Duration = %v, want the command to be cancelled early", result.Duration)
	}
}

func TestTail(t *testing.T) {
	tests := []struct {
		name   string
		output string
		lines  int
		want   string
	}{
		{name: "shorter than limit", output: "a\nb", lines: 3, want: "a\nb"},
		{name: "exactly at limit", output: "a\nb\nc", lines: 3, want: "a\nb\nc"},
		{name: "truncated", output: "a\nb\nc\nd", lines: 3, want: "b\nc\nd"},
		{name: "trailing newline", output: "a\nb\n", lines: 3, want: "a\nb"},
		{name: "empty", output: "", lines: 3, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tail(tt.output, tt.lines); got != tt.want {
				t.Errorf("tail(%q, %d) = %q, want %q", tt.output, tt.lines, got, tt.want)
			}
		})
	}
}
