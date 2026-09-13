package remote

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"qsc/tools"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3"
)

func setupSessionManagerTest(t *testing.T) (*SessionManagerImpl, *OsWrapperMock) {
	osWrapperMock := NewOsWrapperMock(t)
	return &SessionManagerImpl{
		AppStoreClient: &store.AppStoreClientImpl{
			Parent: u.ComponentClient{},
		},
		Config: &tools.GlobalConfig{
			ConfigFilePath: "/tmp/qsc/config.yml",
		},
		OsWrapper: osWrapperMock,
	}, osWrapperMock
}

func getSampleSession() SessionData {
	return SessionData{
		Maintainer:     "alice",
		Cookie:         "cookie123",
		ExpirationDate: time.Now().UTC().Add(1 * time.Hour),
	}
}

func getSampleSigningConfig() SigningConfig {
	return SigningConfig{
		PublicKeyRawBase64: "public-key",
		PrivateKeyPath:     "/tmp/private-key.pem",
	}
}

func TestSessionManagerSetConfigStoresSession(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	session := getSampleSession()
	osWrapperMock.EXPECT().MkdirAll("/tmp/qsc", os.FileMode(0o700)).Return(nil)
	osWrapperMock.EXPECT().WriteFile("/tmp/qsc/config.yml", mock.Anything, os.FileMode(0o600)).
		RunAndReturn(func(_ string, data []byte, _ os.FileMode) error {
			var actual LocalConfig
			assert.Nil(t, yaml.Unmarshal(data, &actual))
			assert.Equal(t, "alice", actual.Session.Maintainer)
			assert.Equal(t, "cookie123", actual.Session.Cookie)
			return nil
		})

	err := sessionManager.SetConfig(&LocalConfig{Session: &session})
	assert.Nil(t, err)
}

func TestSessionManagerApplySessionFromConfig(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	session := getSampleSession()
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

session:
  username: alice
  cookie: cookie123
  expiration_date: `+session.ExpirationDate.Format(time.RFC3339Nano)+`
`), nil)
	osWrapperMock.EXPECT().Now().Return(session.ExpirationDate.Add(-1 * time.Minute))

	err := sessionManager.ApplySessionFromConfig()
	assert.Nil(t, err)
	assert.Equal(t, &http.Cookie{
		Name:    "auth",
		Value:   "cookie123",
		Expires: session.ExpirationDate,
	}, sessionManager.AppStoreClient.Parent.Cookie)
}

func TestSessionManagerResolveMaintainer_UsesInputMaintainerBeforeSession(t *testing.T) {
	sessionManager, _ := setupSessionManagerTest(t)

	maintainer, err := sessionManager.ResolveMaintainer("offline-maintainer")
	assert.Nil(t, err)
	assert.Equal(t, "offline-maintainer", maintainer)
}

func TestSessionManagerResolveMaintainer_UsesSessionMaintainerWhenInputIsEmpty(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

session:
  username: alice
`), nil)

	maintainer, err := sessionManager.ResolveMaintainer("")
	assert.Nil(t, err)
	assert.Equal(t, "alice", maintainer)
}

func TestSessionManagerResolveMaintainer_FailsWithoutInputOrSession(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

signing:
  public_key_raw_base64: public-key
`), nil)

	maintainer, err := sessionManager.ResolveMaintainer("")
	assert.Equal(t, "", maintainer)
	assert.Equal(t, "requires an active session or --maintainer for offline usage", u.ExtractError(err))
}

func TestSessionManagerRequireSession_ReturnsStoredSession(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

session:
  username: alice
  cookie: cookie123
`), nil)

	session, err := sessionManager.RequireSession()
	assert.Nil(t, err)
	assert.Equal(t, "alice", session.Maintainer)
	assert.Equal(t, "cookie123", session.Cookie)
}

func TestSessionManagerRequireSession_FailsWithoutStoredSession(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

signing:
  public_key_raw_base64: public-key
`), nil)

	session, err := sessionManager.RequireSession()
	assert.Nil(t, session)
	assert.Equal(t, tools.MissingSessionError, u.ExtractError(err))
}

func TestSessionManagerApplySessionFromConfigExpired(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	expirationDate := time.Now().UTC().Add(-1 * time.Hour)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

session:
  username: alice
  cookie: cookie123
  expiration_date: `+expirationDate.Format(time.RFC3339Nano)+`
`), nil)
	osWrapperMock.EXPECT().Now().Return(expirationDate.Add(1 * time.Minute))

	err := sessionManager.ApplySessionFromConfig()
	assert.Equal(t, sessionExpiredError, err.Error())
}

