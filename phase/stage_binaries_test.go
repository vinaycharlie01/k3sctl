package phase

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestStageBinaries_Title(t *testing.T) {
	p := &StageBinaries{}
	assert.Equal(t, "Stage K3s binaries", p.Title())
}

func TestResolveBinaryProviders_Title(t *testing.T) {
	p := &ResolveBinaryProviders{}
	assert.Equal(t, "Resolve K3s binary providers", p.Title())
}

func TestStageBinaries_Prepare_FiltersNeedsUpgrade(t *testing.T) {
	needsUpgrade := &cluster.Host{
		Role: cluster.RoleServer,
		Metadata: cluster.HostMetadata{
			NeedsUpgrade:   true,
			BinaryProvider: &fakeBinaryProvider{},
		},
	}
	noUpgrade := &cluster.Host{
		Role: cluster.RoleServer,
		Metadata: cluster.HostMetadata{
			NeedsUpgrade:   false,
			BinaryProvider: &fakeBinaryProvider{},
		},
	}
	noProvider := &cluster.Host{
		Role: cluster.RoleServer,
		Metadata: cluster.HostMetadata{
			NeedsUpgrade: true,
		},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{needsUpgrade, noUpgrade, noProvider},
		},
	}
	p := &StageBinaries{}
	p.Config = cfg
	p.Prepare()
	assert.Len(t, p.hosts, 1)
	assert.Equal(t, needsUpgrade, p.hosts[0])
}

func TestStageBinaries_ShouldRun_NoHosts(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &StageBinaries{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestStageBinaries_ShouldRun_WithHosts(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{
					Role: cluster.RoleServer,
					Metadata: cluster.HostMetadata{
						NeedsUpgrade:   true,
						BinaryProvider: &fakeBinaryProvider{},
					},
				},
			},
		},
	}
	p := &StageBinaries{}
	p.Config = cfg
	p.Prepare()
	assert.True(t, p.ShouldRun())
}

type fakeBinaryProvider struct {
	stageErr     error
	stageCalls   int
	cleanupCalls int
}

func (f *fakeBinaryProvider) NeedsUpgrade(_ context.Context) (bool, error) { return false, nil }
func (f *fakeBinaryProvider) IsUpload() bool                               { return false }

func (f *fakeBinaryProvider) Stage(_ context.Context) error {
	f.stageCalls++
	return f.stageErr
}

func (f *fakeBinaryProvider) CleanUp(_ context.Context) error {
	f.cleanupCalls++
	return nil
}

func TestStageBinaries_Run_StagesBinary(t *testing.T) {
	provider := &fakeBinaryProvider{}
	host := &cluster.Host{
		Role: cluster.RoleServer,
		Metadata: cluster.HostMetadata{
			NeedsUpgrade:   true,
			BinaryProvider: provider,
		},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{host},
			K3s:   &cluster.K3s{Version: "v1.29.0+k3s1"},
		},
	}
	phase := &StageBinaries{}
	phase.Config = cfg
	phase.Manager = &Manager{ConcurrentUploads: 1}
	phase.Prepare()

	require.NoError(t, phase.Run(context.Background()))
	assert.Equal(t, 1, provider.stageCalls)
}

func TestStageBinaries_Run_PropagatesStageError(t *testing.T) {
	provider := &fakeBinaryProvider{stageErr: fmt.Errorf("download failed")}
	host := &cluster.Host{
		Role: cluster.RoleServer,
		Metadata: cluster.HostMetadata{
			NeedsUpgrade:   true,
			BinaryProvider: provider,
		},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{host},
			K3s:   &cluster.K3s{Version: "v1.29.0+k3s1"},
		},
	}
	phase := &StageBinaries{}
	phase.Config = cfg
	phase.Manager = &Manager{ConcurrentUploads: 1}
	phase.Prepare()

	err := phase.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage binary")
}

func TestStageBinaries_CleanUp(t *testing.T) {
	provider := &fakeBinaryProvider{}
	host := &cluster.Host{
		Role: cluster.RoleServer,
		Metadata: cluster.HostMetadata{
			NeedsUpgrade:   true,
			BinaryProvider: provider,
		},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{host},
			K3s:   &cluster.K3s{Version: "v1.29.0+k3s1"},
		},
	}
	phase := &StageBinaries{}
	phase.Config = cfg
	phase.Manager = &Manager{ConcurrentUploads: 1}
	phase.Prepare()

	phase.CleanUp()
	assert.Equal(t, 1, provider.cleanupCalls)
}

func TestStageBinaries_Run_NilProvider(t *testing.T) {
	// A host with NeedsUpgrade but nil BinaryProvider should be filtered
	// out by Prepare, so it never reaches Run. Verify Prepare does filter.
	host := &cluster.Host{
		Role: cluster.RoleServer,
		Metadata: cluster.HostMetadata{
			NeedsUpgrade: true,
		},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{host},
		},
	}
	phase := &StageBinaries{}
	phase.Config = cfg
	phase.Prepare()

	assert.False(t, phase.ShouldRun())
}
