package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreCommandHasExpectedFlags(t *testing.T) {
	app := NewK3sctl()

	var found bool
	for _, cmd := range app.Commands {
		if cmd.Name == "restore" {
			found = true

			flagNames := make(map[string]bool)
			for _, f := range cmd.Flags {
				for _, n := range f.Names() {
					flagNames[n] = true
				}
			}

			assert.True(t, flagNames["config"], "missing --config flag")
			assert.True(t, flagNames["from"], "missing --from flag")
			assert.True(t, flagNames["force"], "missing --force flag")
			assert.True(t, flagNames["concurrency"], "missing --concurrency flag")
			break
		}
	}
	require.True(t, found, "restore command not registered")
}