func TestSessionManagerDeleteSessionPreservesOtherConfig(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	sessionManager.AppStoreClient.Parent.Cookie = &http.Cookie{Name: "auth", Value: "cookie123"}
	publicKeyRaw := []byte("public-key-raw")
	publicKeySignature := []byte("public-key-signature")
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

session:
  username: alice
  cookie: cookie123
  expiration_date: `+time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)+`
signing:
  public_key_raw_base64: public-key
  private_key_path: /tmp/private-key.pem
trusted_maintainer_keys:
  alice:
    public_key_raw_base64: `+base64.StdEncoding.EncodeToString(publicKeyRaw)+`
    public_key_signature_base64: `+base64.StdEncoding.EncodeToString(publicKeySignature)+`
`), nil)
	osWrapperMock.EXPECT().MkdirAll("/tmp/qsc", os.FileMode(0o700)).Return(nil)
	osWrapperMock.EXPECT().WriteFile("/tmp/qsc/config.yml", mock.Anything, os.FileMode(0o600)).
		RunAndReturn(func(_ string, data []byte, _ os.FileMode) error {
			var actual LocalConfig
			assert.Nil(t, yaml.Unmarshal(data, &actual))
			assert.Nil(t, actual.Session)
			assert.Equal(t, "public-key", actual.Signing.PublicKeyRawBase64)
			assert.Equal(t, "/tmp/private-key.pem", actual.Signing.PrivateKeyPath)
			assert.Equal(t, base64.StdEncoding.EncodeToString(publicKeyRaw), actual.TrustedMaintainerKeys["alice"].PublicKeyRawBase64)
			assert.Equal(t, base64.StdEncoding.EncodeToString(publicKeySignature), actual.TrustedMaintainerKeys["alice"].PublicKeySignatureBase64)
			return nil
		})

	err := sessionManager.DeleteSession()
	assert.Nil(t, err)
	assert.Nil(t, sessionManager.AppStoreClient.Parent.Cookie)
}

func TestSessionManagerPersistCurrentSessionUpdatesStoredCookieAndExpiration(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	session := getSampleSession()
	refreshedExpirationDate := session.ExpirationDate.Add(24 * time.Hour)
	sessionManager.AppStoreClient.Parent.Cookie = &http.Cookie{
		Name:    "auth",
		Value:   "cookie456",
		Expires: refreshedExpirationDate,
	}
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

session:
  username: alice
  cookie: cookie123
  expiration_date: `+session.ExpirationDate.Format(time.RFC3339Nano)+`
signing:
  public_key_raw_base64: public-key
  private_key_path: /tmp/private-key.pem
`), nil)
	osWrapperMock.EXPECT().MkdirAll("/tmp/qsc", os.FileMode(0o700)).Return(nil)
	osWrapperMock.EXPECT().WriteFile("/tmp/qsc/config.yml", mock.Anything, os.FileMode(0o600)).
		RunAndReturn(func(_ string, data []byte, _ os.FileMode) error {
			var actual LocalConfig
			assert.Nil(t, yaml.Unmarshal(data, &actual))
			assert.Equal(t, "alice", actual.Session.Maintainer)
			assert.Equal(t, "cookie456", actual.Session.Cookie)
			assert.Equal(t, refreshedExpirationDate, actual.Session.ExpirationDate)
			assert.Equal(t, "public-key", actual.Signing.PublicKeyRawBase64)
			return nil
		})

	err := sessionManager.PersistCurrentSession()
	assert.Nil(t, err)
}

