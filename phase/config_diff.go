package phase

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

// DiffEntry represents a single difference between the desired configuration
// and the actual cluster state.
type DiffEntry struct {
	Host     string
	Field    string
	Current  string
	Desired  string
}

// ConfigDiff compares the desired configuration against the gathered facts and
// writes a human-readable diff to Writer.
type ConfigDiff struct {
	GenericPhase
	Writer  io.Writer
	Entries []DiffEntry
}

func (p *ConfigDiff) Title() string {
	return "Compare configuration"
}

func (p *ConfigDiff) Run(_ context.Context) error {
	p.Entries = nil

	targetVersion := p.Config.Spec.K3s.Version
	for _, h := range p.Config.Spec.Hosts {
		p.diffHost(h, targetVersion)
	}

	if len(p.Entries) == 0 {
		fmt.Fprintln(p.Writer, "No differences found. Cluster matches the configuration.")
		return nil
	}

	fmt.Fprintf(p.Writer, "%d difference(s) found:\n\n", len(p.Entries))

	maxHost, maxField := 4, 5
	for _, e := range p.Entries {
		if len(e.Host) > maxHost {
			maxHost = len(e.Host)
		}
		if len(e.Field) > maxField {
			maxField = len(e.Field)
		}
	}

	headerFmt := fmt.Sprintf("%%-%ds  %%-%ds  %%s  →  %%s\n", maxHost, maxField)
	rowFmt := headerFmt

	fmt.Fprintf(p.Writer, headerFmt, "HOST", "FIELD", "CURRENT", "DESIRED")
	fmt.Fprintf(p.Writer, headerFmt,
		strings.Repeat("-", maxHost),
		strings.Repeat("-", maxField),
		strings.Repeat("-", 7),
		strings.Repeat("-", 7))

	for _, e := range p.Entries {
		fmt.Fprintf(p.Writer, rowFmt, e.Host, e.Field, e.Current, e.Desired)
	}

	return nil
}

func (p *ConfigDiff) diffHost(h *cluster.Host, targetVersion string) {
	hostname := h.String()

	// K3s version
	installed := effectiveInstalledVersion(h)
	if installed == "" && !h.Metadata.K3sServiceInstalled {
		p.addEntry(hostname, "k3s", "(not installed)", targetVersion)
	} else if installed != "" && installed != targetVersion {
		p.addEntry(hostname, "k3s-version", installed, targetVersion)
	}

	// Service state
	if h.Metadata.K3sServiceInstalled && !h.Metadata.K3sServiceRunning {
		p.addEntry(hostname, "service", "stopped", "running")
	}

	// Install flags
	if h.Metadata.K3sInstallFlagsKnown {
		desired := h.K3sInstallFlags()
		current := h.Metadata.K3sCurrentInstallFlags
		if !desired.Equals(current) {
			p.addEntry(hostname, "install-flags", formatFlags(current), formatFlags(desired))
		}
	}

	// K3s config.yaml
	if h.Metadata.K3sExistingConfig != "" {
		p.diffConfig(hostname, h)
	}
}

func (p *ConfigDiff) diffConfig(hostname string, h *cluster.Host) {
	existing := parseConfigKeys(h.Metadata.K3sExistingConfig)
	desired := p.desiredConfigKeys(h)

	allKeys := make(map[string]struct{})
	for k := range existing {
		allKeys[k] = struct{}{}
	}
	for k := range desired {
		allKeys[k] = struct{}{}
	}

	sorted := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	for _, key := range sorted {
		cur, hasCur := existing[key]
		des, hasDes := desired[key]

		switch {
		case hasCur && !hasDes:
			p.addEntry(hostname, "config:"+key, cur, "(removed)")
		case !hasCur && hasDes:
			p.addEntry(hostname, "config:"+key, "(unset)", des)
		case cur != des:
			p.addEntry(hostname, "config:"+key, cur, des)
		}
	}
}

func (p *ConfigDiff) desiredConfigKeys(h *cluster.Host) map[string]string {
	result := make(map[string]string)
	if p.Config.Spec.K3s == nil {
		return result
	}
	for k, v := range p.Config.Spec.K3s.Config {
		result[k] = fmt.Sprintf("%v", v)
	}
	if h.PrivateAddress != "" {
		result["node-ip"] = h.PrivateAddress
	}
	return result
}

func parseConfigKeys(raw string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		result[key] = val
	}
	return result
}

func formatFlags(f cluster.Flags) string {
	if len(f) == 0 {
		return "(none)"
	}
	sorted := make([]string, len(f))
	copy(sorted, f)
	sort.Strings(sorted)
	return strings.Join(sorted, " ")
}

func (p *ConfigDiff) addEntry(host, field, current, desired string) {
	p.Entries = append(p.Entries, DiffEntry{
		Host:    host,
		Field:   field,
		Current: current,
		Desired: desired,
	})
}
