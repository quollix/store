package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quollix/common/assert"
)

func TestGetUserFromContext_ReturnsUserAndOk(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	expectedUser := &User{Name: "sample"}
	req = req.WithContext(context.WithValue(req.Context(), UserCtxKey, expectedUser))

	user, ok := GetUserFromContext(req)

	assert.True(t, ok)
	assert.Equal(t, expectedUser, user)
}

func TestGetUserFromContext_MissingUserReturnsFalse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	user, ok := GetUserFromContext(req)

	assert.False(t, ok)
	assert.Nil(t, user)
}

func TestGetUserFromContextOrWriteError_WritesErrorWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	user, ok := GetUserFromContextOrWriteError(recorder, req)

	assert.False(t, ok)
	assert.Nil(t, user)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, UserMissingInRequestContextError+"\n", recorder.Body.String())
}
