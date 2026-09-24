//go:build component

package check

import (
	"crypto/ed25519"
	"strings"
	"testing"
	"time"

	"server/maintainers"
	serversetup "server/setup"
	"server/tools"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

func TestAdminCanCreateMaintainerAndMaintainerCanSetInitialPassword(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()

	publicKey := u.GetOtherLocalTestingPublicKeyRaw()
	err := createMaintainerByAdmin(t, adminClient, sampleMaintainer, sampleEmail, publicKey)
	assert.Nil(t, err)

	maintainerList, err := adminClient.ListMaintainersByAdmin()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(maintainerList))
	assert.Equal(t, maintainers.QuollixAdminUsername, maintainerList[0].Name)
	assert.Equal(t, maintainers.QuollixAdminEmail, maintainerList[0].Email)
	assert.True(t, maintainerList[0].IsActive)
	assert.Equal(t, sampleMaintainer, maintainerList[1].Name)
	assert.Equal(t, sampleEmail, maintainerList[1].Email)
	assert.Equal(t, publicKey, maintainerList[1].PublicKeyRaw)
	assert.False(t, maintainerList[1].IsActive)

	userClient := GetStoreClient()
	err = userClient.Login(sampleMaintainer, samplePassword)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.IncorrectUsernameOrPasswordError)

	assert.Nil(t, setupInitialPassword(userClient, maintainers.DefaultAccountRegistrationCode, samplePassword))
	maintainerList, err = adminClient.ListMaintainersByAdmin()
	assert.Nil(t, err)
	assert.True(t, maintainerList[1].IsActive)

	assert.Nil(t, userClient.Login(sampleMaintainer, samplePassword))
	accountDetails, err := userClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, sampleMaintainer, accountDetails.Name)
	assert.Equal(t, sampleEmail, accountDetails.Email)
	assert.Equal(t, publicKey, accountDetails.PublicKeyRaw)
	assert.Equal(t, tools.UserStorageLimitInBytes, accountDetails.StorageLimitInBytes)
	assert.False(t, accountDetails.IsAdmin)
}

func TestAdminCanSetMaintainerStorageLimitAndNegativeLimitIsRejected(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()

	storageLimit := 20 * tools.OneMegaByteInBytes
	assert.Nil(t, adminClient.SetMaintainerStorageLimitByAdmin(maintainers.QuollixAdminUsername, storageLimit))
	accountDetails, err := adminClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, storageLimit, accountDetails.StorageLimitInBytes)

	err = adminClient.SetMaintainerStorageLimitByAdmin(maintainers.QuollixAdminUsername, -1)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.StorageLimitMustBeNonNegativeError)
	accountDetails, err = adminClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, storageLimit, accountDetails.StorageLimitInBytes)
}

func TestAdminMaintainerCreationRejectsExistingValues(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()
	assert.Nil(t, createAndActivateMaintainer(t, adminClient, sampleMaintainer, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw()))

	err := createMaintainerByAdmin(t, adminClient, sampleMaintainer, u.SampleEmailFailingRecipient, getOtherTestingPublicKey())
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.MaintainerAlreadyExistsError)

	err = createMaintainerByAdmin(t, adminClient, "other", sampleEmail, getOtherTestingPublicKey())
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.EmailAlreadyExistsError)

	err = createMaintainerByAdmin(t, adminClient, "other", u.SampleEmailFailingRecipient, u.GetOtherLocalTestingPublicKeyRaw())
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.PublicKeyAlreadyExistsError)
}

func TestAdminMaintainerCreationRejectsInvalidPublicKey(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()

	err := adminClient.CreateMaintainerByAdmin(sampleMaintainer, sampleEmail, []byte{1, 2, 3}, make([]byte, ed25519.SignatureSize))
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.InvalidPublicKeyError)
}

func TestAdminMaintainerCreationRejectsInvalidPublicKeySignature(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()

	err := adminClient.CreateMaintainerByAdmin(sampleMaintainer, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw(), []byte("short"))
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.InvalidPublicKeySignatureError)
}

func TestAdminMaintainerCreationRollsBackUserWhenSetupEmailFails(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()
	maintainerName := sampleMaintainer
	publicKey := u.GetOtherLocalTestingPublicKeyRaw()
	err := adminClient.CreateMaintainerByAdmin(maintainerName, sampleEmail, publicKey, signMaintainerPublicKey(t, maintainerName, publicKey))
	assert.NotNil(t, err)

	_, err = GetStoreClient().GetMaintainerPublicKeyRecord(maintainerName)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.MaintainerNotFoundError)
}

func TestMaintainerPublicKeyEndpointReturnsSignedMaintainerKey(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()

	publicKey := u.GetOtherLocalTestingPublicKeyRaw()
	assert.Nil(t, createAndActivateMaintainer(t, adminClient, sampleMaintainer, sampleEmail, publicKey))

	record, err := GetStoreClient().GetMaintainerPublicKeyRecord(sampleMaintainer)
	assert.Nil(t, err)
	assert.Equal(t, sampleMaintainer, record.Maintainer)
	assert.Equal(t, publicKey, record.PublicKeyRaw)
	ok, err := store.VerifyMaintainerPublicKeySignature(ed25519.PublicKey(getQuollixAdminPublicKey()), record)
	assert.Nil(t, err)
	assert.True(t, ok)

	_, err = GetStoreClient().GetMaintainerPublicKeyRecord(maintainers.QuollixAdminUsername)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.MaintainerNotFoundError)
}

