//go:build component

package check

import (
	"strings"
	"testing"
	"time"

	"server/versions"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

func TestVersionUploadAndDownload(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, SampleVersionFileContent, time.Now().UTC().Round(time.Microsecond))
	assert.Nil(t, err)
	createdVersion, err := storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, SampleVersionFileContent, signature)
	assert.Nil(t, err)
	assert.True(t, createdVersion.AppId > 0)
	assert.True(t, createdVersion.VersionId > 0)
	version, err := storeClient.DownloadVersionByID(createdVersion.VersionId)
	assert.Nil(t, err)
	assert.Equal(t, createdVersion.VersionId, version.VersionId)
	assert.Equal(t, SampleVersionFileContent, version.Content)
	assert.Equal(t, signature, version.Signature)
	assert.Equal(t, u.GetOtherLocalTestingPublicKeyRaw(), version.MaintainerPublicKeyRaw)
	assert.Equal(t, sampleMaintainer, version.Maintainer)
	assert.Equal(t, sampleApp, version.AppName)
	assert.Equal(t, sampleVersion, version.VersionName)
	assert.True(t, time.Now().UTC().Add(-10*time.Second).Before(version.VersionCreationTimestamp))
	assert.True(t, time.Now().UTC().Add(10*time.Second).After(version.VersionCreationTimestamp))
}

func TestVersionUploadAllowsComposeImageDigest(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	content := sampleVersionContentWithImage(t, "sample/sampleapp:1.2.3@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, content, time.Now().UTC().Round(time.Microsecond))
	assert.Nil(t, err)

	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, content, signature)
	assert.Nil(t, err)
}

func TestVersionUploadRejectsCarriageReturn(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	body := strings.TrimPrefix(string(SampleVersionFileContent), validation.AppDefinitionLicenseNotice)
	content := []byte(validation.AppDefinitionLicenseNotice + strings.ReplaceAll(body, "\n", "\r\n"))
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, content, time.Now().UTC().Round(time.Microsecond))
	assert.Nil(t, err)

	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, content, signature)
	u.AssertDeepStackErrorFromRequest(t, err, versions.VersionContentContainsCarriageReturn)
}

func TestVersionUploadRejectsInvalidComposeImageDigest(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	content := sampleVersionContentWithImage(t, "sample/sampleapp:1.2.3@sha256:not-a-real-digest")
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, content, time.Now().UTC().Round(time.Microsecond))
	assert.Nil(t, err)

	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, content, signature)
	u.AssertDeepStackErrorFromRequest(t, err, "the image digest must be a valid OCI digest")
}

func TestUploadedVersionContentAlreadyExists(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := UploadSignedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, "1.0.1", SampleVersionFileContent, time.Now().Add(-4*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, "1.0.1", creationTimestamp, SampleVersionFileContent, signature)
	u.AssertDeepStackErrorFromRequest(t, err, store.VersionContentAlreadyExists)
}

func TestDuplicateVersionNamesAreAllowed(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	olderContent := versionContentWithComment("older")
	olderTimestamp, olderSignature, err := NewSignedVersionUpload(sampleApp, sampleVersion, olderContent, time.Now().Add(-5*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, olderTimestamp, olderContent, olderSignature)
	assert.Nil(t, err)

	newerContent := versionContentWithComment("newer")
	newerTimestamp, newerSignature, err := NewSignedVersionUpload(sampleApp, sampleVersion, newerContent, time.Now().Add(-4*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, newerTimestamp, newerContent, newerSignature)
	assert.Nil(t, err)

	versionList, err := storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(versionList))
	assert.Equal(t, sampleVersion, versionList[0].Name)
	assert.Equal(t, sampleVersion, versionList[1].Name)
	assert.True(t, versionList[0].VersionId != versionList[1].VersionId)
	assert.Equal(t, newerTimestamp, versionList[0].CreationTimestamp)
}

func TestUploadVersionRejectsTimestampOlderThanLatestAppVersion(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	newerContent := versionContentWithComment("newer")
	newerTimestamp, newerSignature, err := NewSignedVersionUpload(sampleApp, sampleVersion, newerContent, time.Now().Add(-4*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, newerTimestamp, newerContent, newerSignature)
	assert.Nil(t, err)

	olderContent := versionContentWithComment("older")
	olderTimestamp, olderSignature, err := NewSignedVersionUpload(sampleApp, "1.0.1", olderContent, time.Now().Add(-5*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, "1.0.1", olderTimestamp, olderContent, olderSignature)
	u.AssertDeepStackErrorFromRequest(t, err, versions.VersionUploadTimestampNotNewer)
}

func TestDownloadVersionByID(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	content := versionContentWithComment("by-id")
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, content, time.Now().Add(-5*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	createdVersion, err := storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, content, signature)
	assert.Nil(t, err)

	versionList, err := storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), versionList[0].DownloadCount)

	version, err := storeClient.DownloadVersionByID(createdVersion.VersionId)
	assert.Nil(t, err)
	assert.Equal(t, createdVersion.VersionId, version.VersionId)
	assert.Equal(t, content, version.Content)

	versionList, err = storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), versionList[0].DownloadCount)
}

