package phase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ValidateFacts struct {
	GenericPhase
	DisableDowngradeCheck bool
}

func (p *ValidateFacts) Title() string {
	return "Validate facts"
}

func (p *ValidateFacts) Run(_ context.Context) error {
	targetVersion := p.Config.Spec.K3s.Version
	if targetVersion == "" {
		return fmt.Errorf("target K3s version not set")
	}

	if !p.DisableDowngradeCheck {
		for _, h := range p.Config.Spec.Hosts {
			installed := effectiveInstalledVersion(h)
			if installed == "" {
				continue
			}
			if h.Reset {
				continue
			}
			if isDowngrade(installed, targetVersion) {
				return fmt.Errorf("%s: downgrade from %s to %s is not supported (use --disable-downgrade-check to override)",
					h, installed, targetVersion)
			}
		}
	}

	for _, h := range p.Config.Spec.Hosts {
		if h.Reset {
			continue
		}

		installed := effectiveInstalledVersion(h)
		switch {
		case installed == "":
			h.Metadata.NeedsUpgrade = true
			log.Infof("%s: K3s is not installed, will install %s", h, targetVersion)
		case installed != targetVersion:
			h.Metadata.NeedsUpgrade = true
			log.Infof("%s: needs upgrade from %s to %s", h, installed, targetVersion)
		default:
			h.Metadata.NeedsUpgrade = false
		}

		if h.Metadata.K3sInstallFlagsKnown {
			h.Metadata.NeedsReinstall = h.Metadata.K3sServiceRunning &&
				!h.K3sInstallFlags().Equals(h.Metadata.K3sCurrentInstallFlags)
			if h.Metadata.NeedsReinstall {
				log.Infof("%s: install flags changed, will reinstall service", h)
			}
		} else {
			h.Metadata.NeedsReinstall = false
		}
	}

	return nil
}

// effectiveInstalledVersion prefers the binary version detected on the host and
// falls back to the running version when the binary could not be inspected.
func effectiveInstalledVersion(h *cluster.Host) string {
	if h.Metadata.K3sInstalledVersion != "" {
		return h.Metadata.K3sInstalledVersion
	}
	return h.Metadata.K3sRunningVersion
}

func isDowngrade(current, target string) bool {
	cv := normalizeVersion(current)
	tv := normalizeVersion(target)
	return cv > tv
}

func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.Index(v, "+"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	result := ""
	for _, p := range parts {
		for len(p) < 10 {
			p = "0" + p
		}
		result += p
	}
	return result
}
