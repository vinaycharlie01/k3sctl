package phase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type GatherK3sFacts struct {
	GenericPhase
}

func (p *GatherK3sFacts) Title() string {
	return "Gather K3s facts"
}

func (p *GatherK3sFacts) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		binaryPath := h.K3sBinaryDir()

		if h.Sudo().FS().FileExist(binaryPath) {
			output, err := h.Sudo().ExecOutput(fmt.Sprintf("%s --version", binaryPath))
			if err == nil {
				ver := parseK3sVersion(output)
				if ver != "" {
					h.Metadata.K3sInstalledVersion = ver
					log.Infof("%s: installed k3s version: %s", h, ver)
				}
			}
		}

		serviceName := h.K3sServiceName()
		svc, err := h.Sudo().Service(serviceName)
		if err == nil {
			h.Metadata.K3sServiceInstalled = true
			if svc.IsRunning(ctx) {
				h.Metadata.K3sServiceRunning = true
				h.Metadata.K3sRunningVersion = h.Metadata.K3sInstalledVersion
				log.Infof("%s: k3s service %s is running", h, serviceName)
			}
		}

		unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
		if h.Sudo().FS().FileExist(unitPath) {
			output, err := h.Sudo().ExecOutput(fmt.Sprintf("cat %s", shellQuote(unitPath)))
			if err == nil {
				h.Metadata.K3sCurrentInstallFlags = parseExecStartFlags(output)
				h.Metadata.K3sInstallFlagsKnown = true
			}
		}

		if h.IsServer() {
			tokenPath := h.Configurer.K3sTokenPath()
			output, err := h.Sudo().ExecOutput(fmt.Sprintf("cat %s 2>/dev/null", shellQuote(tokenPath)))
			if err == nil && strings.TrimSpace(output) != "" {
				h.Metadata.NodeToken = strings.TrimSpace(output)
				log.Debugf("%s: found node token", h)
			}
		}

		configPath := h.K3sConfigDir()
		if h.Sudo().FS().FileExist(configPath) {
			output, err := h.Sudo().ExecOutput(fmt.Sprintf("cat %s", shellQuote(configPath)))
			if err == nil {
				h.Metadata.K3sExistingConfig = output
			}
		}

		return nil
	})
}

func parseK3sVersion(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "k3s version") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return ""
}

// parseExecStartFlags extracts the K3s command line flags from a systemd unit's
// ExecStart directive, dropping the binary path and the server/agent subcommand.
func parseExecStartFlags(unit string) cluster.Flags {
	flags := cluster.Flags{}
	for _, line := range strings.Split(unit, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ExecStart=") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "ExecStart="))
		if len(fields) < 2 {
			continue
		}
		if fields[1] != "server" && fields[1] != "agent" {
			continue
		}
		flags = append(flags, fields[2:]...)
	}
	return flags
}
