package remote

import (
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func setupSigningKeyManagerTest(t *testing.T) (*SigningKeyManagerImpl, *SessionManagerMock, *OsWrapperMock) {
	sessionManagerMock := NewSessionManagerMock(t)
	osWrapperMock := NewOsWrapperMock(t)
	return &SigningKeyManagerImpl{
		SessionManager: sessionManagerMock,
		OsWrapper:      osWrapperMock,
	}, sessionManagerMock, osWrapperMock
}

func TestSigningKeyManagerStorePublicKey_InitializesSigningConfig(t *testing.T) {
	keyManager, sessionManagerMock, _ := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{}, nil)
	sessionManagerMock.EXPECT().SetConfig(&LocalConfig{
		Signing: &SigningConfig{
			PublicKeyRawBase64: base64.StdEncoding.EncodeToString([]byte{1, 2, 3}),
		},
	}).Return(nil)

	err := keyManager.StorePublicKeyRaw([]byte{1, 2, 3})
	assert.Nil(t, err)
}

func TestSigningKeyManagerStorePublicKey_PreservesPrivateKeyPath(t *testing.T) {
	keyManager, sessionManagerMock, _ := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Signing: &SigningConfig{
			PrivateKeyPath: "/tmp/private-key.pem",
		},
	}, nil)
	sessionManagerMock.EXPECT().SetConfig(&LocalConfig{
		Signing: &SigningConfig{
			PublicKeyRawBase64: base64.StdEncoding.EncodeToString([]byte{4, 5, 6}),
			PrivateKeyPath:     "/tmp/private-key.pem",
		},
	}).Return(nil)

	err := keyManager.StorePublicKeyRaw([]byte{4, 5, 6})
	assert.Nil(t, err)
}

func TestSigningKeyManagerSetPrivateKeyPath_FailsWithoutSession(t *testing.T) {
	keyManager, sessionManagerMock, _ := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{}, nil)

	err := keyManager.SetPrivateKeyPath("/tmp/private-key.pem", u.LocalTestingPrivateKeyPassphrase)
	assert.Equal(t, signingKeyNeedsSessionError, u.ExtractError(err))
}

func TestSigningKeyManagerSetPrivateKeyPath_FailsWithoutStoredPublicKey(t *testing.T) {
	keyManager, sessionManagerMock, _ := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
	}, nil)

	err := keyManager.SetPrivateKeyPath("/tmp/private-key.pem", u.LocalTestingPrivateKeyPassphrase)
	assert.Equal(t, signingKeyNeedsStoredPublicKeyError, u.ExtractError(err))
}

func TestSigningKeyManagerSetPrivateKeyPath_FailsWhenReadingPrivateKey(t *testing.T) {
	keyManager, sessionManagerMock, osWrapperMock := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{PublicKeyRawBase64: base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw())},
	}, nil)
	osWrapperMock.EXPECT().GetFileMode("/tmp/private-key.pem").Return(os.FileMode(0o600), nil)
	osWrapperMock.EXPECT().ReadFile("/tmp/private-key.pem").Return(nil, errors.New("boom"))

	err := keyManager.SetPrivateKeyPath("/tmp/private-key.pem", u.LocalTestingPrivateKeyPassphrase)
	assert.NotNil(t, err)
}

func TestSigningKeyManagerSetPrivateKeyPath_FailsForOpenPermissions(t *testing.T) {
	keyManager, sessionManagerMock, osWrapperMock := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{PublicKeyRawBase64: base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw())},
	}, nil)
	osWrapperMock.EXPECT().GetFileMode("/tmp/private-key.pem").Return(os.FileMode(0o644), nil)

	err := keyManager.SetPrivateKeyPath("/tmp/private-key.pem", u.LocalTestingPrivateKeyPassphrase)
	assert.Equal(t, "private key file permissions are too open; run: chmod 600 /tmp/private-key.pem", u.ExtractError(err))
}

func TestSigningKeyManagerSetPrivateKeyPath_FailsForInvalidPrivateKey(t *testing.T) {
	keyManager, sessionManagerMock, osWrapperMock := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{PublicKeyRawBase64: base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw())},
	}, nil)
	osWrapperMock.EXPECT().GetFileMode("/tmp/private-key.pem").Return(os.FileMode(0o600), nil)
	osWrapperMock.EXPECT().ReadFile("/tmp/private-key.pem").Return([]byte("invalid"), nil)

	err := keyManager.SetPrivateKeyPath("/tmp/private-key.pem", u.LocalTestingPrivateKeyPassphrase)
	assert.NotNil(t, err)
}

