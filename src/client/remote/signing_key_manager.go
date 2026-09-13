package remote

import (
	"bytes"
	"encoding/base64"
	"fmt"

	u "github.com/quollix/common/utils"
)

type SigningKeyManager interface {
	StorePublicKeyRaw(publicKeyRaw []byte) error
	SetPrivateKeyPath(path string, passphrase string) error
	GetPrivateKeyPath() (string, error)
	ValidateConfiguredPrivateKey(passphrase string) error
}

const (
	signingKeyNeedsSessionError                  = "setting a private key path requires an active session"
	signingKeyNeedsStoredPublicKeyError          = "setting a private key path requires a stored local account public key"
	signingKeyValidationNeedsSessionError        = "upload requires an active session"
	signingKeyValidationNeedsLocalPublicKeyError = "upload requires a stored local account public key"
	privateKeyPathMissingError                   = "upload requires a configured private key path"
	privateKeyMismatchError                      = "private key does not match stored public key"
	privateKeyOpenPermissionsErrorFormat         = "private key file permissions are too open; run: chmod 600 %s"
	privateKeyUnprotectedErrorFormat             = "private key file must be passphrase-protected; run: ssh-keygen -p -o -a 100 -f %s"
)

type SigningKeyManagerImpl struct {
	SessionManager SessionManager
	OsWrapper      u.OsWrapper
}

func (s *SigningKeyManagerImpl) StorePublicKeyRaw(publicKeyRaw []byte) error {
	config, err := s.SessionManager.GetConfig()
	if err != nil {
		return err
	}
	if config.Signing == nil {
		config.Signing = &SigningConfig{}
	}
	config.Signing.PublicKeyRawBase64 = base64.StdEncoding.EncodeToString(publicKeyRaw)
	return s.SessionManager.SetConfig(config)
}

func (s *SigningKeyManagerImpl) SetPrivateKeyPath(path string, passphrase string) error {
	config, err := s.getConfigWithStoredPublicKeyRaw()
	if err != nil {
		return err
	}
	if err := s.validatePrivateKeyPath(path, passphrase, config.Signing.PublicKeyRawBase64); err != nil {
		return err
	}
	config.Signing.PrivateKeyPath = path
	return s.SessionManager.SetConfig(config)
}

func (s *SigningKeyManagerImpl) ValidateConfiguredPrivateKey(passphrase string) error {
	config, err := s.getConfigWithStoredPublicKeyRawForValidation()
	if err != nil {
		return err
	}
	if config.Signing.PrivateKeyPath == "" {
		return u.Logger.NewError(privateKeyPathMissingError)
	}
	return s.validatePrivateKeyPath(config.Signing.PrivateKeyPath, passphrase, config.Signing.PublicKeyRawBase64)
}

func (s *SigningKeyManagerImpl) getConfigWithStoredPublicKeyRawForValidation() (*LocalConfig, error) {
	config, err := s.SessionManager.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.Session == nil {
		return nil, u.Logger.NewError(signingKeyValidationNeedsSessionError)
	}
	if config.Signing == nil || config.Signing.PublicKeyRawBase64 == "" {
		return nil, u.Logger.NewError(signingKeyValidationNeedsLocalPublicKeyError)
	}
	return config, nil
}

func (s *SigningKeyManagerImpl) getConfigWithStoredPublicKeyRaw() (*LocalConfig, error) {
	config, err := s.SessionManager.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.Session == nil {
		return nil, u.Logger.NewError(signingKeyNeedsSessionError)
	}
	if config.Signing == nil || config.Signing.PublicKeyRawBase64 == "" {
		return nil, u.Logger.NewError(signingKeyNeedsStoredPublicKeyError)
	}
	return config, nil
}

func (s *SigningKeyManagerImpl) validatePrivateKeyPath(path string, passphrase string, encodedStoredPublicKeyRaw string) error {
	storedPublicKeyRaw, err := base64.StdEncoding.DecodeString(encodedStoredPublicKeyRaw)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	mode, err := s.OsWrapper.GetFileMode(path)
	if err != nil {
		return err
	}
	if mode.Perm()&0o077 != 0 {
		return u.Logger.NewError(fmt.Sprintf(privateKeyOpenPermissionsErrorFormat, path), "path", path, "mode", mode.Perm())
	}
	privateKeyBytes, err := s.OsWrapper.ReadFile(path)
	if err != nil {
		return err
	}
	protected, err := u.IsPrivateKeyPassphraseProtectedOpenSSH(privateKeyBytes)
	if err != nil {
		return err
	}
	if !protected {
		return u.Logger.NewError(fmt.Sprintf(privateKeyUnprotectedErrorFormat, path), "path", path)
	}
	publicKeyRaw, err := GetPublicKeyRawFromPrivateKey(privateKeyBytes, passphrase)
	if err != nil {
		return err
	}
	if !bytes.Equal(publicKeyRaw, storedPublicKeyRaw) {
		return u.Logger.NewError(privateKeyMismatchError, "path", path)
	}
	return nil
}

func (s *SigningKeyManagerImpl) GetPrivateKeyPath() (string, error) {
	config, err := s.SessionManager.GetConfig()
	if err != nil {
		return "", err
	}
	if config.Signing == nil || config.Signing.PrivateKeyPath == "" {
		return "", u.Logger.NewError(privateKeyPathMissingError)
	}
	return config.Signing.PrivateKeyPath, nil
}
