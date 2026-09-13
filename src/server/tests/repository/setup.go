//go:build integration

package repository

import (
	"crypto/sha256"
	"server/di"
	"server/setup"
	"server/versions"
	"testing"

	"github.com/quollix/common/assert"
)

var (
	isInitialized bool
	deps          *setup.InitializerDependencies
)

func InitDeps(t *testing.T) {
	if isInitialized {
		return
	}

	deps = di.WireDependencies()
	deps.DatabaseProvider.Host = "localhost"
	deps.DatabaseSnapshotRepository.DatabaseHost = "localhost"
	assert.Nil(t, deps.PathProvider.Initialize())
	assert.Nil(t, deps.DatabaseProvider.InitializeDatabase())
	assert.Nil(t, deps.DatabaseSnapshotRepository.CreateDatabaseSnapshot())
	isInitialized = true
}

func WipeDatabase(t *testing.T) {
	assert.NotNil(t, deps)
	assert.Nil(t, deps.DatabaseSnapshotRepository.ResetDatabaseToSnapshot())
}

func versionContentHash(content []byte) versions.VersionContentHash {
	return versions.VersionContentHash(sha256.Sum256(content))
}
