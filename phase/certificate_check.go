package phase

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type CertInfo struct {
	Host     string
	Name     string
	Subject  string
	NotAfter string
	DaysLeft int
	Error    string
}

type CertificateCheck struct {
	GenericPhase
	Results []CertInfo
	mu      sync.Mutex
}

func (p *CertificateCheck) Title() string {
	return "Check K3s certificates"
}

func (p *CertificateCheck) ShouldRun() bool {
	return p.Config.Spec.Hosts.Servers().Count() > 0
}

func (p *CertificateCheck) Run(ctx context.Context) error {
	servers := p.Config.Spec.Hosts.Servers()
	return p.parallelDo(ctx, servers, func(ctx context.Context, h *cluster.Host) error {
		tlsDir := fmt.Sprintf("%s/server/tls", h.K3sDataDir())

		output, err := h.Sudo().ExecOutput(fmt.Sprintf("find %s -name '*.crt' -type f 2>/dev/null", shellQuote(tlsDir)))
		if err != nil {
			log.Debugf("%s: no TLS directory found: %v", h, err)
			return nil
		}

		certPaths := strings.Split(strings.TrimSpace(output), "\n")
		for _, certPath := range certPaths {
			certPath = strings.TrimSpace(certPath)
			if certPath == "" {
				continue
			}

			info := p.checkCert(h, certPath)
			p.mu.Lock()
			p.Results = append(p.Results, info)
			p.mu.Unlock()
		}

		return nil
	})
}

func (p *CertificateCheck) checkCert(h *cluster.Host, certPath string) CertInfo {
	name := strings.TrimSuffix(filepath.Base(certPath), ".crt")
	info := CertInfo{
		Host: h.String(),
		Name: name,
	}

	output, err := h.Sudo().ExecOutput(fmt.Sprintf("openssl x509 -in %s -noout -enddate -subject 2>/dev/null", shellQuote(certPath)))
	if err != nil {
		info.Error = fmt.Sprintf("openssl failed: %v", err)
		return info
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "notAfter=") {
			dateStr := strings.TrimPrefix(line, "notAfter=")
			expiry, parseErr := parseOpenSSLDate(dateStr)
			if parseErr != nil {
				info.Error = fmt.Sprintf("parse date: %v", parseErr)
				continue
			}
			info.NotAfter = expiry.Format("2006-01-02")
			info.DaysLeft = int(time.Until(expiry).Hours() / 24)
		}
		if strings.HasPrefix(line, "subject=") {
			info.Subject = strings.TrimSpace(strings.TrimPrefix(line, "subject="))
		}
	}

	return info
}

func parseOpenSSLDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	// openssl outputs dates like "Oct 20 12:00:00 2025 GMT"
	// Single-digit days may be space-padded: "Oct  5 12:00:00 2025 GMT"
	layouts := []string{
		"Jan  2 15:04:05 2006 MST",
		"Jan 2 15:04:05 2006 MST",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", s)
}
