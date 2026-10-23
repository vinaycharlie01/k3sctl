package phase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type TokenInfo struct {
	Host  string
	Type  string
	Token string
}

type TokenList struct {
	GenericPhase
	Tokens []TokenInfo
}

func (p *TokenList) Title() string {
	return "List K3s tokens"
}

func (p *TokenList) ShouldRun() bool {
	return p.Config.Spec.Hosts.Servers().Count() > 0
}

func (p *TokenList) Run(_ context.Context) error {
	for _, h := range p.Config.Spec.Hosts.Servers() {
		if !h.Metadata.K3sServiceRunning {
			log.Infof("%s: k3s not running, skipping", h)
			continue
		}

		// Node token (used by agents to join)
		output, err := h.Sudo().ExecOutput(fmt.Sprintf("cat %s 2>/dev/null", shellQuote(h.Configurer.K3sTokenPath())))
		if err == nil {
			token := strings.TrimSpace(output)
			if token != "" {
				p.Tokens = append(p.Tokens, TokenInfo{Host: h.String(), Type: "node", Token: token})
				log.Debugf("%s: found node token", h)
			}
		}

		// Server token (used by servers to join each other)
		serverTokenPath := fmt.Sprintf("%s/server/token", h.K3sDataDir())
		output, err = h.Sudo().ExecOutput(fmt.Sprintf("cat %s 2>/dev/null", shellQuote(serverTokenPath)))
		if err == nil {
			token := strings.TrimSpace(output)
			if token != "" {
				p.Tokens = append(p.Tokens, TokenInfo{Host: h.String(), Type: "server", Token: token})
				log.Debugf("%s: found server token", h)
			}
		}

		// Bootstrap tokens via k3s token list (K3s 1.30+, may not exist)
		output, err = h.Sudo().ExecOutput(h.Configurer.K3sCmdf("token list 2>/dev/null"))
		if err == nil {
			for _, line := range strings.Split(output, "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "TOKEN") {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) >= 1 {
					p.Tokens = append(p.Tokens, TokenInfo{Host: h.String(), Type: "bootstrap", Token: fields[0]})
					log.Debugf("%s: found bootstrap token", h)
				}
			}
		}
	}

	return nil
}
