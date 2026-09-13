//go:build component

package check

import (
	"testing"

	"server/maintainers"
	"server/tools"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestLogin(t *testing.T) {
	storeClient := GetStoreClient()
	err := storeClient.Login(sampleMaintainer, samplePassword)
	assert.NotNil(t, err)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.IncorrectUsernameOrPasswordError)
}

func TestChangePassword(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	newPassword := samplePassword + "x"

	assert.Nil(t, storeClient.ChangePassword(samplePassword, newPassword))
	err := storeClient.Login(sampleMaintainer, samplePassword)
	assert.NotNil(t, err)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.IncorrectUsernameOrPasswordError)

	storeClient.Parent.Cookie = nil
	err = storeClient.Login(sampleMaintainer, newPassword)
	assert.Nil(t, err)
	assert.NotNil(t, storeClient.Parent.Cookie)
}

func TestLogout(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	err := storeClient.CreateApp(sampleApp)
	assert.Nil(t, err)
	assert.Nil(t, storeClient.Logout())
	err = storeClient.CreateApp(sampleApp)
	assert.NotNil(t, err)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.CookieNotFoundError)
}

func TestTolerateTwoUsersWithSamePassword(t *testing.T) {
	client1 := GetStoreClientAndLogin(t)
	defer client1.WipeData()

	client2 := GetStoreClient()
	err := CreateMaintainerByAdminAndSetPassword(t, "user2", "password2", u.SampleEmailFailingRecipient, getOtherTestingPublicKey())
	assert.Nil(t, err)
	assert.Nil(t, client2.Login("user2", "password2"))
}

func TestGetAccountDetails(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	account, err := storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, sampleMaintainer, account.Name)
	assert.Equal(t, sampleEmail, account.Email)
	assert.Equal(t, u.GetOtherLocalTestingPublicKeyRaw(), account.PublicKeyRaw)
	assert.Equal(t, tools.UserStorageLimitInBytes, account.StorageLimitInBytes)
	assert.False(t, account.IsAdmin)

	adminClient := GetStoreClient()
	assert.Nil(t, adminClient.Login(maintainers.QuollixAdminUsername, maintainers.QuollixAdminPassword))
	adminAccount, err := adminClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, maintainers.QuollixAdminUsername, adminAccount.Name)
	assert.Equal(t, maintainers.QuollixAdminEmail, adminAccount.Email)
	assert.Equal(t, tools.AdminStorageLimitInBytes, adminAccount.StorageLimitInBytes)
	assert.Equal(t, getQuollixAdminPublicKey(), adminAccount.PublicKeyRaw)
	assert.True(t, adminAccount.IsAdmin)
}

func TestEmailChange(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	account, err := storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, sampleEmail, account.Email)

	newEmail := u.SampleEmailFailingRecipient
	assert.Nil(t, storeClient.ChangeEmail(newEmail))

	account, err = storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, newEmail, account.Email)
}

func TestEmailChangeEmailAlreadyExists(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	err := storeClient.ChangeEmail(sampleEmail)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.EmailAlreadyExistsError)
}

func TestEmailChangeRejectsEmailOfAnotherExistingUser(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	otherEmail := u.SampleEmailFailingRecipient
	err := CreateMaintainerByAdminAndSetPassword(t, sampleMaintainer+"2", samplePassword, otherEmail, getOtherTestingPublicKey())
	assert.Nil(t, err)

	err = storeClient.ChangeEmail(otherEmail)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.EmailAlreadyExistsError)
}

func TestDeleteOwnAccountDeletesAppsAndAllowsNameReuse(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()

	err := UploadSignedVersion(storeClient, sampleApp, sampleVersion, SampleVersionFileContent)
	assert.Nil(t, err)

	searchedApps, err := storeClient.SearchForApps(sampleMaintainer, sampleApp, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(searchedApps))

	err = storeClient.DeleteOwnAccount()
	assert.Nil(t, err)

	publicClient := GetStoreClient()
	searchedApps, err = publicClient.SearchForApps(sampleMaintainer, sampleApp, true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(searchedApps))

	err = CreateMaintainerByAdminAndSetPassword(t, sampleMaintainer, samplePassword, sampleEmail, getOtherTestingPublicKey())
	assert.Nil(t, err)
}