func TestSigningKeyManagerSetPrivateKeyPath_FailsForKeyMismatch(t *testing.T) {
	keyManager, sessionManagerMock, osWrapperMock := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{PublicKeyRawBase64: base64.StdEncoding.EncodeToString([]byte("different"))},
	}, nil)
	osWrapperMock.EXPECT().GetFileMode("/tmp/private-key.pem").Return(os.FileMode(0o600), nil)
	osWrapperMock.EXPECT().ReadFile("/tmp/private-key.pem").Return([]byte(u.LocalTestingPrivateKeyOpenSSH), nil)

	err := keyManager.SetPrivateKeyPath("/tmp/private-key.pem", u.LocalTestingPrivateKeyPassphrase)
	assert.Equal(t, privateKeyMismatchError, u.ExtractError(err))
}

func TestSigningKeyManagerSetPrivateKeyPath_SavesPath(t *testing.T) {
	keyManager, sessionManagerMock, osWrapperMock := setupSigningKeyManagerTest(t)
	config := &LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{PublicKeyRawBase64: base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw())},
	}
	sessionManagerMock.EXPECT().GetConfig().Return(config, nil)
	osWrapperMock.EXPECT().GetFileMode("/tmp/private-key.pem").Return(os.FileMode(0o600), nil)
	osWrapperMock.EXPECT().ReadFile("/tmp/private-key.pem").Return([]byte(u.LocalTestingPrivateKeyOpenSSH), nil)
	sessionManagerMock.EXPECT().SetConfig(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{
			PublicKeyRawBase64: base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw()),
			PrivateKeyPath:     "/tmp/private-key.pem",
		},
	}).Return(nil)

	err := keyManager.SetPrivateKeyPath("/tmp/private-key.pem", u.LocalTestingPrivateKeyPassphrase)
	assert.Nil(t, err)
}

func TestSigningKeyManagerValidateConfiguredPrivateKey_FailsWithoutStoredPublicKey(t *testing.T) {
	keyManager, sessionManagerMock, _ := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
	}, nil)

	err := keyManager.ValidateConfiguredPrivateKey(u.LocalTestingPrivateKeyPassphrase)
	assert.Equal(t, signingKeyValidationNeedsLocalPublicKeyError, u.ExtractError(err))
}

func TestSigningKeyManagerValidateConfiguredPrivateKey_FailsWithoutPrivateKeyPath(t *testing.T) {
	keyManager, sessionManagerMock, _ := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{PublicKeyRawBase64: base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw())},
	}, nil)

	err := keyManager.ValidateConfiguredPrivateKey(u.LocalTestingPrivateKeyPassphrase)
	assert.Equal(t, privateKeyPathMissingError, u.ExtractError(err))
}

func TestSigningKeyManagerValidateConfiguredPrivateKey_FailsForKeyMismatch(t *testing.T) {
	keyManager, sessionManagerMock, osWrapperMock := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{
			PublicKeyRawBase64: base64.StdEncoding.EncodeToString([]byte("different")),
			PrivateKeyPath:     "/tmp/private-key.pem",
		},
	}, nil)
	osWrapperMock.EXPECT().GetFileMode("/tmp/private-key.pem").Return(os.FileMode(0o600), nil)
	osWrapperMock.EXPECT().ReadFile("/tmp/private-key.pem").Return([]byte(u.LocalTestingPrivateKeyOpenSSH), nil)

	err := keyManager.ValidateConfiguredPrivateKey(u.LocalTestingPrivateKeyPassphrase)
	assert.Equal(t, privateKeyMismatchError, u.ExtractError(err))
}

func TestSigningKeyManagerValidateConfiguredPrivateKey_Succeeds(t *testing.T) {
	keyManager, sessionManagerMock, osWrapperMock := setupSigningKeyManagerTest(t)
	sessionManagerMock.EXPECT().GetConfig().Return(&LocalConfig{
		Session: &SessionData{Maintainer: "alice"},
		Signing: &SigningConfig{
			PublicKeyRawBase64: base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw()),
			PrivateKeyPath:     "/tmp/private-key.pem",
		},
	}, nil)
	osWrapperMock.EXPECT().GetFileMode("/tmp/private-key.pem").Return(os.FileMode(0o600), nil)
	osWrapperMock.EXPECT().ReadFile("/tmp/private-key.pem").Return([]byte(u.LocalTestingPrivateKeyOpenSSH), nil)

	err := keyManager.ValidateConfiguredPrivateKey(u.LocalTestingPrivateKeyPassphrase)
	assert.Nil(t, err)
}
