package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSnapshotListParsesStandardOutput(t *testing.T) {
	output := `Name                             Location     Size    Created
on-demand-server-1734567890      /var/lib/rancher/k3s/server/db/snapshots/on-demand-server-1734567890  2.1M   2024-12-18T23:04:50Z
etcd-snapshot-server-1734481490  /var/lib/rancher/k3s/server/db/snapshots/etcd-snapshot-server-1734481490  2.0M   2024-12-17T23:04:50Z`

	snapshots := parseSnapshotList(output)

	assert.Len(t, snapshots, 2)
	assert.Equal(t, "on-demand-server-1734567890", snapshots[0].Name)
	assert.Equal(t, "2.1M", snapshots[0].Size)
	assert.Equal(t, "2024-12-18T23:04:50Z", snapshots[0].Created)
	assert.Equal(t, "etcd-snapshot-server-1734481490", snapshots[1].Name)
}

func TestParseSnapshotListSkipsHeaderLine(t *testing.T) {
	output := "Name                             Location     Size    Created\n"
	snapshots := parseSnapshotList(output)
	assert.Empty(t, snapshots)
}

func TestParseSnapshotListSkipsEmptyLines(t *testing.T) {
	output := "\n\n\n"
	snapshots := parseSnapshotList(output)
	assert.Empty(t, snapshots)
}

func TestParseSnapshotListSkipsShortLines(t *testing.T) {
	output := "partial-data only-two-fields\nName\n"
	snapshots := parseSnapshotList(output)
	assert.Empty(t, snapshots)
}

func TestParseSnapshotListHandlesEmptyOutput(t *testing.T) {
	snapshots := parseSnapshotList("")
	assert.Empty(t, snapshots)
}

func TestParseSnapshotListHandlesSingleSnapshot(t *testing.T) {
	output := "on-demand-snap  /var/lib/rancher/k3s/server/db/snapshots/on-demand-snap  1.5M  2024-12-01T10:00:00Z"
	snapshots := parseSnapshotList(output)
	assert.Len(t, snapshots, 1)
	assert.Equal(t, "on-demand-snap", snapshots[0].Name)
	assert.Equal(t, "1.5M", snapshots[0].Size)
	assert.Equal(t, "2024-12-01T10:00:00Z", snapshots[0].Created)
}

func TestParseSnapshotListTrimsWhitespace(t *testing.T) {
	output := "  snap-001  /path/to/snap  500K  2024-01-01T00:00:00Z  \n"
	snapshots := parseSnapshotList(output)
	assert.Len(t, snapshots, 1)
	assert.Equal(t, "snap-001", snapshots[0].Name)
}
