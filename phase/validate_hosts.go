package phase

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/configurer"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ValidateHosts struct {
	GenericPhase
}

func (p *ValidateHosts) Title() string {
	return "Validate hosts"
}

func (p *ValidateHosts) Run(ctx context.Context) error {
	hosts := p.Config.Spec.Hosts

	seen := make(map[string]bool)
	for _, h := range hosts {
		addr := h.Address()
		if seen[addr] {
			return fmt.Errorf("duplicate host address: %s", addr)
		}
		seen[addr] = true
	}

	for _, h := range hosts {
		osKind, err := h.OSKind()
		if err != nil {
			return fmt.Errorf("%s: %w", h, err)
		}
		if osKind != "linux" {
			return fmt.Errorf("%s: K3s only supports Linux hosts (detected: %s)", h, osKind)
		}

		if v, ok := h.Configurer.(configurer.HostValidator); ok {
			if err := v.ValidateHost(h); err != nil {
				return fmt.Errorf("%s: host validation failed: %w", h, err)
			}
		}

		output, err := h.Sudo().ExecOutput("id -u")
		if err != nil {
			return fmt.Errorf("%s: failed to check sudo access: %w", h, err)
		}
		if strings.TrimSpace(output) != "0" {
			slog.Warn(fmt.Sprintf("%s: sudo user is not root (uid: %s), some operations may fail", h, strings.TrimSpace(output)))
		}
	}

	return p.validateClockSkew(ctx, hosts)
}

func (p *ValidateHosts) validateClockSkew(ctx context.Context, hosts cluster.Hosts) error {
	slog.Info("validating clock skew")

	type hostSkew struct {
		host *cluster.Host
		skew time.Duration
	}

	var mu sync.Mutex
	var skews []hostSkew
	var values []time.Duration

	err := p.parallelDo(ctx, hosts, func(_ context.Context, h *cluster.Host) error {
		output, err := h.ExecOutput("date +%s")
		if err != nil {
			slog.Warn(fmt.Sprintf("%s: failed to check clock: %v", h, err))
			return nil
		}
		secs, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
		if err != nil {
			slog.Warn(fmt.Sprintf("%s: failed to parse remote time %q: %v", h, output, err))
			return nil
		}

		skew := time.Now().UTC().Sub(time.Unix(secs, 0)).Round(time.Second)
		mu.Lock()
		skews = append(skews, hostSkew{host: h, skew: skew})
		values = append(values, skew)
		mu.Unlock()
		return nil
	})
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}

	deviations := clockSkewDeviations(values)
	var exceeded int
	for i, hs := range skews {
		if deviations[i] > maxClockSkew {
			slog.Error(fmt.Sprintf("%s: clock skew of %.0f seconds exceeds the maximum of %.0f seconds", hs.host, deviations[i].Seconds(), maxClockSkew.Seconds()))
			exceeded++
		}
	}

	if exceeded > 0 {
		return fmt.Errorf("clock skew exceeds the maximum on %d host(s)", exceeded)
	}
	return nil
}

// maxClockSkew is the maximum tolerated deviation from the median host clock.
const maxClockSkew = 30 * time.Second

func medianDuration(values []time.Duration) time.Duration {
	sorted := make([]time.Duration, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2]
}

// clockSkewDeviations returns each value's absolute deviation from the median,
// preserving the input order.
func clockSkewDeviations(values []time.Duration) []time.Duration {
	median := medianDuration(values)
	deviations := make([]time.Duration, len(values))
	for i, v := range values {
		deviation := v - median
		if deviation < 0 {
			deviation = -deviation
		}
		deviations[i] = deviation
	}
	return deviations
}
