//go:build terminal

package tests

import (
	"strings"
	"testing"

	"qsc/tools"

	"github.com/quollix/common/assert"
)

func TestPruneIncompatibleVersionsTerminalWorkflow(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()

	legacy, err := setup.storeClient.AddIncompatibleVersion(terminalNginxApp, "0.1.0")
	assert.Nil(t, err)
	before, err := setup.storeClient.ListVersions(tools.SampleMaintainer, terminalNginxApp)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(before))
	assert.Equal(t, legacy.VersionId, before[0].VersionId)

	output := runClientCommand(t, "versions prune-incompatible -d")
	assert.True(t, strings.Contains(output, "Versions to delete:"))
	assert.True(t, strings.Contains(output, "0.1.0"))
	assert.True(t, strings.Contains(output, "yaml:"))
	assert.False(t, strings.Contains(output, "Type DELETE"))
	afterPreview, err := setup.storeClient.ListVersions(tools.SampleMaintainer, terminalNginxApp)
	assert.Nil(t, err)
	assert.Equal(t, before, afterPreview)

	output = runClientCommandWithInput(t, "DELETE\n", "versions prune-incompatible")
	assert.True(t, strings.Contains(output, "Type DELETE to confirm"))
	assert.True(t, strings.Contains(output, "Deleted versions:"))
	assert.True(t, strings.Contains(output, "0.1.0"))
	assert.True(t, strings.Contains(output, "Pruning was successful."))
	remaining, err := setup.storeClient.ListVersions(tools.SampleMaintainer, terminalNginxApp)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(remaining))

	output = runClientCommand(t, "versions prune-incompatible -d")
	assert.Equal(t, "No incompatible versions found.\n", output)
}
