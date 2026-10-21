package multinode

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const diagnosticCommandTimeout = 30 * time.Second

// DiagnosticTarget describes a node to collect remote diagnostics from.
type DiagnosticTarget struct {
	Name    string
	Address string
	Service string
}

// CollectDiagnostics gathers platform and remote diagnostics. Failures are
// embedded in the output; the function never returns an error so a broken node
// cannot prevent the remaining diagnostics from being collected.
func CollectDiagnostics(ctx context.Context, remote RemoteExecutor, targets []DiagnosticTarget, platform func(context.Context) (string, error)) string {
	var b strings.Builder

	if platform != nil {
		b.WriteString("=== Platform ===\n")
		if output, err := platform(ctx); err != nil {
			fmt.Fprintf(&b, "error: %v\n", err)
		} else {
			b.WriteString(ensureTrailingNewline(output))
		}
		b.WriteString("\n")
	}

	for _, target := range targets {
		fmt.Fprintf(&b, "=== %s (%s) ===\n", target.Name, target.Address)
		for _, command := range diagnosticCommands(target.Service) {
			commandCtx, cancel := context.WithTimeout(ctx, diagnosticCommandTimeout)
			result, err := remote.Run(commandCtx, target.Address, command)
			cancel()

			fmt.Fprintf(&b, "--- %s ---\n", command)
			b.WriteString(ensureTrailingNewline(result.Stdout))
			b.WriteString(ensureTrailingNewline(result.Stderr))
			if err != nil {
				fmt.Fprintf(&b, "error: %v\n", err)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func diagnosticCommands(service string) []string {
	return []string{
		"ip addr",
		fmt.Sprintf("systemctl status %s --no-pager", service),
		fmt.Sprintf("sudo journalctl -u %s --no-pager -n 100", service),
		"free -m",
		"df -h",
		"ls -la /dev/kmsg",
		"cloud-init status --long",
		"findmnt -o TARGET,PROPAGATION /",
	}
}

func ensureTrailingNewline(value string) string {
	if value == "" || strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}
