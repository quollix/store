//go:build component

package check

import (
	"encoding/json"
	"io"
	"net/http"
	"server/setup"
	"testing"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/deploy"
	u "github.com/quollix/common/utils"
)

func TestFrontendPlaceholderHandler(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/")
	assert.Nil(t, err)
	defer u.Close(resp.Body)

	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)

	assert.Equal(t, setup.PlaceHolderFrontend, string(body))
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestHealthEndpoint(t *testing.T) {
	storeClient := GetStoreClient()

	responseBody, err := storeClient.Parent.DoRequest(deploy.HealthPath, nil)
	assert.Nil(t, err)

	var healthResponse map[string]string
	assert.Nil(t, json.Unmarshal(responseBody, &healthResponse))
	assert.Equal(t, "ok", healthResponse["status"])
}
