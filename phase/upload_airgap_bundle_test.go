package phase

import (
	"testing"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestUploadAirgapBundle_ShouldRun_Disabled(t *testing.T) {
	p := &UploadAirgapBundle{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
		Hosts: cluster.Hosts{&cluster.Host{}},
	}}

	p.Prepare()
	if p.ShouldRun() {
		t.Error("ShouldRun() = true, want false when airgap is not configured")
	}
}

func TestUploadAirgapBundle_ShouldRun_EnabledNoBundle(t *testing.T) {
	p := &UploadAirgapBundle{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s: &cluster.K3s{
			Version: "v1.30.0+k3s1",
			Airgap:  &cluster.AirgapConfig{Enabled: true},
		},
		Hosts: cluster.Hosts{&cluster.Host{}},
	}}

	p.Prepare()
	if p.ShouldRun() {
		t.Error("ShouldRun() = true, want false when airgap enabled but no image bundle specified")
	}
}

func TestUploadAirgapBundle_ShouldRun_EnabledWithBundle(t *testing.T) {
	p := &UploadAirgapBundle{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s: &cluster.K3s{
			Version: "v1.30.0+k3s1",
			Airgap: &cluster.AirgapConfig{
				Enabled:     true,
				ImageBundle: "/tmp/k3s-airgap-images.tar.zst",
			},
		},
		Hosts: cluster.Hosts{&cluster.Host{}},
	}}

	p.Prepare()
	if !p.ShouldRun() {
		t.Error("ShouldRun() = false, want true when airgap enabled with image bundle")
	}
}

func TestUploadAirgapBundle_ShouldRun_NilK3s(t *testing.T) {
	p := &UploadAirgapBundle{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		Hosts: cluster.Hosts{&cluster.Host{}},
	}}

	p.Prepare()
	if p.ShouldRun() {
		t.Error("ShouldRun() = true, want false when K3s config is nil")
	}
}

func TestUploadAirgapBundle_Title(t *testing.T) {
	p := &UploadAirgapBundle{}
	if got := p.Title(); got != "Upload air-gap image bundle" {
		t.Errorf("Title() = %q, want 'Upload air-gap image bundle'", got)
	}
}
