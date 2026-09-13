//go:build integration

package repository

import (
	"server/tools"
	"testing"
	"time"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
)

func TestAppRepository_DoesAppExistByMaintainerID(t *testing.T) {
	InitDeps(t)
	WipeDatabase(t)
	defer WipeDatabase(t)

	err := deps.UserRepo.CreateUser("normal-user", "hashed-password", "normal@example.com", []byte{1, 1, 1}, tools.UserStorageLimitInBytes, false)
	assert.Nil(t, err)

	user, err := deps.UserRepo.GetUserByName("normal-user")
	assert.Nil(t, err)

	exists, err := deps.AppRepo.DoesAppExistByMaintainerID(user.Id, "sample-app")
	assert.Nil(t, err)
	assert.False(t, exists)

	assert.Nil(t, deps.AppRepo.CreateApp(user.Id, "sample-app"))

	exists, err = deps.AppRepo.DoesAppExistByMaintainerID(user.Id, "sample-app")
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestAppRepository_SearchForAppsReturnsLatestVersion(t *testing.T) {
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

	firstTimestamp := time.Now().UTC().Add(-10 * time.Minute).Round(time.Microsecond)
	secondTimestamp := time.Now().UTC().Add(-5 * time.Minute).Round(time.Microsecond)
	firstContent := make([]byte, 101)
	secondContent := append(make([]byte, 100), 1)
	_, err = deps.VersionRepo.CreateVersion(app.AppId, "1.0.0", firstTimestamp, firstContent, versionContentHash(firstContent), []byte{9, 9, 9})
	assert.Nil(t, err)
	createdLatestVersion, err := deps.VersionRepo.CreateVersion(app.AppId, "1.0.1", secondTimestamp, secondContent, versionContentHash(secondContent), []byte{8, 8, 8})
	assert.Nil(t, err)

	results, err := deps.AppRepo.SearchForApps(store.SearchRequest{
		MaintainerSearchTerm: "normal-user",
		AppSearchTerm:        "sample-app",
		ShowUnofficialApps:   true,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "normal-user", results[0].Maintainer)
	assert.Equal(t, "sample-app", results[0].AppName)
	assert.Equal(t, createdLatestVersion.VersionId, results[0].LatestVersionId)
	assert.Equal(t, "1.0.1", results[0].LatestVersionName)
	assert.Equal(t, secondTimestamp, results[0].LatestVersionCreationTimestamp)
}
