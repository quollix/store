//go:build integration

package repository

import (
	"crypto/ed25519"
	"testing"
	"time"

	"server/maintainers"
	"server/tools"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestUserRepository_CreateReadAndUpdateStorageLimit(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("normal-user", "hashed-password", "normal@example.com", []byte{1, 1, 1}, tools.UserStorageLimitInBytes, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("normal-user")
	assert.Nil(t, err)
	assert.Equal(t, "normal-user", user.Name)
	assert.Equal(t, "normal@example.com", user.Email)
	assert.Equal(t, int64(0), user.UsedSpaceInBytes)
	assert.Equal(t, tools.UserStorageLimitInBytes, user.StorageLimitInBytes)
	assert.False(t, user.IsAdmin)

	user.UsedSpaceInBytes = 1234
	user.StorageLimitInBytes = 5678
	assert.Nil(t, deps.UserRepo.UpdateUser(user))

	updatedUser, err := deps.UserRepo.GetUserByName("normal-user")
	assert.Nil(t, err)
	assert.Equal(t, int64(1234), updatedUser.UsedSpaceInBytes)
	assert.Equal(t, int64(5678), updatedUser.StorageLimitInBytes)
}

func TestUserRepository_AdminGetsHigherStorageLimit(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("admin-user", "hashed-password", "admin@example.com", []byte{2, 2, 2}, tools.AdminStorageLimitInBytes, true)
	assert.Nil(t, err)

	adminUser, err := deps.UserRepo.GetUserByName("admin-user")
	assert.Nil(t, err)
	assert.True(t, adminUser.IsAdmin)
	assert.Equal(t, tools.AdminStorageLimitInBytes, adminUser.StorageLimitInBytes)
}

func TestUserRepository_GetAllEmailsIncludesAdminAndCreatedMaintainers(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser(maintainers.QuollixAdminUsername, "hashed-admin-password", maintainers.QuollixAdminEmail, []byte{2, 2, 2}, tools.AdminStorageLimitInBytes, true)
	assert.Nil(t, err)

	emails, err := deps.UserRepo.GetAllEmails()
	assert.Nil(t, err)
	assert.Equal(t, []string{maintainers.QuollixAdminEmail}, emails)

	err = deps.UserRepo.CreateUser("normal-user", "hashed-password", "normal@example.com", []byte{3, 3, 3}, tools.UserStorageLimitInBytes, false)
	assert.Nil(t, err)

	emails, err = deps.UserRepo.GetAllEmails()
	assert.Nil(t, err)
	assert.Equal(t, []string{maintainers.QuollixAdminEmail, "normal@example.com"}, emails)
}

func TestUserRepository_GetMaintainerPublicKeyRecord(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	publicKey := u.GetOtherLocalTestingPublicKeyRaw()
	signature := make([]byte, ed25519.SignatureSize)
	err := deps.UserRepo.CreateUserWithSetupToken("normal-user", "normal@example.com", publicKey, signature, tools.UserStorageLimitInBytes, "setup-token-hash", time.Now().UTC().Add(time.Hour))
	assert.Nil(t, err)

	record, err := deps.UserRepo.GetMaintainerPublicKeyRecord("normal-user")
	assert.Nil(t, err)
	assert.Equal(t, "normal-user", record.Maintainer)
	assert.Equal(t, publicKey, record.PublicKeyRaw)
	assert.Equal(t, signature, record.PublicKeySignature)

	err = deps.UserRepo.CreateUser(maintainers.QuollixAdminUsername, "hashed-admin-password", maintainers.QuollixAdminEmail, []byte{2, 2, 2}, tools.AdminStorageLimitInBytes, true)
	assert.Nil(t, err)
	_, err = deps.UserRepo.GetMaintainerPublicKeyRecord(maintainers.QuollixAdminUsername)
	assert.Equal(t, maintainers.MaintainerNotFoundError, u.ExtractError(err))
}

func TestUserRepository_DeleteUserCascadesAppsAndVersions(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("normal-user", "hashed-password", "normal@example.com", []byte{1, 1, 1}, tools.UserStorageLimitInBytes, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("normal-user")
	assert.Nil(t, err)

	assert.Nil(t, deps.AppRepo.CreateApp(user.Id, "sample-app"))
	app, err := deps.AppRepo.GetAppByName(user.Id, "sample-app")
	assert.Nil(t, err)

	content := make([]byte, 101)
	_, err = deps.VersionRepo.CreateVersion(app.AppId, "1.0.0", time.Now().UTC(), content, versionContentHash(content), []byte{9, 9, 9})
	assert.Nil(t, err)

	assert.Nil(t, deps.UserRepo.DeleteUser(user.Id))

	exists, err := deps.UserRepo.DoesUserExist("normal-user")
	assert.Nil(t, err)
	assert.False(t, exists)

	appExists, err := deps.AppRepo.DoesAppExistByMaintainerName("normal-user", "sample-app")
	assert.Nil(t, err)
	assert.False(t, appExists)

	versionExists, err := deps.VersionRepo.DoesVersionNameExistByNames("normal-user", "sample-app", "1.0.0")
	assert.Nil(t, err)
	assert.False(t, versionExists)

	unrelatedExists, err := deps.UserRepo.DoesUserExist("someone-else")
	assert.Nil(t, err)
	assert.False(t, unrelatedExists)
}

func TestUserRepository_DeleteUserAllowsNameReuse(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("disabled-user", "hashed-password", "disabled@example.com", []byte{4, 4, 4}, tools.UserStorageLimitInBytes, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("disabled-user")
	assert.Nil(t, err)

	assert.Nil(t, deps.UserRepo.DeleteUser(user.Id))

	exists, err := deps.UserRepo.DoesUserExist("disabled-user")
	assert.Nil(t, err)
	assert.False(t, exists)

	_, err = deps.UserRepo.GetUserByName("disabled-user")
	assert.NotNil(t, err)
}
