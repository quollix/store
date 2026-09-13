package tools

import (
	"testing"

	"github.com/quollix/common/assert"
)

func TestInitGlobalConfig(t *testing.T) {
	testConfig := InitGlobalConfig("TEST", "/tmp/user-config")
	assert.Equal(t, "http://localhost:8080", testConfig.AppStoreRootURL)
	assert.Equal(t, "/tmp/user-config/qsc-test/config.yml", testConfig.ConfigFilePath)
	assert.False(t, testConfig.VerifyAppStoreCertificate)

	prodConfig := InitGlobalConfig("something-else", "/tmp/user-config")
	assert.Equal(t, "https://store.quollix.org", prodConfig.AppStoreRootURL)
	assert.Equal(t, "/tmp/user-config/qsc/config.yml", prodConfig.ConfigFilePath)
	assert.True(t, prodConfig.VerifyAppStoreCertificate)
}
