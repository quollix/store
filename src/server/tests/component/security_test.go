//go:build component

package check

import (
	"net/http"
	"testing"
	"time"

	"server/maintainers"
	"server/setup"
	"server/tools"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

func TestCookieExpirationTimeIsUpdatedAfterAuthenticatedRequest(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	accountDetails1, err := storeClient.GetAccountDetails()
	assert.Nil(t, err)
	accountDetails2, err := storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.True(t, accountDetails1.CookieExpirationDate.Before(accountDetails2.CookieExpirationDate))
}

func TestGetAccountDetailsUsedSpaceChanges(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	accountDetails, err := storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, sampleMaintainer, accountDetails.Name)
	assert.Equal(t, sampleEmail, accountDetails.Email)
	assert.Equal(t, u.GetOtherLocalTestingPublicKeyRaw(), accountDetails.PublicKeyRaw)
	assert.Equal(t, int64(0), accountDetails.UsedSpaceInBytes)
	assert.Equal(t, tools.UserStorageLimitInBytes, accountDetails.StorageLimitInBytes)
	assert.False(t, accountDetails.IsAdmin)

	assert.True(t, time.Now().UTC().AddDate(0, 0, maintainers.AuthCookieValidDays-1).Before(accountDetails.CookieExpirationDate))
	assert.True(t, time.Now().UTC().AddDate(0, 0, maintainers.AuthCookieValidDays+1).After(accountDetails.CookieExpirationDate))

	createdVersion, err := UploadSignedVersionAndReturnCreatedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	accountDetails, err = storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.True(t, len(SampleVersionFileContent) > 0)
	assert.Equal(t, int64(len(SampleVersionFileContent)), accountDetails.UsedSpaceInBytes)
	assert.Equal(t, tools.UserStorageLimitInBytes, accountDetails.StorageLimitInBytes)

	err = storeClient.DeleteVersionByID(createdVersion.VersionId)
	assert.Nil(t, err)
	accountDetails, err = storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), accountDetails.UsedSpaceInBytes)
	assert.Equal(t, tools.UserStorageLimitInBytes, accountDetails.StorageLimitInBytes)

	err = UploadSignedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	accountDetails, err = storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, int64(len(SampleVersionFileContent)), accountDetails.UsedSpaceInBytes)
	err = storeClient.DeleteApp(sampleApp)
	assert.Nil(t, err)
	accountDetails, err = storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), accountDetails.UsedSpaceInBytes)
	assert.Equal(t, tools.UserStorageLimitInBytes, accountDetails.StorageLimitInBytes)
}

func TestDeleteAppFreesUsedSpaceOfAllContainedVersions(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	version1 := SampleVersionFileContent
	version2 := append([]byte(nil), SampleVersionFileContent...)
	version2 = append(version2, []byte("\n# extra-bytes-to-change-size")...)

	err := UploadSignedVersion(storeClient, sampleApp, sampleVersion, version1)
	assert.Nil(t, err)
	err = UploadSignedVersion(storeClient, sampleApp, "0.0.2", version2)
	assert.Nil(t, err)

	accountDetails, err := storeClient.GetAccountDetails()
	assert.Nil(t, err)
	expectedUsedSpace := int64(len(version1) + len(version2))
	assert.Equal(t, expectedUsedSpace, accountDetails.UsedSpaceInBytes)

	err = storeClient.DeleteApp(sampleApp)
	assert.Nil(t, err)

	accountDetails, err = storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), accountDetails.UsedSpaceInBytes)
	assert.Equal(t, tools.UserStorageLimitInBytes, accountDetails.StorageLimitInBytes)
}

func TestFindAppsSecurity(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	storeClient.Parent.SetCookieHeader = false

	_, err := storeClient.SearchForApps("", "notexistingapp", true)
	assert.Nil(t, err)

	_, err = storeClient.SearchForApps("notexistinguser", "", true)
	assert.Nil(t, err)
}

func TestDownloadAppSecurity(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	createdVersion, err := UploadSignedVersionAndReturnCreatedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)
	storeClient.Parent.SetCookieHeader = false
	version, err := storeClient.DownloadVersionByID(createdVersion.VersionId)
	assert.Nil(t, err)
	assert.Equal(t, SampleVersionFileContent, version.Content)
}

func TestDeleteVersionByIDRejectsOtherMaintainer(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	createdVersion, err := UploadSignedVersionAndReturnCreatedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	otherClient := GetStoreClient()
	otherMaintainer := "othermaintainer"
	err = CreateMaintainerByAdminAndSetPassword(t, otherMaintainer, samplePassword, u.SampleEmailFailingRecipient, getOtherTestingPublicKey())
	assert.Nil(t, err)
	err = otherClient.Login(otherMaintainer, samplePassword)
	assert.Nil(t, err)

	err = otherClient.DeleteVersionByID(createdVersion.VersionId)
	u.AssertDeepStackErrorFromRequest(t, err, tools.DoesNotExistError)
}

