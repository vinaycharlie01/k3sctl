package localhost

import (
	"os"
	"strings"
	"testing"
)

func TestNewRunnerRejectsIncompleteOptions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Options)
		want   string
	}{
		{
			name:   "missing binary",
			mutate: func(o *Options) { o.Binary = "" },
			want:   "k3sctl binary path is required",
		},
		{
			name:   "missing work dir",
			mutate: func(o *Options) { o.WorkDir = "" },
			want:   "work directory is required",
		},
		{
			name:   "missing artifact dir",
			mutate: func(o *Options) { o.ArtifactDir = "" },
			want:   "artifact directory is required",
		},
		{
			name:   "missing k3s version",
			mutate: func(o *Options) { o.K3sVersion = "" },
			want:   "k3s version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Binary:      "/usr/local/bin/k3sctl",
				WorkDir:     "/tmp/work",
				ArtifactDir: "/tmp/artifacts",
				K3sVersion:  "v1.30.0+k3s1",
			}
			tt.mutate(&opts)

			_, err := NewRunner(opts)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestNewRunnerSetsDefaults(t *testing.T) {
	opts := Options{
		Binary:      "/usr/local/bin/k3sctl",
		WorkDir:     "/tmp/work",
		ArtifactDir: "/tmp/artifacts",
		K3sVersion:  "v1.30.0+k3s1",
	}

	runner, err := NewRunner(opts)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if runner.opts.ClusterName != "k3sctl-localhost" {
		t.Errorf("ClusterName = %q, want k3sctl-localhost", runner.opts.ClusterName)
	}
	if runner.opts.ApplyTimeout != defaultApplyTimeout {
		t.Errorf("ApplyTimeout = %v, want %v", runner.opts.ApplyTimeout, defaultApplyTimeout)
	}
	if runner.opts.VerifyTimeout != defaultVerifyTimeout {
		t.Errorf("VerifyTimeout = %v, want %v", runner.opts.VerifyTimeout, defaultVerifyTimeout)
	}
	if runner.opts.ResetTimeout != defaultResetTimeout {
		t.Errorf("ResetTimeout = %v, want %v", runner.opts.ResetTimeout, defaultResetTimeout)
	}
}

func TestWriteConfigGeneratesLocalhostManifest(t *testing.T) {
	workDir := t.TempDir()
	artifactDir := t.TempDir()

	runner, err := NewRunner(Options{
		Binary:      "/usr/local/bin/k3sctl",
		WorkDir:     workDir,
		ArtifactDir: artifactDir,
		ClusterName: "test-cluster",
		K3sVersion:  "v1.30.0+k3s1",
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	path, err := runner.writeConfig()
	if err != nil {
		t.Fatalf("writeConfig() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	text := string(data)
	if !strings.Contains(text, "localhost: true") {
		t.Errorf("config missing localhost: true:\n%s", text)
	}
	if !strings.Contains(text, "name: test-cluster") {
		t.Errorf("config missing cluster name:\n%s", text)
	}
	if !strings.Contains(text, "v1.30.0+k3s1") {
		t.Errorf("config missing k3s version:\n%s", text)
	}
	if !strings.Contains(text, "role: server") {
		t.Errorf("config missing role: server:\n%s", text)
	}
}
