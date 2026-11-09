package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Member represents an etcd cluster member.
type Member struct {
	ID       uint64
	Name     string
	PeerURLs []string
}

// Client wraps an etcd v3 client and provides the operations k3sctl
// previously performed by shelling out to etcdctl or curling the etcd
// gRPC gateway on a remote host.
type Client interface {
	MemberList(ctx context.Context) ([]Member, error)
	MemberRemove(ctx context.Context, id uint64) error
	Close() error
}

type client struct {
	c *clientv3.Client
}

// New creates an etcd v3 client using TLS certificates read from a K3s
// server node. The PEM-encoded CA certificate, client certificate and
// client key are the contents of the files under
// /var/lib/rancher/k3s/server/tls/etcd/ on the server.
func New(ctx context.Context, endpoints []string, caCert, cert, key []byte) (Client, error) {
	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, fmt.Errorf("parse etcd client certificate: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("parse etcd CA certificate")
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}

	c, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		TLS:         tlsCfg,
		DialTimeout: 10 * time.Second,
		Context:     ctx,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to etcd: %w", err)
	}

	return &client{c: c}, nil
}

func (c *client) MemberList(ctx context.Context) ([]Member, error) {
	resp, err := c.c.MemberList(ctx)
	if err != nil {
		return nil, fmt.Errorf("etcd member list: %w", err)
	}

	members := make([]Member, len(resp.Members))
	for i, m := range resp.Members {
		members[i] = Member{
			ID:       m.ID,
			Name:     m.Name,
			PeerURLs: m.PeerURLs,
		}
	}
	return members, nil
}

func (c *client) MemberRemove(ctx context.Context, id uint64) error {
	_, err := c.c.MemberRemove(ctx, id)
	if err != nil {
		return fmt.Errorf("etcd member remove %x: %w", id, err)
	}
	return nil
}

func (c *client) Close() error {
	return c.c.Close()
}

// FindStaleMembers checks a member list and returns members whose peer
// URLs do not contain any of the known addresses.
func FindStaleMembers(members []Member, knownAddrs map[string]bool) []Member {
	var stale []Member
	for _, m := range members {
		peerURL := ""
		if len(m.PeerURLs) > 0 {
			peerURL = m.PeerURLs[0]
		}

		found := false
		for addr := range knownAddrs {
			if strings.Contains(peerURL, addr) {
				found = true
				break
			}
		}
		if !found {
			stale = append(stale, m)
		}
	}
	return stale
}

// FindMemberByPeerAddr returns the member whose peer URL contains the
// given address, or nil if none matches.
func FindMemberByPeerAddr(members []Member, peerAddr string) *Member {
	for i, m := range members {
		for _, u := range m.PeerURLs {
			if strings.Contains(u, peerAddr) {
				return &members[i]
			}
		}
	}
	return nil
}
