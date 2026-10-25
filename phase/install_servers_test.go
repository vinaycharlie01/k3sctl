package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestInstallServers_Prepare(t *testing.T) {
	leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer}
	leader.Metadata.IsLeader = true
	leader.Metadata.K3sServiceRunning = true

	notRunning := &cluster.Host{Hostname: "server-2", Role: cluster.RoleServer}
	alreadyRunning := &cluster.Host{Hostname: "server-3", Role: cluster.RoleServer}
	alreadyRunning.Metadata.K3sServiceRunning = true

	agent := &cluster.Host{Hostname: "agent-1", Role: cluster.RoleAgent}

	p := &InstallServers{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		Hosts: cluster.Hosts{leader, notRunning, alreadyRunning, agent},
	}}
	p.Prepare()

	assert.Equal(t, leader, p.leader)
	assert.Len(t, p.hosts, 1)
	assert.Equal(t, notRunning, p.hosts[0])
}

func TestInstallServers_ShouldRun(t *testing.T) {
	t.Run("true when hosts to install", func(t *testing.T) {
		p := &InstallServers{hosts: cluster.Hosts{&cluster.Host{}}}
		assert.True(t, p.ShouldRun())
	})

	t.Run("false when no hosts", func(t *testing.T) {
		p := &InstallServers{}
		assert.False(t, p.ShouldRun())
	})
}

func TestQuorumBatchSize(t *testing.T) {
	tests := []struct {
		running  int
		expected int
	}{
		{0, 1},
		{1, 1},
		{2, 1},
		{3, 1},
		{4, 1},
		{5, 2},
		{7, 3},
		{9, 4},
		{10, 4},
		{11, 5},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := quorumBatchSize(tt.running)
			assert.Equal(t, tt.expected, got, "running=%d", tt.running)
		})
	}
}

func TestQuorumSafeBatches(t *testing.T) {
	makeHosts := func(n int) cluster.Hosts {
		hosts := make(cluster.Hosts, n)
		for i := range hosts {
			hosts[i] = &cluster.Host{Hostname: "new-" + string(rune('a'+i))}
		}
		return hosts
	}

	t.Run("empty hosts", func(t *testing.T) {
		batches := quorumSafeBatches(1, nil)
		assert.Nil(t, batches)
	})

	t.Run("single running, add 1", func(t *testing.T) {
		hosts := makeHosts(1)
		batches := quorumSafeBatches(1, hosts)
		assert.Len(t, batches, 1)
		assert.Len(t, batches[0], 1)
	})

	t.Run("single running, add 2", func(t *testing.T) {
		hosts := makeHosts(2)
		batches := quorumSafeBatches(1, hosts)
		// batch 1: size 1 (running=1 → tolerance=1)
		// batch 2: size 1 (running=2 → tolerance=1)
		assert.Len(t, batches, 2)
		assert.Len(t, batches[0], 1)
		assert.Len(t, batches[1], 1)
	})

	t.Run("single running, add 4", func(t *testing.T) {
		hosts := makeHosts(4)
		batches := quorumSafeBatches(1, hosts)
		// batch 1: 1 (running=1), batch 2: 1 (running=2), batch 3: 1 (running=3), batch 4: 1 (running=4)
		assert.Len(t, batches, 4)
	})

	t.Run("3 running, add 4", func(t *testing.T) {
		hosts := makeHosts(4)
		batches := quorumSafeBatches(3, hosts)
		// batch 1: 1 (running=3, tolerance=1), batch 2: 1 (running=4, tolerance=1), batch 3: 2 (running=5, tolerance=2)
		assert.Len(t, batches, 3)
		assert.Len(t, batches[0], 1)
		assert.Len(t, batches[1], 1)
		assert.Len(t, batches[2], 2)
	})

	t.Run("5 running, add 3", func(t *testing.T) {
		hosts := makeHosts(3)
		batches := quorumSafeBatches(5, hosts)
		// batch 1: 2 (running=5, tolerance=2), batch 2: 1 (running=7, tolerance=3)
		assert.Len(t, batches, 2)
		assert.Len(t, batches[0], 2)
		assert.Len(t, batches[1], 1)
	})

	t.Run("all hosts appear exactly once", func(t *testing.T) {
		hosts := makeHosts(6)
		batches := quorumSafeBatches(1, hosts)
		var allHosts cluster.Hosts
		for _, b := range batches {
			allHosts = append(allHosts, b...)
		}
		assert.Equal(t, len(hosts), len(allHosts))
		for i, h := range hosts {
			assert.Equal(t, h, allHosts[i])
		}
	})
}

func TestInstallServers_CleanUp(t *testing.T) {
	t.Run("cleans up failed hosts only", func(t *testing.T) {
		failed := &cluster.Host{Hostname: "server-2", Role: cluster.RoleServer}
		succeeded := &cluster.Host{Hostname: "server-3", Role: cluster.RoleServer}
		succeeded.Metadata.K3sServiceRunning = true

		p := &InstallServers{
			hosts: cluster.Hosts{failed, succeeded},
		}
		p.Manager = &Manager{DryRun: true}

		// Should not panic; dry-run skips remote calls.
		p.CleanUp()
	})

	t.Run("no-op when all hosts succeeded", func(t *testing.T) {
		running := &cluster.Host{Hostname: "server-2", Role: cluster.RoleServer}
		running.Metadata.K3sServiceRunning = true

		p := &InstallServers{
			hosts: cluster.Hosts{running},
		}

		// Should not panic; nothing to clean up.
		p.CleanUp()
	})

	t.Run("no-op when hosts list is empty", func(t *testing.T) {
		p := &InstallServers{}
		p.CleanUp()
	})
}

func TestPropagateToken_Prepare(t *testing.T) {
	leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer}
	leader.Metadata.IsLeader = true
	leader.Metadata.NodeToken = "abc123"

	withToken := &cluster.Host{Hostname: "server-2", Role: cluster.RoleServer}
	withToken.Metadata.NodeToken = "abc123"

	noToken := &cluster.Host{Hostname: "agent-1", Role: cluster.RoleAgent}

	p := &PropagateToken{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		Hosts: cluster.Hosts{leader, withToken, noToken},
	}}
	p.Prepare()

	assert.Equal(t, leader, p.leader)
	assert.Len(t, p.hosts, 1)
	assert.Equal(t, noToken, p.hosts[0])
}

func TestPropagateToken_ShouldRun(t *testing.T) {
	t.Run("true when hosts need token", func(t *testing.T) {
		leader := &cluster.Host{}
		leader.Metadata.NodeToken = "tok"
		p := &PropagateToken{
			hosts:  cluster.Hosts{&cluster.Host{}},
			leader: leader,
		}
		assert.True(t, p.ShouldRun())
	})

	t.Run("false when no hosts", func(t *testing.T) {
		leader := &cluster.Host{}
		leader.Metadata.NodeToken = "tok"
		p := &PropagateToken{leader: leader}
		assert.False(t, p.ShouldRun())
	})

	t.Run("false when leader has no token", func(t *testing.T) {
		p := &PropagateToken{
			hosts:  cluster.Hosts{&cluster.Host{}},
			leader: &cluster.Host{},
		}
		assert.False(t, p.ShouldRun())
	})
}
