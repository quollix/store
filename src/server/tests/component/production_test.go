//go:build production

package check

import (
	"server/maintainers"
	"server/setup"
	"server/tools"
	"testing"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
)

func TestWipeEndpointIsNotAvailableInProduction(t *testing.T) {
	storeClient := GetStoreClient()
	body, err := storeClient.Parent.DoRequest(store.WipeDataPath, nil)
	assert.Nil(t, err)
	// If an endpoint is not found, the frontend is returned as the default fallback.
	assert.Equal(t, setup.PlaceHolderFrontend, string(body))
}

func TestInitialAdminCanLoginInProduction(t *testing.T) {
	storeClient := GetStoreClient()

	assert.Nil(t, storeClient.Login(maintainers.QuollixAdminUsername, maintainers.QuollixAdminPassword))
	account, err := storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, maintainers.QuollixAdminUsername, account.Name)
	assert.Equal(t, tools.AdminStorageLimitInBytes, account.StorageLimitInBytes)
	assert.True(t, account.IsAdmin)
}