func TestAdminCanDeleteMaintainerAndRemoveAppsFromSearch(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()
	assert.Nil(t, createAndActivateMaintainer(t, adminClient, sampleMaintainer, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw()))
	userClient := GetStoreClient()
	assert.Nil(t, userClient.Login(sampleMaintainer, samplePassword))
	assert.Nil(t, userClient.CreateApp(sampleApp))
	content := versionContentForMaintainerApp(sampleMaintainer, sampleApp, SampleVersionFileContent)
	assert.Nil(t, uploadSignedVersionForMaintainer(userClient, sampleMaintainer, sampleApp, sampleVersion, content))

	apps, err := GetStoreClient().SearchForApps(sampleMaintainer, sampleApp, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))

	assert.Nil(t, adminClient.DeleteMaintainerByAdmin(sampleMaintainer))

	apps, err = GetStoreClient().SearchForApps(sampleMaintainer, sampleApp, true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
	err = userClient.Login(sampleMaintainer, samplePassword)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.IncorrectUsernameOrPasswordError)
}

func TestAdminCanDeleteMaintainer(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()
	assert.Nil(t, createAndActivateMaintainer(t, adminClient, sampleMaintainer, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw()))

	assert.Nil(t, deleteMaintainerByAdmin(adminClient, sampleMaintainer))

	userClient := GetStoreClient()
	err := userClient.Login(sampleMaintainer, samplePassword)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.IncorrectUsernameOrPasswordError)
}

func TestAdminCanDeleteMaintainerBeforePasswordSetup(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()

	err := createMaintainerByAdmin(t, adminClient, sampleMaintainer, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw())
	assert.Nil(t, err)
	assert.Nil(t, adminClient.DeleteMaintainerByAdmin(sampleMaintainer))

	err = createMaintainerByAdmin(t, adminClient, sampleMaintainer, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw())
	assert.Nil(t, err)
}

func TestAdminMaintainerDeletionRejectsNotFoundAndAdmin(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()

	err := deleteMaintainerByAdmin(adminClient, "missing")
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.MaintainerNotFoundError)

	err = deleteMaintainerByAdmin(adminClient, maintainers.QuollixAdminUsername)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.AdminMaintainerDeleteError)
}

func TestAppMaintainerCanNotUseAdminMaintainerEndpoints(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()
	assert.Nil(t, createAndActivateMaintainer(t, adminClient, sampleMaintainer, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw()))

	userClient := GetStoreClient()
	assert.Nil(t, userClient.Login(sampleMaintainer, samplePassword))
	err := userClient.CreateMaintainerByAdmin("other", u.SampleEmailFailingRecipient, getOtherTestingPublicKey(), signMaintainerPublicKey(t, "other", getOtherTestingPublicKey()))
	u.AssertDeepStackErrorFromRequest(t, err, serversetup.AccessDeniedForNonAdminUserError)
	_, err = userClient.ListMaintainersByAdmin()
	u.AssertDeepStackErrorFromRequest(t, err, serversetup.AccessDeniedForNonAdminUserError)
}

func TestSetupInitialPasswordRejectsInvalidToken(t *testing.T) {
	client := GetStoreClient()
	err := setupInitialPassword(client, strings.Repeat("a", 64), samplePassword)
	u.AssertDeepStackErrorFromRequest(t, err, maintainers.RegistrationCodeNotFoundError)
}

func createAndActivateMaintainer(t *testing.T, adminClient *store.AppStoreClientImpl, name string, email string, publicKeyRaw []byte) error {
	err := createMaintainerByAdmin(t, adminClient, name, email, publicKeyRaw)
	if err != nil {
		return err
	}
	return setupInitialPassword(GetStoreClient(), maintainers.DefaultAccountRegistrationCode, samplePassword)
}

func setupInitialPassword(client *store.AppStoreClientImpl, setupToken string, password string) error {
	return client.SetupInitialPassword(setupToken, password)
}

func uploadSignedVersionForMaintainer(client *store.AppStoreClientImpl, maintainerName string, appName string, versionName string, content []byte) error {
	creationTimestamp := time.Now().Add(-5 * time.Minute).UTC()
	privateKey, err := u.DecodeEd25519PrivateKeyOpenSSH([]byte(u.OtherLocalTestingPrivateKeyOpenSSH), []byte(u.OtherLocalTestingPrivateKeyPassphrase))
	if err != nil {
		return err
	}
	signature, err := versionSigningService.SignVersion(privateKey, &store.Version{
		Maintainer:               maintainerName,
		AppName:                  appName,
		VersionName:              versionName,
		Content:                  content,
		VersionCreationTimestamp: creationTimestamp,
	})
	if err != nil {
		return err
	}
	_, err = client.UploadVersionAndReturnCreatedVersion(appName, versionName, creationTimestamp, content, signature)
	return err
}

func versionContentForMaintainerApp(maintainerName string, appName string, content []byte) []byte {
	return []byte(strings.ReplaceAll(string(content), sampleMaintainer+"_"+sampleApp, maintainerName+"_"+appName))
}

func deleteMaintainerByAdmin(client *store.AppStoreClientImpl, name string) error {
	return client.DeleteMaintainerByAdmin(name)
}
