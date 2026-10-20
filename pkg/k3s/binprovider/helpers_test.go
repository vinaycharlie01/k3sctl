package binprovider

import (
	"testing"
)

func TestStageTempPath(t *testing.T) {
	got := stageTempPath(&FakeHost{}, "v1.30.0+k3s1")
	want := "/tmp/k3s-v1.30.0+k3s1.tmp"
	if got != want {
		t.Errorf("stageTempPath() = %q, want %q", got, want)
	}
}

func TestStagedFile_StagedPath(t *testing.T) {
	s := &stagedFile{host: &FakeHost{}, path: "/tmp/k3s-v1.30.0+k3s1.tmp"}
	if got := s.StagedPath(); got != s.path {
		t.Errorf("StagedPath() = %q, want %q", got, s.path)
	}

	var empty stagedFile
	if got := empty.StagedPath(); got != "" {
		t.Errorf("empty StagedPath() = %q, want empty", got)
	}
}

func TestNeedsUpgrade(t *testing.T) {
	tests := []struct {
		name      string
		installed string
		target    string
		want      bool
	}{
		{"fresh host", "", "v1.30.0+k3s1", true},
		{"same version", "v1.30.0+k3s1", "v1.30.0+k3s1", false},
		{"different version", "v1.29.0+k3s1", "v1.30.0+k3s1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &FakeHost{}
			host.InstalledK3sVersionReturns(tt.installed)
			if got := needsUpgrade(host, tt.target); got != tt.want {
				t.Errorf("needsUpgrade() = %v, want %v", got, tt.want)
			}
		})
	}
}
