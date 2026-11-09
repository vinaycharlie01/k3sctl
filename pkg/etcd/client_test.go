package etcd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindStaleMembers(t *testing.T) {
	members := []Member{
		{ID: 0x8e9e05c52164694d, Name: "server-1", PeerURLs: []string{"https://10.0.0.1:2380"}},
		{ID: 0x3a1b2c3d4e5f6a7b, Name: "server-2", PeerURLs: []string{"https://10.0.0.2:2380"}},
		{ID: 0xa1b2c3d4e5f6a7b8, Name: "server-old", PeerURLs: []string{"https://10.0.0.99:2380"}},
	}

	known := map[string]bool{
		"10.0.0.1": true,
		"10.0.0.2": true,
	}

	t.Run("detects stale member", func(t *testing.T) {
		stale := FindStaleMembers(members, known)
		assert.Len(t, stale, 1)
		assert.Equal(t, uint64(0xa1b2c3d4e5f6a7b8), stale[0].ID)
		assert.Contains(t, stale[0].PeerURLs[0], "10.0.0.99")
	})

	t.Run("no stale when all known", func(t *testing.T) {
		allKnown := map[string]bool{
			"10.0.0.1":  true,
			"10.0.0.2":  true,
			"10.0.0.99": true,
		}
		stale := FindStaleMembers(members, allKnown)
		assert.Empty(t, stale)
	})

	t.Run("empty members", func(t *testing.T) {
		stale := FindStaleMembers(nil, known)
		assert.Empty(t, stale)
	})

	t.Run("member with no peer URLs", func(t *testing.T) {
		noPeers := []Member{
			{ID: 123, Name: "orphan"},
		}
		stale := FindStaleMembers(noPeers, known)
		assert.Len(t, stale, 1)
	})
}

func TestFindMemberByPeerAddr(t *testing.T) {
	members := []Member{
		{ID: 1, Name: "server-1", PeerURLs: []string{"https://10.0.0.1:2380"}},
		{ID: 2, Name: "server-2", PeerURLs: []string{"https://10.0.0.2:2380"}},
	}

	t.Run("finds by address", func(t *testing.T) {
		m := FindMemberByPeerAddr(members, "10.0.0.1")
		assert.NotNil(t, m)
		assert.Equal(t, "server-1", m.Name)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		m := FindMemberByPeerAddr(members, "10.0.0.99")
		assert.Nil(t, m)
	})

	t.Run("empty list", func(t *testing.T) {
		m := FindMemberByPeerAddr(nil, "10.0.0.1")
		assert.Nil(t, m)
	})
}
