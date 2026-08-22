package doctor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckTools_AlwaysSucceeds(t *testing.T) {
	// age and sops are embedded as Go libraries, so there is nothing to check.
	assert.NoError(t, CheckTools())
}

func TestCheckToolsVerbose_PrintsEmbeddedLibraries(t *testing.T) {
	var ok bool
	output := captureStdout(t, func() {
		ok = CheckToolsVerbose()
	})

	assert.True(t, ok)
	assert.Contains(t, output, "Embedded libraries (no installation needed):")
	assert.Contains(t, output, "filippo.io/age")
	assert.Contains(t, output, "github.com/YewFence/sops/v3")
	assert.Contains(t, output, "No external tools are required.")
}
