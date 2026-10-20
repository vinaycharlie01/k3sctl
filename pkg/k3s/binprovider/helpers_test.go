package binprovider

import (
	"testing"
)

func TestStageTempPath(t *testing.T) {
	got := stageTempPath("v1.30.0+k3s1")
	want := "/tmp/k3s-v1.30.0+k3s1.tmp"
	if got != want {
		t.Errorf("stageTempPath() = %q, want %q", got, want)
	}
}

func TestStagedFile_StagedPath(t *testing.T) {
	s := &stagedFile{path: "/tmp/k3s-v1.30.0+k3s1.tmp"}
	if got := s.StagedPath(); got != s.path {
		t.Errorf("StagedPath() = %q, want %q", got, s.path)
	}

	var empty stagedFile
	if got := empty.StagedPath(); got != "" {
		t.Errorf("empty StagedPath() = %q, want empty", got)
	}
}