func TestSetVersionMigrationCheckpointRejectsOtherMaintainer(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	createdVersion, err := UploadSignedVersionAndReturnCreatedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	otherClient := GetStoreClient()
	otherMaintainer := "othermaintainer"
	err = CreateMaintainerByAdminAndSetPassword(t, otherMaintainer, samplePassword, u.SampleEmailFailingRecipient, getOtherTestingPublicKey())
	assert.Nil(t, err)
	err = otherClient.Login(otherMaintainer, samplePassword)
	assert.Nil(t, err)

	err = otherClient.SetVersionMigrationCheckpoint(createdVersion.VersionId, true)
	u.AssertDeepStackErrorFromRequest(t, err, tools.DoesNotExistError)
}

func TestGetVersionsSecurity(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := UploadSignedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	storeClient.Parent.SetCookieHeader = false
	versions, err := storeClient.ListVersions(sampleMaintainer, sampleApp)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(versions))
	assert.Equal(t, sampleVersion, versions[0].Name)
}

func TestChangePasswordSecurity(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	newPassword := samplePassword + "x"
	correctlyFormattedButNotMatchingPassword := samplePassword + "xy"

	err := storeClient.ChangePassword(correctlyFormattedButNotMatchingPassword, newPassword)
	assert.NotNil(t, err)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.IncorrectUsernameOrPasswordError)
}

func TestLoginSetsSecureCookie(t *testing.T) {
	storeClient := GetStoreClient()
	defer storeClient.WipeData()
	err := CreateMaintainerByAdminAndSetPassword(t, sampleMaintainer, samplePassword, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw())
	assert.Nil(t, err)
	assert.Nil(t, storeClient.Parent.Cookie)
	assert.Nil(t, storeClient.Login(sampleMaintainer, samplePassword))
	assert.NotNil(t, storeClient.Parent.Cookie)
	checkCookie(t, storeClient)
}

func TestAuthenticatedRequestRefreshesSecureCookie(t *testing.T) {
	storeClient := GetStoreClient()
	defer storeClient.WipeData()
	err := CreateMaintainerByAdminAndSetPassword(t, sampleMaintainer, samplePassword, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw())
	assert.Nil(t, err)
	assert.Nil(t, storeClient.Login(sampleMaintainer, samplePassword))
	firstCookieExpires := storeClient.Parent.Cookie.Expires

	time.Sleep(1 * time.Second)
	err = storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)
	checkCookie(t, storeClient)
	assert.True(t, firstCookieExpires.Before(storeClient.Parent.Cookie.Expires))
}

func TestLoginSecurity(t *testing.T) {
	storeClient := GetStoreClient()
	defer storeClient.WipeData()
	err := CreateMaintainerByAdminAndSetPassword(t, sampleMaintainer, samplePassword, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw())
	assert.Nil(t, err)
	storeClient.Parent.Cookie = nil
	correctlyFormattedButNotMatchingPassword := samplePassword + "x"
	err = storeClient.Login(sampleMaintainer, correctlyFormattedButNotMatchingPassword)
	assert.NotNil(t, err)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.IncorrectUsernameOrPasswordError)
}

func checkCookie(t *testing.T, storeClient *store.AppStoreClientImpl) {
	assert.Equal(t, cookieName, storeClient.Parent.Cookie.Name)
	assert.Equal(t, "/", storeClient.Parent.Cookie.Path)
	assert.Equal(t, http.SameSiteStrictMode, storeClient.Parent.Cookie.SameSite)
	assert.True(t, storeClient.Parent.Cookie.HttpOnly)
	assert.True(t, storeClient.Parent.Cookie.Secure)
	assert.True(t, time.Now().UTC().AddDate(0, 0, maintainers.AuthCookieValidDays-1).Before(storeClient.Parent.Cookie.Expires))
	assert.True(t, time.Now().UTC().AddDate(0, 0, maintainers.AuthCookieValidDays+1).After(storeClient.Parent.Cookie.Expires))
}

func TestCookieAndHostProtection(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	storeClient.Parent.SetCookieHeader = false
	_, err := storeClient.ListOwnApps()
	u.AssertDeepStackErrorFromRequest(t, err, setup.CookieNotSetInRequest)

	storeClient.Parent.SetCookieHeader = true
	storeClient.Parent.Cookie.Value = "some-invalid-cookie-value"
	_, err = storeClient.ListOwnApps()
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.InvalidCookieError)

	validButNonExistentCookie := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	storeClient.Parent.Cookie.Value = validButNonExistentCookie
	_, err = storeClient.ListOwnApps()
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.CookieNotFoundError)

	assert.Nil(t, storeClient.Login(sampleMaintainer, samplePassword))
}

func TestCookie(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	assert.Equal(t, cookieName, storeClient.Parent.Cookie.Name)
	assert.True(t, time.Now().UTC().AddDate(0, 0, maintainers.AuthCookieValidDays).Add(10*time.Second).After(storeClient.Parent.Cookie.Expires))
	assert.True(t, time.Now().UTC().AddDate(0, 0, maintainers.AuthCookieValidDays).Add(-10*time.Second).Before(storeClient.Parent.Cookie.Expires))
	assert.Equal(t, 64, len(storeClient.Parent.Cookie.Value))

	cookie1 := storeClient.Parent.Cookie
	err := storeClient.Login(sampleMaintainer, samplePassword)
	assert.Nil(t, err)
	cookie2 := storeClient.Parent.Cookie
	assert.NotNil(t, cookie2)
	assert.NotEqual(t, cookie1.Value, cookie2.Value)
}
