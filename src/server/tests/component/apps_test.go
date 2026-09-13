//go:build component

package check

import (
	"testing"
	"time"

	"server/apps"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

func TestAppAlreadyExistsError(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)
	err = storeClient.CreateApp(sampleApp)
	u.AssertDeepStackErrorFromRequest(t, err, apps.AppAlreadyExistsError)
}

func TestCreateAndDeleteApp(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)
	foundApps, err := storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(foundApps))
	foundApp := foundApps[0]
	assert.Equal(t, sampleApp, foundApp)
	assert.Nil(t, storeClient.DeleteApp(sampleApp))
	foundApps, err = storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(foundApps))
}

func TestGetAppList(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	apps, err := storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
	err = storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)
	apps, err = storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, sampleApp, apps[0])
}

func TestCreationOfBrandAppIsForbidden(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := storeClient.CreateApp(u.OfficialBrandAppName)
	assert.NotNil(t, err)
	u.AssertDeepStackErrorFromRequest(t, err, apps.AppNameReservedError)
}

func TestUploadingVersionAutomaticallyCreatesAppIfNotExistent(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	apps, err := storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
	err = UploadSignedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)
	apps, err = storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, sampleApp, apps[0])
}

func TestSearchingOptions(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := UploadSignedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	apps, err := storeClient.SearchForApps("", sampleApp, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))

	apps, err = storeClient.SearchForApps(sampleMaintainer, "", true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))

	apps, err = storeClient.SearchForApps("", sampleApp, false)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))

	apps, err = storeClient.SearchForApps(sampleMaintainer, "", false)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
}

func TestAllowEmptyStringAsSearchTerm(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	apps, err := storeClient.SearchForApps("", "", true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))

	err = UploadSignedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	apps, err = storeClient.SearchForApps("", "", true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))
}

func TestTolerateTwoDifferentUsersCreateAppWithSameName(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)

	storeClient2 := GetStoreClient()
	err = CreateMaintainerByAdminAndSetPassword(t, sampleMaintainer+"2", samplePassword, u.SampleEmailFailingRecipient, getOtherTestingPublicKey())
	assert.Nil(t, err)
	err = storeClient2.Login(sampleMaintainer+"2", samplePassword)
	assert.Nil(t, err)
	err = storeClient2.CreateApp(sampleApp)
	assert.Nil(t, err)
}

func TestSearchForNonExistingAppsReturnsEmptyList(t *testing.T) {
	storeClient := GetStoreClient()
	apps, err := storeClient.SearchForApps(sampleMaintainer, sampleApp, true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
}

func TestCantCreateAppTwiceForSameUser(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	err := storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)
	err = storeClient.CreateApp(sampleApp)
	u.AssertDeepStackErrorFromRequest(t, err, apps.AppAlreadyExistsError)
}

func TestAppSearchReturnsTheLatestAppVersion(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, SampleVersionFileContent, time.Now().Add(-10*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, SampleVersionFileContent, signature)
	assert.Nil(t, err)
	version2 := "v0.0.2"
	version2Content := append([]byte(nil), SampleVersionFileContent...)
	version2Content = append(version2Content, []byte("\n# version2")...)
	expectedLatestTimestamp, signature, err := NewSignedVersionUpload(sampleApp, version2, version2Content, time.Now().Add(-5*time.Minute).Round(time.Microsecond))
	assert.Nil(t, err)
	latestVersion, err := storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, version2, expectedLatestTimestamp, version2Content, signature)
	assert.Nil(t, err)

	searchedApps, err := storeClient.SearchForApps("", sampleApp, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(searchedApps))
	assert.Equal(t, latestVersion.VersionId, searchedApps[0].LatestVersionId)
	assert.Equal(t, version2, searchedApps[0].LatestVersionName)
	assert.Equal(t, expectedLatestTimestamp.UTC(), searchedApps[0].LatestVersionCreationTimestamp)
}

func TestUploadVersionHappyPath(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	assert.Nil(t, storeClient.CreateApp(sampleApp))
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, SampleVersionFileContent, time.Now().Add(-5*time.Minute))
	assert.Nil(t, err)
	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, SampleVersionFileContent, signature)
	assert.Nil(t, err)
}

func TestUploadVersionRejectsTooOldTimestamp(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	assert.Nil(t, storeClient.CreateApp(sampleApp))
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, SampleVersionFileContent, time.Now().Add(-2*time.Hour))
	assert.Nil(t, err)

	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, SampleVersionFileContent, signature)
	u.AssertDeepStackErrorFromRequest(t, err, "version upload timestamp is too old")
}

func TestUploadVersionRejectsInvalidSignature(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	assert.Nil(t, storeClient.CreateApp(sampleApp))
	creationTimestamp, signature, err := NewSignedVersionUpload(sampleApp, sampleVersion, SampleVersionFileContent, time.Now().Add(-5*time.Minute))
	assert.Nil(t, err)
	signature[0] ^= 0xFF

	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, SampleVersionFileContent, signature)
	u.AssertDeepStackErrorFromRequest(t, err, "invalid version signature")
}

func TestUploadVersionRejectsSignatureFromAnotherPrivateKey(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	assert.Nil(t, storeClient.CreateApp(sampleApp))
	otherPrivateKey, err := u.DecodeEd25519PrivateKeyOpenSSH([]byte(u.LocalTestingPrivateKeyOpenSSH), []byte(u.LocalTestingPrivateKeyPassphrase))
	assert.Nil(t, err)
	creationTimestamp := time.Now().Add(-5 * time.Minute).UTC()
	signature, err := versionSigningService.SignVersion(otherPrivateKey, &store.Version{
		Maintainer:               sampleMaintainer,
		AppName:                  sampleApp,
		VersionName:              sampleVersion,
		Content:                  SampleVersionFileContent,
		VersionCreationTimestamp: creationTimestamp,
	})
	assert.Nil(t, err)

	_, err = storeClient.UploadVersionAndReturnCreatedVersion(sampleApp, sampleVersion, creationTimestamp, SampleVersionFileContent, signature)
	u.AssertDeepStackErrorFromRequest(t, err, "invalid version signature")
}
