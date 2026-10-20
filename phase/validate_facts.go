package phase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
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
			if h.Metadata.K3sRunningVersion == "" {
				continue
			}
			if h.Reset {
				continue
			}
			if isDowngrade(h.Metadata.K3sRunningVersion, targetVersion) {
				return fmt.Errorf("%s: downgrade from %s to %s is not supported (use --disable-downgrade-check to override)",
					h, h.Metadata.K3sRunningVersion, targetVersion)
			}
		}
	}

	for _, h := range p.Config.Spec.Hosts {
		if h.Metadata.K3sRunningVersion == "" {
			continue
		}
		if h.Reset {
			continue
		}
		if h.Metadata.K3sRunningVersion != targetVersion {
			h.Metadata.NeedsUpgrade = true
			log.Infof("%s: needs upgrade from %s to %s", h, h.Metadata.K3sRunningVersion, targetVersion)
		}
	}

	return nil
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