func TestDownloadNextVersionForUpdateUsesMigrationCheckpointPath(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	v1Content := versionContentWithComment("v1")
	v1Timestamp, v1Signature, err := NewSignedVersionUpload(sampleApp, "1.0.0", v1Content, time.Now().Add(-8*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, "1.0.0", v1Timestamp, v1Content, v1Signature)
	assert.Nil(t, err)

	v2Content := versionContentWithComment("v2")
	v2Timestamp, v2Signature, err := NewSignedVersionUpload(sampleApp, "2.0.0", v2Content, time.Now().Add(-7*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	v2, err := storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, "2.0.0", v2Timestamp, v2Content, v2Signature)
	assert.Nil(t, err)
	assert.Nil(t, storeClient.SetVersionMigrationCheckpoint(v2.VersionId, true))

	v3Content := versionContentWithComment("v3")
	v3Timestamp, v3Signature, err := NewSignedVersionUpload(sampleApp, "3.0.0", v3Content, time.Now().Add(-6*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, "3.0.0", v3Timestamp, v3Content, v3Signature)
	assert.Nil(t, err)

	v4Content := versionContentWithComment("v4")
	v4Timestamp, v4Signature, err := NewSignedVersionUpload(sampleApp, "4.0.0", v4Content, time.Now().Add(-5*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	v4, err := storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, "4.0.0", v4Timestamp, v4Content, v4Signature)
	assert.Nil(t, err)

	response, err := storeClient.DownloadNextVersionForUpdate(sampleMaintainer, sampleApp, v1Timestamp)
	assert.Nil(t, err)
	assert.True(t, response.UpdateAvailable)
	assert.Equal(t, v2.VersionId, response.Version.VersionId)
	assert.Equal(t, v2Content, response.Version.Content)
	assert.Equal(t, v2Signature, response.Version.Signature)
	assert.Equal(t, u.GetOtherLocalTestingPublicKeyRaw(), response.Version.MaintainerPublicKeyRaw)
	assert.Equal(t, sampleMaintainer, response.Version.Maintainer)
	assert.Equal(t, sampleApp, response.Version.AppName)
	assert.Equal(t, "2.0.0", response.Version.VersionName)
	assert.Equal(t, v2Timestamp, response.Version.VersionCreationTimestamp)
	assert.True(t, response.Version.IsMigrationCheckpoint)

	response, err = storeClient.DownloadNextVersionForUpdate(sampleMaintainer, sampleApp, v2Timestamp)
	assert.Nil(t, err)
	assert.True(t, response.UpdateAvailable)
	assert.Equal(t, v4.VersionId, response.Version.VersionId)
	assert.Equal(t, v4Content, response.Version.Content)
	assert.Equal(t, "4.0.0", response.Version.VersionName)
	assert.Equal(t, v4Timestamp, response.Version.VersionCreationTimestamp)
	assert.False(t, response.Version.IsMigrationCheckpoint)

	response, err = storeClient.DownloadNextVersionForUpdate(sampleMaintainer, sampleApp, v4Timestamp)
	assert.Nil(t, err)
	assert.False(t, response.UpdateAvailable)
	assert.Nil(t, response.Version)

	versionList, err := storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), versionList[0].DownloadCount)
	assert.Equal(t, int64(0), versionList[1].DownloadCount)
	assert.Equal(t, int64(1), versionList[2].DownloadCount)
	assert.Equal(t, int64(0), versionList[3].DownloadCount)
}

func TestVersionMigrationCheckpointCanBeSetAndUnset(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	createdVersion, err := UploadSignedVersionAndReturnCreatedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	versionList, err := storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(versionList))
	assert.False(t, versionList[0].IsMigrationCheckpoint)
	assert.Equal(t, int64(0), versionList[0].DownloadCount)

	assert.Nil(t, storeClient.SetVersionMigrationCheckpoint(createdVersion.VersionId, true))
	versionList, err = storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.True(t, versionList[0].IsMigrationCheckpoint)

	assert.Nil(t, storeClient.SetVersionMigrationCheckpoint(createdVersion.VersionId, false))
	versionList, err = storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.False(t, versionList[0].IsMigrationCheckpoint)
}

func TestGetVersions(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)

	versionList, err := storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(versionList))

	err = UploadSignedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	versionList, err = storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(versionList))
	downloadedVersion := versionList[0]
	assert.Equal(t, sampleVersion, downloadedVersion.Name)
	assert.True(t, downloadedVersion.VersionId > 0)
	assert.True(t, time.Now().UTC().Add(-10*time.Minute).Before(downloadedVersion.CreationTimestamp))
	assert.True(t, time.Now().UTC().Add(1*time.Minute).After(downloadedVersion.CreationTimestamp))
	assert.Equal(t, int64(len(SampleVersionFileContent)), downloadedVersion.SizeInBytes)
}

func TestInvalidComposeUpload(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	invalidCompose := []byte(validation.AppDefinitionLicenseNotice + `
services:
  db:
    image: postgres:16
    container_name: sample_sampleapp_db
`)
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, invalidCompose, time.Now().UTC())
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, invalidCompose, signature)
	u.AssertDeepStackErrorFromRequest(t, err, "the main service was not defined")
}

func TestVersionUploadRejectsMissingLicenseNotice(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	content := strings.TrimPrefix(string(SampleVersionFileContent), validation.AppDefinitionLicenseNotice+"\n")
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, []byte(content), time.Now().UTC().Round(time.Microsecond))
	assert.Nil(t, err)

	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, []byte(content), signature)
	u.AssertDeepStackErrorFromRequest(t, err, validation.MissingAppDefinitionLicenseNotice)
}

func versionContentWithComment(comment string) []byte {
	content := append([]byte(nil), SampleVersionFileContent...)
	content = append(content, []byte("\n# "+comment)...)
	return content
}

func sampleVersionContentWithImage(t *testing.T, image string) []byte {
	content := string(SampleVersionFileContent)
	if !strings.Contains(content, "image: sample/sampleapp:1.2.3") {
		t.Fatal("sample version image not found")
	}
	return []byte(strings.Replace(content, "image: sample/sampleapp:1.2.3", "image: "+image, 1))
}
