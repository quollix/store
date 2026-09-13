//go:build component

package check

import (
	"server/maintainers"
	"server/setup"
	"testing"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestEmailConfig(t *testing.T) {
	storeClient := GetStoreClient()
	defer storeClient.WipeData()
	assert.Nil(t, storeClient.Login(maintainers.QuollixAdminUsername, maintainers.QuollixAdminPassword))
	config, err := storeClient.GetEmailConfig()
	assert.Nil(t, err)
	assert.Equal(t, maintainers.DefaultEmailConfig, *config)

	sampleEmailConfig := u.SampleEmailConfig
	assert.Nil(t, storeClient.SetEmailConfig(&sampleEmailConfig))

	updatedConfig, err := storeClient.GetEmailConfig()
	assert.Nil(t, err)
	assert.Equal(t, sampleEmailConfig, *updatedConfig)
}

func TestEmailConfigCanNotBeChangedByNonAdminUser(t *testing.T) {
	storeClient := GetStoreClientAndLogin(t)
	defer storeClient.WipeData()
	_, err := storeClient.GetEmailConfig()
	u.AssertDeepStackErrorFromRequest(t, err, setup.AccessDeniedForNonAdminUserError)
	err = storeClient.SetEmailConfig(&maintainers.DefaultEmailConfig)
	u.AssertDeepStackErrorFromRequest(t, err, setup.AccessDeniedForNonAdminUserError)
}
