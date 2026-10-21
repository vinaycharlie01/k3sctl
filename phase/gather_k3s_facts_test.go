package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGatherK3sFacts_Title(t *testing.T) {
	p := &GatherK3sFacts{}
	assert.Equal(t, "Gather K3s facts", p.Title())
}

func TestParseK3sVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard output",
			input:    "k3s version v1.28.4+k3s2 (39a45f1b)\ngo version go1.21.4",
			expected: "v1.28.4+k3s2",
		},
		{
			name:     "version only line",
			input:    "k3s version v1.29.0+k3s1 (abc123)",
			expected: "v1.29.0+k3s1",
		},
		{
			name:     "empty output",
			input:    "",
			expected: "",
		},
		{
			name:     "no version line",
			input:    "go version go1.21.4\nsome other output",
			expected: "",
		},
		{
			name:     "incomplete version line",
			input:    "k3s version",
			expected: "",
		},
		{
			name:     "version with extra whitespace",
			input:    "  k3s version v1.27.1+k3s1 (commit)  ",
			expected: "v1.27.1+k3s1",
		},
		{
			name:     "partial match - only prefix and space",
			input:    "k3s version ",
			expected: "",
		},
		{
			name:     "version among multiline output",
			input:    "some preamble\nk3s version v1.30.0+k3s1 (abc123)\ngo version go1.22",
			expected: "v1.30.0+k3s1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseK3sVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
