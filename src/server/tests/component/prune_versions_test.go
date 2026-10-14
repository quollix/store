//go:build component

package check

import (
	"slices"
	"strings"
	"testing"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
	"github.com/quollix/common/validation"
)

func TestPruneIncompatibleVersions_PreviewAndDelete(t *testing.T) {
	client := GetStoreClientAndLogin(t)
	defer client.WipeData()

	healthy, err := UploadSignedVersionAndReturnCreatedVersion(client, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)
	legacy, err := client.AddIncompatibleVersion(sampleApp, sampleVersion)
	assert.Nil(t, err)

	before, err := client.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	slices.SortFunc(before, func(a, b store.LeanVersionDto) int {
		return a.CreationTimestamp.Compare(b.CreationTimestamp)
	})
	assert.Equal(t, 2, len(before))
	assert.Equal(t, healthy.VersionId, before[0].VersionId)
	assert.Equal(t, legacy.VersionId, before[1].VersionId)

	preview, err := client.PruneIncompatibleVersions(true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(preview))
	assert.Equal(t, sampleApp, preview[0].AppName)
	assert.Equal(t, before[1], preview[0].Version)
	assert.True(t, strings.Contains(preview[0].Reason, validation.MissingAppDefinitionLicenseNotice))

	afterPreview, err := client.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	slices.SortFunc(afterPreview, func(a, b store.LeanVersionDto) int {
		return a.CreationTimestamp.Compare(b.CreationTimestamp)
	})
	assert.Equal(t, before, afterPreview)

	deleted, err := client.PruneIncompatibleVersions(false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(deleted))
	assert.Equal(t, sampleApp, deleted[0].AppName)
	assert.Equal(t, before[1], deleted[0].Version)
	assert.True(t, strings.Contains(deleted[0].Reason, validation.MissingAppDefinitionLicenseNotice))

	remaining, err := client.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(remaining))
	assert.Equal(t, before[0], remaining[0])
}
