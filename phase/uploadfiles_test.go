package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestUploadFiles_Title(t *testing.T) {
	p := &UploadFiles{}
	assert.Equal(t, "Upload files to hosts", p.Title())
}

func TestUploadFiles_Prepare_FiltersHostsWithFiles(t *testing.T) {
	withFiles := &cluster.Host{
		Role: cluster.RoleServer,
		UploadFiles: []cluster.UploadFile{
			{Source: "/tmp/test.txt", DstDir: "/opt/"},
		},
	}
	withoutFiles := &cluster.Host{
		Role: cluster.RoleServer,
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{withFiles, withoutFiles},
		},
	}
	p := &UploadFiles{}
	p.Config = cfg
	p.Prepare()
	assert.Len(t, p.hosts, 1)
	assert.Equal(t, withFiles, p.hosts[0])
}

func TestUploadFiles_ShouldRun_NoFiles(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &UploadFiles{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestUploadFiles_ShouldRun_WithFiles(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{
					Role: cluster.RoleServer,
					UploadFiles: []cluster.UploadFile{
						{Source: "/tmp/test.txt", DstDir: "/opt/"},
					},
				},
			},
		},
	}
	p := &UploadFiles{}
	p.Config = cfg
	p.Prepare()
	assert.True(t, p.ShouldRun())
}

func TestUploadFiles_Prepare_EmptyHosts(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{},
		},
	}
	p := &UploadFiles{}
	p.Config = cfg
	p.Prepare()
	assert.Len(t, p.hosts, 0)
	assert.False(t, p.ShouldRun())
}

func TestUploadFiles_Prepare_MultipleHostsWithFiles(t *testing.T) {
	h1 := &cluster.Host{
		Role: cluster.RoleServer,
		UploadFiles: []cluster.UploadFile{
			{Source: "a.txt", DstDir: "/tmp"},
		},
	}
	h2 := &cluster.Host{
		Role: cluster.RoleAgent,
		UploadFiles: []cluster.UploadFile{
			{Source: "b.txt", DstDir: "/opt"},
		},
	}
	h3 := &cluster.Host{
		Role: cluster.RoleAgent,
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{h1, h2, h3},
		},
	}
	p := &UploadFiles{}
	p.Config = cfg
	p.Prepare()
	assert.Len(t, p.hosts, 2)
}

func TestUploadFiles_Prepare_AllHostsHaveFiles(t *testing.T) {
	h1 := &cluster.Host{
		Role: cluster.RoleServer,
		UploadFiles: []cluster.UploadFile{
			{Source: "a.txt", DstDir: "/tmp"},
		},
	}
	h2 := &cluster.Host{
		Role: cluster.RoleAgent,
		UploadFiles: []cluster.UploadFile{
			{Source: "b.txt", DstDir: "/opt"},
		},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{h1, h2},
		},
	}
	p := &UploadFiles{}
	p.Config = cfg
	p.Prepare()
	assert.Len(t, p.hosts, 2)
	assert.True(t, p.ShouldRun())
}