func TestSessionManagerPersistCurrentSessionNoOpsWithoutStoredSession(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	sessionManager.AppStoreClient.Parent.Cookie = &http.Cookie{Name: "auth", Value: "cookie456", Expires: time.Now().UTC().Add(time.Hour)}
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

signing:
  public_key_raw_base64: public-key
`), nil)

	err := sessionManager.PersistCurrentSession()
	assert.Nil(t, err)
}

func TestSessionManagerSetConfigWritesSigningOnly(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	signingConfig := getSampleSigningConfig()
	osWrapperMock.EXPECT().MkdirAll("/tmp/qsc", os.FileMode(0o700)).Return(nil)
	osWrapperMock.EXPECT().WriteFile("/tmp/qsc/config.yml", mock.Anything, os.FileMode(0o600)).
		RunAndReturn(func(_ string, data []byte, _ os.FileMode) error {
			var actual LocalConfig
			assert.Nil(t, yaml.Unmarshal(data, &actual))
			assert.Equal(t, signingConfig.PublicKeyRawBase64, actual.Signing.PublicKeyRawBase64)
			return nil
		})

	err := sessionManager.SetConfig(&LocalConfig{
		Signing: &signingConfig,
	})
	assert.Nil(t, err)
}

func TestSessionManagerGetConfigReadsSigningConfig(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`

signing:
  public_key_raw_base64: public-key
  private_key_path: /tmp/private-key.pem
`), nil)

	config, err := sessionManager.GetConfig()
	assert.Nil(t, err)
	assert.Equal(t, "public-key", config.Signing.PublicKeyRawBase64)
	assert.Equal(t, "/tmp/private-key.pem", config.Signing.PrivateKeyPath)
}

func TestSessionManagerSetConfigWritesSigningConfig(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().MkdirAll("/tmp/qsc", os.FileMode(0o700)).Return(nil)
	osWrapperMock.EXPECT().WriteFile("/tmp/qsc/config.yml", mock.Anything, os.FileMode(0o600)).
		RunAndReturn(func(_ string, data []byte, _ os.FileMode) error {
			var actual LocalConfig
			assert.Nil(t, yaml.Unmarshal(data, &actual))
			assert.Equal(t, "public-key", actual.Signing.PublicKeyRawBase64)
			assert.Equal(t, "/tmp/private-key.pem", actual.Signing.PrivateKeyPath)
			return nil
		})

	err := sessionManager.SetConfig(&LocalConfig{
		Signing: &SigningConfig{
			PublicKeyRawBase64: "public-key",
			PrivateKeyPath:     "/tmp/private-key.pem",
		},
	})
	assert.Nil(t, err)
}

func TestSessionManagerMaintainerPublicKeyRecordRoundTrip(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	publicKeyRaw := []byte("public-key-raw")
	publicKeySignature := []byte("public-key-signature")
	var savedConfig []byte
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`{}`), nil).Once()
	osWrapperMock.EXPECT().MkdirAll("/tmp/qsc", os.FileMode(0o700)).Return(nil)
	osWrapperMock.EXPECT().WriteFile("/tmp/qsc/config.yml", mock.Anything, os.FileMode(0o600)).
		RunAndReturn(func(_ string, data []byte, _ os.FileMode) error {
			savedConfig = append([]byte(nil), data...)
			return nil
		})

	err := sessionManager.SaveMaintainerPublicKeyRecord(&store.MaintainerPublicKeyRecord{
		Maintainer:         "maintainer",
		PublicKeyRaw:       publicKeyRaw,
		PublicKeySignature: publicKeySignature,
	})
	assert.Nil(t, err)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return(savedConfig, nil).Once()

	record, err := sessionManager.LoadMaintainerPublicKeyRecord("maintainer")
	assert.Nil(t, err)
	assert.Equal(t, "maintainer", record.Maintainer)
	assert.Equal(t, publicKeyRaw, record.PublicKeyRaw)
	assert.Equal(t, publicKeySignature, record.PublicKeySignature)
}

func TestSessionManagerLoadMaintainerPublicKeyRecordReturnsCacheMiss(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte(`{}`), nil)

	record, err := sessionManager.LoadMaintainerPublicKeyRecord("maintainer")
	assert.Nil(t, record)
	assert.Equal(t, "maintainer public key cache miss", u.ExtractError(err))
}

func TestSessionManagerSetConfigWritesEmptyConfig(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().MkdirAll("/tmp/qsc", os.FileMode(0o700)).Return(nil)
	osWrapperMock.EXPECT().WriteFile("/tmp/qsc/config.yml", []byte("{}\n"), os.FileMode(0o600)).Return(nil)

	err := sessionManager.SetConfig(&LocalConfig{})
	assert.Nil(t, err)
}

func TestSessionManagerGetConfigWrapsYamlUnmarshalError(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return([]byte("invalid: ["), nil)

	_, err := sessionManager.GetConfig()
	assert.NotNil(t, err)
}

func TestSessionManagerSetConfigPassesWriteErrorUp(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().MkdirAll("/tmp/qsc", os.FileMode(0o700)).Return(nil)
	osWrapperMock.EXPECT().WriteFile("/tmp/qsc/config.yml", mock.Anything, os.FileMode(0o600)).Return(errors.New("boom"))

	err := sessionManager.SetConfig(&LocalConfig{
		Signing: &SigningConfig{
			PublicKeyRawBase64: "public-key",
		},
	})
	assert.NotNil(t, err)
}

func TestSessionManagerGetConfigPassesReadErrorUp(t *testing.T) {
	sessionManager, osWrapperMock := setupSessionManagerTest(t)
	osWrapperMock.EXPECT().ReadFile("/tmp/qsc/config.yml").Return(nil, errors.New("boom"))

	_, err := sessionManager.GetConfig()
	assert.NotNil(t, err)
}
