package phase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

var requiredModules = []string{
	"br_netfilter",
	"overlay",
	"ip_tables",
	"nf_conntrack",
}

var requiredPorts = map[string][]int{
	"server": {6443, 10250, 2379, 2380},
	"agent":  {10250},
}

type Preflight struct {
	GenericPhase
}

func (p *Preflight) Title() string {
	return "Pre-flight checks"
}

func (p *Preflight) ShouldRun() bool {
	cfg := p.Config.Spec.K3s
	if cfg == nil || cfg.Preflight == nil {
		return false
	}
	return cfg.Preflight.Enabled
}

func (p *Preflight) Run(ctx context.Context) error {
	pf := p.Config.Spec.K3s.Preflight
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		var warnings []string

		if pf.CheckCgroups {
			if w := p.checkCgroups(h); w != "" {
				warnings = append(warnings, w)
			}
		}

		if pf.CheckModules {
			for _, mod := range requiredModules {
				if w := p.checkModule(h, mod); w != "" {
					warnings = append(warnings, w)
				}
			}
		}

		if pf.CheckPorts {
			ports := requiredPorts[string(h.Role)]
			for _, port := range ports {
				if w := p.checkPort(h, port); w != "" {
					warnings = append(warnings, w)
				}
			}
		}

		if pf.CheckDisk {
			minGB := pf.MinDiskGB
			if minGB <= 0 {
				minGB = 5
			}
			if w := p.checkDisk(h, minGB); w != "" {
				warnings = append(warnings, w)
			}
		}

		if len(warnings) > 0 {
			for _, w := range warnings {
				slog.Warn(fmt.Sprintf("%s: preflight: %s", h, w))
			}
			return fmt.Errorf("%s: %d preflight warning(s)", h, len(warnings))
		}

		slog.Info(fmt.Sprintf("%s: preflight checks passed", h))
		return nil
	})
}

func (p *Preflight) checkCgroups(h *cluster.Host) string {
	output, err := h.Sudo().ExecOutput("cat /proc/cgroups 2>/dev/null | grep -c '^[a-z]' || echo 0")
	if err != nil {
		return "unable to read cgroups"
	}
	count, _ := strconv.Atoi(strings.TrimSpace(output))
	if count == 0 {
		return "no cgroup controllers found — K3s requires cgroups v1 or v2"
	}

	// Check cgroup v2 unified hierarchy
	_, err = h.Sudo().ExecOutput("test -f /sys/fs/cgroup/cgroup.controllers && echo v2")
	if err != nil {
		// cgroups v1 — check for memory and cpu
		for _, ctrl := range []string{"memory", "cpu"} {
			out, err := h.Sudo().ExecOutput(fmt.Sprintf("grep -c '^%s' /proc/cgroups 2>/dev/null || echo 0", ctrl))
			if err != nil || strings.TrimSpace(out) == "0" {
				return fmt.Sprintf("cgroup controller %q not available", ctrl)
			}
		}
	}
	return ""
}

func (p *Preflight) checkModule(h *cluster.Host, name string) string {
	// Check if loaded or built-in
	cmd := fmt.Sprintf("lsmod | grep -q '^%s ' || grep -q '^%s ' /proc/modules || test -f /sys/module/%s/initstate 2>/dev/null",
		name, name, name)
	if err := h.Sudo().Exec(cmd); err != nil {
		return fmt.Sprintf("kernel module %q not loaded", name)
	}
	return ""
}

func (p *Preflight) checkPort(h *cluster.Host, port int) string {
	cmd := fmt.Sprintf("ss -tlnp 2>/dev/null | grep -q ':%d ' && echo in-use || echo free", port)
	output, err := h.Sudo().ExecOutput(cmd)
	if err != nil {
		return ""
	}
	if strings.TrimSpace(output) == "in-use" {
		return fmt.Sprintf("port %d already in use", port)
	}
	return ""
}

func (p *Preflight) checkDisk(h *cluster.Host, minGB int) string {
	dataDir := h.K3sDataDir()
	cmd := fmt.Sprintf("df -BG %s 2>/dev/null | tail -1 | awk '{print $4}'", shellQuote(dataDir))
	output, err := h.Sudo().ExecOutput(cmd)
	if err != nil {
		// Directory may not exist yet; check parent
		cmd = "df -BG / 2>/dev/null | tail -1 | awk '{print $4}'"
		output, err = h.Sudo().ExecOutput(cmd)
		if err != nil {
			return "unable to check disk space"
		}
	}
	avail := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(output), "G"))
	gb, _ := strconv.Atoi(avail)
	if gb < minGB {
		return fmt.Sprintf("only %dG disk space available (minimum: %dG)", gb, minGB)
	}
	return ""
}
