package commands

import (
	"encoding/base64"
	"qsc/configuration"
	"qsc/tools"
	"testing"
	"time"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

var sampleSessionExpirationDate = time.Date(2026, 9, 10, 12, 30, 0, 0, time.UTC)

func getSampleLocalConfig() *configuration.Config {
	return &configuration.Config{
		AppsDirectory: "/home/alice/apps",
		Session: &configuration.SessionData{
			Maintainer:     "alice",
			Cookie:         "session-cookie-secret",
			ExpirationDate: sampleSessionExpirationDate,
		},
		Signing: &configuration.SigningConfig{
			PublicKeyRawBase64: base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw()),
			PrivateKeyPath:     "/home/alice/.ssh/id_ed25519",
		},
		DockerHub: &tools.DockerHubAuth{
			Username: "alice-docker",
			Token:    "docker-hub-token-secret",
		},
		TrustedMaintainerKeys: map[string]configuration.TrustedMaintainerKey{
			"quollix": {PublicKeyRawBase64: "public-key", PublicKeySignatureBase64: "signature"},
		},
	}
}

func getSampleGlobalConfig() *tools.GlobalConfig {
	return &tools.GlobalConfig{
		ConfigFilePath:  "/home/alice/.config/qsc/config.yml",
		AppStoreRootURL: "https://store.quollix.org",
	}
}

func TestRenderSessionConfig_ReturnsNotConfiguredWithoutSession(t *testing.T) {
	config := getSampleLocalConfig()
	config.Session = nil
	config.AppsDirectory = ""
	config.Signing = nil
	config.DockerHub = nil
	config.TrustedMaintainerKeys = nil

	output := renderSessionConfig(config, nil)

	expectedOutput := `- apps_directory: not configured
- session:
  - not configured
- signing:
  - not configured
- docker_hub:
  - not configured
- trusted_maintainer_keys:
  - count: 0
`
	assert.Equal(t, expectedOutput, output)
}

func TestRenderSessionConfig_IncludesLocalConfigMetadata(t *testing.T) {
	output := renderSessionConfig(getSampleLocalConfig(), getSampleGlobalConfig())

	expectedOutput := `- config_file: /home/alice/.config/qsc/config.yml
- server_url: https://store.quollix.org
- apps_directory: /home/alice/apps
- session:
  - maintainer: alice
  - expiration_date: 2026-09-10 12:30:00
  - access_cookie: *****************cret
- signing:
  - public_key_configured: true
  - public_key_fingerprint: SHA256:QL91usdSz5KndtEmrv1z4p4KJTUpMA9Vqhqpzqduhbc
  - private_key_path: /home/alice/.ssh/id_ed25519
- docker_hub:
  - username: alice-docker
  - token: *******************cret
- trusted_maintainer_keys:
  - count: 1
`
	assert.Equal(t, expectedOutput, output)
}

func TestRenderAppSearchTable_UsesAlignedColumns(t *testing.T) {
	output := renderAppSearchTable([]store.AppWithLatestVersion{
		{
			Maintainer:                     "sample-maintainer",
			AppName:                        "nginx",
			LatestVersionName:              "1.27.4",
			LatestVersionCreationTimestamp: time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC),
		},
	})

	expectedOutput := `maintainer         app    latest version  latest version timestamp
sample-maintainer  nginx  1.27.4          2026-09-04 10:00:00
`
	assert.Equal(t, expectedOutput, output)
}

func TestRenderIndexedVersionsTable_UsesAlignedColumns(t *testing.T) {
	output := renderIndexedVersionsTable([]store.LeanVersionDto{
		{
			VersionId:             3,
			Name:                  "1.27.4",
			CreationTimestamp:     versionSelectorBaseTimestamp,
			SizeInBytes:           1234,
			IsMigrationCheckpoint: true,
			DownloadCount:         5,
		},
		{
			VersionId:         2,
			Name:              "1.27.3",
			CreationTimestamp: versionSelectorBaseTimestamp.Add(-time.Minute),
			SizeInBytes:       987,
			DownloadCount:     4,
		},
	})

	expectedOutput := `index  version  timestamp            size     migration checkpoint  downloads
0      1.27.4   2026-09-04 10:00:00  1.21 KB  true                  5
1      1.27.3   2026-09-04 09:59:00  987 B    false                 4
`
	assert.Equal(t, expectedOutput, output)
}

func TestRenderMaintainersTable_ShowsStatusAndPublicKeyFingerprint(t *testing.T) {
	output := renderMaintainersTable([]store.AdminMaintainer{
		{Name: "alice", Email: "alice@example.com", PublicKeyRaw: u.GetLocalTestingPublicKeyRaw(), IsActive: true},
		{Name: "bob", Email: "bob@example.com", PublicKeyRaw: u.GetOtherLocalTestingPublicKeyRaw(), IsActive: false},
	})

	expectedOutput := `name   email              status   public key fingerprint
alice  alice@example.com  active   SHA256:QL91usdSz5KndtEmrv1z4p4KJTUpMA9Vqhqpzqduhbc
bob    bob@example.com    pending  SHA256:RVqr2+zRJn6XkjcOfl80D6eO3fJygFwqG+jsX4LZPYA
`
	assert.Equal(t, expectedOutput, output)
}
