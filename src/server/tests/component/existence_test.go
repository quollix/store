//go:build component

package check

import (
	"testing"
	"time"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestDownloadVersionByIDCanThrowDoesNotExistError(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	_, err := storeClient.DownloadVersionByID(1)
	u.AssertDeepStackErrorFromRequest(t, err, doesNotExistError)
	err = storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)

	_, err = storeClient.DownloadVersionByID(1)
	u.AssertDeepStackErrorFromRequest(t, err, doesNotExistError)
}

func TestDeletingVersionCanThrowDoesNotExistError(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := storeClient.DeleteVersionByID(1)
	u.AssertDeepStackErrorFromRequest(t, err, doesNotExistError)
	err = storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)
	err = storeClient.DeleteVersionByID(1)
	u.AssertDeepStackErrorFromRequest(t, err, doesNotExistError)
	createdVersion, err := UploadSignedVersionAndReturnCreatedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)
	err = storeClient.DeleteVersionByID(createdVersion.VersionId)
	assert.Nil(t, err)
}
func TestDeletingAppCanThrowDoesNotExistError(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := storeClient.DeleteApp(sampleApp)
	u.AssertDeepStackErrorFromRequest(t, err, doesNotExistError)
}

func TestGetVersionsCanThrowDoesNotExistError(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	_, err := storeClient.ListVersions(sampleMaintainer, sampleApp)
	u.AssertDeepStackErrorFromRequest(t, err, doesNotExistError)
	err = storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)
	versionList, err := storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(versionList))
}

func TestDownloadVersionReturnsStoredSignature(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	assert.Nil(t, storeClient.CreateApp(sampleApp))
	expectedTime := time.Now().Add(-5 * time.Minute)
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, SampleVersionFileContent, expectedTime)
	assert.Nil(t, err)
	createdVersion, err := storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, SampleVersionFileContent, signature)
	assert.Nil(t, err)

	version, err := storeClient.DownloadVersionByID(createdVersion.VersionId)
	assert.Nil(t, err)
	assert.Equal(t, signature, version.Signature)
	assertTime(t, expectedTime, version.VersionCreationTimestamp)
}

func assertTime(t *testing.T, expected, actual time.Time) {
	assert.True(t, expected.Before(actual.Add(1*time.Minute)))
	assert.True(t, expected.After(actual.Add(-1*time.Minute)))
}
