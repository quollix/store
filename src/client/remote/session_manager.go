package remote

import (
	"encoding/base64"
	"errors"
	"net/http"

	"qsc/configuration"
	"qsc/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

var (
	sessionExpiredError    = "session expired, please sign in again"
	missingMaintainerError = "requires an active session or --maintainer for offline usage"
)

type SessionManager interface {
	RequireSession() (*configuration.SessionData, error)
	ResolveMaintainer(inputMaintainer string) (string, error)
	DeleteSession() error
	GetDockerHubAuth() (*tools.DockerHubAuth, error)
	SetDockerHubAuth(username, token string) error
	DeleteDockerHubAuth() error
	ApplySessionFromConfig() error
	PersistCurrentSession() error
	LoadMaintainerPublicKeyRecord(maintainer string) (*store.MaintainerPublicKeyRecord, error)
	SaveMaintainerPublicKeyRecord(record *store.MaintainerPublicKeyRecord) error
}

type SessionManagerImpl struct {
	AppStoreClient *store.AppStoreClientImpl
	ConfigProvider configuration.Provider
	OsWrapper      u.OsWrapper
}

func (c *SessionManagerImpl) SetConfig(config *configuration.Config) error {
	return c.ConfigProvider.SetConfig(config)
}

func (c *SessionManagerImpl) GetConfig() (*configuration.Config, error) {
	return c.ConfigProvider.GetConfig()
}

func (c *SessionManagerImpl) RequireSession() (*configuration.SessionData, error) {
	config, err := c.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.Session == nil {
		return nil, u.Logger.NewError(tools.MissingSessionError)
	}
	return config.Session, nil
}

func (c *SessionManagerImpl) ResolveMaintainer(inputMaintainer string) (string, error) {
	if inputMaintainer != "" {
		return inputMaintainer, nil
	}

	config, err := c.GetConfig()
	if err != nil {
		return "", err
	}
	if config.Session != nil && config.Session.Maintainer != "" {
		return config.Session.Maintainer, nil
	}
	return "", u.Logger.NewError(missingMaintainerError)
}

func (c *SessionManagerImpl) ApplySessionFromConfig() error {
	config, err := c.GetConfig()
	if err != nil {
		return err
	}
	if config.Session == nil {
		return nil
	}
	session := config.Session
	if session.ExpirationDate.Before(c.OsWrapper.Now()) {
		return errors.New(sessionExpiredError)
	}
	c.AppStoreClient.Parent.Cookie = &http.Cookie{ // #nosec G124 -- local client restores a trusted session cookie for outbound requests
		Name:    "auth",
		Value:   session.Cookie,
		Expires: session.ExpirationDate,
	}
	return nil
}

func (c *SessionManagerImpl) DeleteSession() error {
	config, err := c.GetConfig()
	if err != nil {
		return err
	}
	config.Session = nil
	c.AppStoreClient.Parent.Cookie = nil
	return c.SetConfig(config)
}

func (c *SessionManagerImpl) GetDockerHubAuth() (*tools.DockerHubAuth, error) {
	config, err := c.GetConfig()
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	return config.DockerHub, nil
}

func (c *SessionManagerImpl) SetDockerHubAuth(username, token string) error {
	if username == "" {
		return u.Logger.NewError("Docker Hub username must not be empty")
	}
	if token == "" {
		return u.Logger.NewError("Docker Hub token must not be empty")
	}
	config, err := c.GetConfig()
	if err != nil {
		return err
	}
	config.DockerHub = &tools.DockerHubAuth{
		Username: username,
		Token:    token,
	}
	return c.SetConfig(config)
}

func (c *SessionManagerImpl) DeleteDockerHubAuth() error {
	config, err := c.GetConfig()
	if err != nil {
		return err
	}
	config.DockerHub = nil
	return c.SetConfig(config)
}

func (c *SessionManagerImpl) PersistCurrentSession() error {
	config, err := c.GetConfig()
	if err != nil {
		return err
	}
	if config.Session == nil || c.AppStoreClient.Parent.Cookie == nil {
		return nil
	}
	config.Session.Cookie = c.AppStoreClient.Parent.Cookie.Value
	config.Session.ExpirationDate = c.AppStoreClient.Parent.Cookie.Expires
	return c.SetConfig(config)
}

func (c *SessionManagerImpl) LoadMaintainerPublicKeyRecord(maintainer string) (*store.MaintainerPublicKeyRecord, error) {
	config, err := c.GetConfig()
	if err != nil {
		return nil, err
	}
	if config == nil || config.TrustedMaintainerKeys == nil {
		return nil, u.Logger.NewError("maintainer public key cache miss")
	}
	cachedRecord, ok := config.TrustedMaintainerKeys[maintainer]
	if !ok {
		return nil, u.Logger.NewError("maintainer public key cache miss")
	}
	publicKeyRaw, err := base64.StdEncoding.DecodeString(cachedRecord.PublicKeyRawBase64)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	publicKeySignature, err := base64.StdEncoding.DecodeString(cachedRecord.PublicKeySignatureBase64)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	return &store.MaintainerPublicKeyRecord{
		Maintainer:         maintainer,
		PublicKeyRaw:       publicKeyRaw,
		PublicKeySignature: publicKeySignature,
	}, nil
}

func (c *SessionManagerImpl) SaveMaintainerPublicKeyRecord(record *store.MaintainerPublicKeyRecord) error {
	if record == nil {
		return u.Logger.NewError("maintainer public key record must not be nil")
	}
	config, err := c.GetConfig()
	if err != nil {
		return err
	}
	if config.TrustedMaintainerKeys == nil {
		config.TrustedMaintainerKeys = map[string]configuration.TrustedMaintainerKey{}
	}
	config.TrustedMaintainerKeys[record.Maintainer] = configuration.TrustedMaintainerKey{
		PublicKeyRawBase64:       base64.StdEncoding.EncodeToString(record.PublicKeyRaw),
		PublicKeySignatureBase64: base64.StdEncoding.EncodeToString(record.PublicKeySignature),
	}
	return c.SetConfig(config)
}
