//go:build integration

package localhost_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vinaycharlie01/k3sctl/integration/localhost"
)

const defaultK3sVersion = "v1.37.1+k3s1"

// TestLocalhostSingleNode installs a single-node K3s cluster on the current
// machine using localhost: true, verifies the cluster, resets it, and confirms
// teardown.
//
// It requires:
//   - K3SCTL_LOCALHOST_INTEGRATION=1
//   - K3SCTL_BINARY pointing at the compiled k3sctl binary
//   - systemd as PID 1
//   - passwordless sudo
//
// Local reproduction:
//
//	task build
//	K3SCTL_LOCALHOST_INTEGRATION=1 \
//	K3SCTL_BINARY="$PWD/k3sctl" \
//	  go test -tags=integration -timeout 30m -v ./integration/localhost/ \
//	  -run TestLocalhostSingleNode
func TestLocalhostSingleNode(t *testing.T) {
	if os.Getenv("K3SCTL_LOCALHOST_INTEGRATION") != "1" {
		t.Skip("set K3SCTL_LOCALHOST_INTEGRATION=1 to run the localhost integration test")
	}

	binary := os.Getenv("K3SCTL_BINARY")
	if binary == "" {
		t.Fatal("K3SCTL_BINARY must point at the compiled k3sctl binary")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("K3SCTL_BINARY %q is not usable: %v", binary, err)
	}

	k3sVersion := os.Getenv("K3S_VERSION")
	if k3sVersion == "" {
		k3sVersion = defaultK3sVersion
	}

	runID := os.Getenv("K3SCTL_RUN_ID")
	if runID == "" {
		runID = fmt.Sprintf("local-%d", time.Now().Unix())
	}

	artifactDir := os.Getenv("K3SCTL_ARTIFACT_DIR")
	if artifactDir == "" {
		artifactDir = filepath.Join("integration-artifacts", runID)
	}

	t.Logf("run id: %s", runID)
	t.Logf("artifacts: %s", artifactDir)
	t.Logf("k3s version: %s", k3sVersion)

	workDir := t.TempDir()

	runner, err := localhost.NewRunner(localhost.Options{
		Binary:      binary,
		WorkDir:     workDir,
		ArtifactDir: artifactDir,
		ClusterName: "k3sctl-" + runID,
		K3sVersion:  k3sVersion,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("localhost single-node integration test failed: %v", err)
	}
}
