package node

import (
	"encoding/json"
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
		{"a'b'c", "'a'\\''b'\\''c'"},
		{"$(whoami)", "'$(whoami)'"},
		{"`id`", "'`id`'"},
		{"foo;bar", "'foo;bar'"},
		{"hello\nworld", "'hello\nworld'"},
	}

	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.expected {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestKubeNodeStatusParsing(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantReady bool
		wantErr   bool
	}{
		{
			name:      "node ready",
			json:      `{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`,
			wantReady: true,
		},
		{
			name:      "node not ready",
			json:      `{"status":{"conditions":[{"type":"Ready","status":"False"}]}}`,
			wantReady: false,
		},
		{
			name:      "no ready condition",
			json:      `{"status":{"conditions":[{"type":"DiskPressure","status":"False"}]}}`,
			wantReady: false,
			wantErr:   true,
		},
		{
			name:      "empty conditions",
			json:      `{"status":{"conditions":[]}}`,
			wantReady: false,
			wantErr:   true,
		},
		{
			name: "multiple conditions",
			json: `{"status":{"conditions":[
				{"type":"DiskPressure","status":"False"},
				{"type":"MemoryPressure","status":"False"},
				{"type":"Ready","status":"True"}
			]}}`,
			wantReady: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &kubeNodeStatus{}
			if err := json.Unmarshal([]byte(tt.json), status); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			ready := false
			foundCondition := false
			for _, c := range status.Status.Conditions {
				if c.Type == "Ready" {
					foundCondition = true
					ready = c.Status == "True"
					break
				}
			}

			if tt.wantErr && foundCondition && ready {
				t.Error("expected error condition but got ready")
			}
			if !tt.wantErr && !foundCondition {
				t.Error("expected to find Ready condition")
			}
			if !tt.wantErr && ready != tt.wantReady {
				t.Errorf("ready = %v, want %v", ready, tt.wantReady)
			}
		})
	}
}

func TestKubeNodeStatusUnmarshalInvalid(t *testing.T) {
	status := &kubeNodeStatus{}
	err := json.Unmarshal([]byte("not json"), status)
	if err == nil {
		t.Error("expected unmarshal error for invalid JSON")
	}
}
