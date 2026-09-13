//go:build integration

package repository

import (
	"testing"
	"time"

	"server/maintainers"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestVersionRepository_CreateVersionRejectsExceededStorageLimit(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("limited-user", "hashed-password", "limited@example.com", []byte{1, 2, 3}, 150, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("limited-user")
	assert.Nil(t, err)

	assert.Nil(t, deps.AppRepo.CreateApp(user.Id, "sample-app"))
	app, err := deps.AppRepo.GetAppByName(user.Id, "sample-app")
	assert.Nil(t, err)

	content := make([]byte, 151)
	_, err = deps.VersionRepo.CreateVersion(app.AppId, "1.0.0", time.Now().UTC(), content, versionContentHash(content), []byte{9, 9, 9})
	assert.NotNil(t, err)
	assert.Equal(t, maintainers.MaximumStorageExceededError, u.ExtractError(err))

	reloadedUser, err := deps.UserRepo.GetUserByName("limited-user")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), reloadedUser.UsedSpaceInBytes)

	exists, err := deps.VersionRepo.DoesVersionNameExistByNames("limited-user", "sample-app", "1.0.0")
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestVersionRepository_CreateVersionAllowsDuplicateVersionNamePerApp(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("limited-user", "hashed-password", "limited@example.com", []byte{1, 2, 3}, 1024, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("limited-user")
	assert.Nil(t, err)

	assert.Nil(t, deps.AppRepo.CreateApp(user.Id, "sample-app"))
	app, err := deps.AppRepo.GetAppByName(user.Id, "sample-app")
	assert.Nil(t, err)

	firstContent := []byte("first version content")
	_, err = deps.VersionRepo.CreateVersion(app.AppId, "1.0.0", time.Now().UTC(), firstContent, versionContentHash(firstContent), []byte{9, 9, 9})
	assert.Nil(t, err)

	secondContent := []byte("second version content")
	_, err = deps.VersionRepo.CreateVersion(app.AppId, "1.0.0", time.Now().UTC(), secondContent, versionContentHash(secondContent), []byte{8, 8, 8})
	assert.Nil(t, err)

	exists, err := deps.VersionRepo.DoesVersionNameExistByNames("limited-user", "sample-app", "1.0.0")
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestVersionRepository_DoesVersionContentHashExist(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("limited-user", "hashed-password", "limited@example.com", []byte{1, 2, 3}, 1024, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("limited-user")
	assert.Nil(t, err)

	assert.Nil(t, deps.AppRepo.CreateApp(user.Id, "sample-app"))
	app, err := deps.AppRepo.GetAppByName(user.Id, "sample-app")
	assert.Nil(t, err)

	content := []byte("version content")
	contentHash := versionContentHash(content)
	exists, err := deps.VersionRepo.DoesVersionContentHashExist(contentHash)
	assert.Nil(t, err)
	assert.False(t, exists)

	_, err = deps.VersionRepo.CreateVersion(app.AppId, "1.0.0", time.Now().UTC(), content, contentHash, []byte{9, 9, 9})
	assert.Nil(t, err)

	exists, err = deps.VersionRepo.DoesVersionContentHashExist(contentHash)
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestVersionRepository_DoesVersionNameExistByAppID(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("limited-user", "hashed-password", "limited@example.com", []byte{1, 2, 3}, 1024, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("limited-user")
	assert.Nil(t, err)

	assert.Nil(t, deps.AppRepo.CreateApp(user.Id, "sample-app"))
	app, err := deps.AppRepo.GetAppByName(user.Id, "sample-app")
	assert.Nil(t, err)

	exists, err := deps.VersionRepo.DoesVersionNameExistByAppID(app.AppId, "1.0.0")
	assert.Nil(t, err)
	assert.False(t, exists)

	content := make([]byte, 101)
	_, err = deps.VersionRepo.CreateVersion(app.AppId, "1.0.0", time.Now().UTC(), content, versionContentHash(content), []byte{9, 9, 9})
	assert.Nil(t, err)

	exists, err = deps.VersionRepo.DoesVersionNameExistByAppID(app.AppId, "1.0.0")
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestVersionRepository_DeleteVersionUpdatesUsedSpace(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("limited-user", "hashed-password", "limited@example.com", []byte{1, 2, 3}, 1024, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("limited-user")
	assert.Nil(t, err)

	assert.Nil(t, deps.AppRepo.CreateApp(user.Id, "sample-app"))
	app, err := deps.AppRepo.GetAppByName(user.Id, "sample-app")
	assert.Nil(t, err)

	versionBytes := make([]byte, 101)
	createdVersion, err := deps.VersionRepo.CreateVersion(app.AppId, "1.0.0", time.Now().UTC(), versionBytes, versionContentHash(versionBytes), []byte{9, 9, 9})
	assert.Nil(t, err)

	reloadedUser, err := deps.UserRepo.GetUserByName("limited-user")
	assert.Nil(t, err)
	assert.Equal(t, int64(len(versionBytes)), reloadedUser.UsedSpaceInBytes)

	assert.Nil(t, deps.VersionRepo.DeleteVersionByID(createdVersion.VersionId))

	reloadedUser, err = deps.UserRepo.GetUserByName("limited-user")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), reloadedUser.UsedSpaceInBytes)

	exists, err := deps.VersionRepo.DoesVersionNameExistByAppID(app.AppId, "1.0.0")
	assert.Nil(t, err)
	assert.False(t, exists)
}
