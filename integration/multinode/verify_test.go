package multinode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const allReadyJSON = `{"items":[
{"metadata":{"name":"server"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"agent1"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"agent2"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`

const agentNotReadyJSON = `{"items":[
{"metadata":{"name":"server"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"agent1"},"status":{"conditions":[{"type":"Ready","status":"False"}]}},
{"metadata":{"name":"agent2"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`

const extraNodeJSON = `{"items":[
{"metadata":{"name":"server"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"agent1"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"agent2"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"stale"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`

func TestParseNodes(t *testing.T) {
	nodes, err := ParseNodes(allReadyJSON)
	if err != nil {
		t.Fatalf("ParseNodes() error = %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("len(nodes) = %d, want 3", len(nodes))
	}
	if nodes[0].Name != "server" || !nodes[0].Ready {
		t.Errorf("nodes[0] = %+v, want server Ready", nodes[0])
	}

	t.Run("not ready", func(t *testing.T) {
		nodes, err := ParseNodes(agentNotReadyJSON)
		if err != nil {
			t.Fatalf("ParseNodes() error = %v", err)
		}
		if len(nodes) != 3 || nodes[1].Name != "agent1" || nodes[1].Ready {
			t.Errorf("nodes[1] = %+v, want agent1 not Ready", nodes[1])
		}
	})

	t.Run("missing Ready condition", func(t *testing.T) {
		nodes, err := ParseNodes(`{"items":[{"metadata":{"name":"server"}}]}`)
		if err != nil {
			t.Fatalf("ParseNodes() error = %v", err)
		}
		if len(nodes) != 1 || nodes[0].Ready {
			t.Errorf("nodes = %+v, want a single not Ready node", nodes)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		if _, err := ParseNodes("not json"); err == nil {
			t.Fatal("ParseNodes() error = nil, want a parse error")
		}
	})
}

func TestVerifyNodes(t *testing.T) {
	expected := []string{"server", "agent1", "agent2"}

	tests := []struct {
		name     string
		output   string
		expected []string
		wantErr  string
	}{
		{name: "all ready", output: allReadyJSON, expected: expected},
		{name: "one not ready", output: agentNotReadyJSON, expected: expected, wantErr: "not Ready"},
		{name: "extra ready node", output: extraNodeJSON, expected: expected, wantErr: "expected 3 Ready nodes, got 4"},
		{name: "unexpected node name", output: allReadyJSON, expected: []string{"server", "agent1", "other"}, wantErr: `node "other" is not Ready`},
		{name: "malformed output", output: "not json", expected: expected, wantErr: "parse node list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyNodes(tt.output, tt.expected)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("VerifyNodes() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("VerifyNodes() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyVersion(t *testing.T) {
	const output = "k3s version v1.37.1+k3s1 (f0d1e5b7)\ngo version go1.22.5\n"

	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{name: "matching", output: output, want: "v1.37.1+k3s1"},
		{name: "mismatch", output: output, want: "v1.30.0+k3s1", wantErr: true},
		{name: "empty expected", output: output, want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyVersion(tt.output, tt.want)
			if tt.wantErr && err == nil {
				t.Fatal("VerifyVersion() error = nil, want an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("VerifyVersion() error = %v", err)
			}
		})
	}
}

func TestVerifyKubeconfig(t *testing.T) {
	const output = "apiVersion: v1\nclusters:\n- cluster:\n    server: https://10.0.0.10:6443\n"

	if err := VerifyKubeconfig(output, "https://10.0.0.10:6443"); err != nil {
		t.Fatalf("VerifyKubeconfig() error = %v", err)
	}
	if err := VerifyKubeconfig(output, "https://10.0.0.99:6443"); err == nil {
		t.Fatal("VerifyKubeconfig() error = nil, want a mismatch error")
	}
}

func TestVerifyEtcdMembers(t *testing.T) {
	const tableOutput = `+------------------+---------+--------------------+---------------------------+---------------------------+------------+
|        ID        | STATUS  |        NAME        |        PEER ADDRS         |       CLIENT ADDRS        | IS LEARNER |
+------------------+---------+--------------------+---------------------------+---------------------------+------------+
| 8e9e05c52164694d | started | server1            | https://10.0.0.1:2380     | https://10.0.0.1:2379     | false      |
| 3a1b2c3d4e5f6a7b | started | server2            | https://10.0.0.2:2380     | https://10.0.0.2:2379     | false      |
| a1b2c3d4e5f6a7b8 | started | server3            | https://10.0.0.3:2380     | https://10.0.0.3:2379     | false      |
+------------------+---------+--------------------+---------------------------+---------------------------+------------+`

	const simpleOutput = `8e9e05c52164694d, started, server1, https://10.0.0.1:2380, https://10.0.0.1:2379, false
3a1b2c3d4e5f6a7b, started, server2, https://10.0.0.2:2380, https://10.0.0.2:2379, false
a1b2c3d4e5f6a7b8, started, server3, https://10.0.0.3:2380, https://10.0.0.3:2379, false`

	tests := []struct {
		name     string
		output   string
		expected int
		wantErr  string
	}{
		{name: "table output 3 members", output: tableOutput, expected: 3},
		{name: "simple output 3 members", output: simpleOutput, expected: 3},
		{name: "table output wrong count", output: tableOutput, expected: 5, wantErr: "etcd cluster has 3 members, want 5"},
		{name: "simple output wrong count", output: simpleOutput, expected: 2, wantErr: "etcd cluster has 3 members, want 2"},
		{name: "empty output", output: "", expected: 1, wantErr: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyEtcdMembers(tt.output, tt.expected)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("VerifyEtcdMembers() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("VerifyEtcdMembers() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyEtcdMembersFromAPI(t *testing.T) {
	const validJSON = `{"header":{"cluster_id":"12345"},"members":[` +
		`{"ID":"10276657743932975437","name":"server1","peerURLs":["https://10.0.0.1:2380"],"clientURLs":["https://10.0.0.1:2379"]},` +
		`{"ID":"4211926507498291319","name":"server2","peerURLs":["https://10.0.0.2:2380"],"clientURLs":["https://10.0.0.2:2379"]},` +
		`{"ID":"11634834085411903084","name":"server3","peerURLs":["https://10.0.0.3:2380"],"clientURLs":["https://10.0.0.3:2379"]}` +
		`]}`

	tests := []struct {
		name     string
		output   string
		expected int
		wantErr  string
	}{
		{name: "3 members match", output: validJSON, expected: 3},
		{name: "wrong count", output: validJSON, expected: 5, wantErr: "etcd cluster has 3 members, want 5"},
		{name: "empty output", output: "", expected: 1, wantErr: "empty"},
		{name: "invalid json", output: "not json", expected: 1, wantErr: "parse etcd member list response"},
		{name: "single member", output: `{"members":[{"ID":"123","name":"s1","peerURLs":["https://10.0.0.1:2380"]}]}`, expected: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyEtcdMembersFromAPI(tt.output, tt.expected)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("VerifyEtcdMembersFromAPI() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("VerifyEtcdMembersFromAPI() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestCheckAPIURL(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{name: "ok", status: http.StatusOK},
		{name: "unauthorized", status: http.StatusUnauthorized},
		{name: "forbidden", status: http.StatusForbidden},
		{name: "server error", status: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			err := checkAPIURL(context.Background(), server.URL)
			if tt.wantErr && err == nil {
				t.Fatal("checkAPIURL() error = nil, want an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("checkAPIURL() error = %v", err)
			}
		})
	}

	t.Run("unreachable", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := server.URL
		server.Close()

		if err := checkAPIURL(context.Background(), url); err == nil {
			t.Fatal("checkAPIURL() error = nil, want a connection error")
		}
	})
}
