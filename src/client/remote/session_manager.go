package remote

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"qsc/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"gopkg.in/yaml.v3"
)

var (
	sessionExpiredError    = "session expired, please sign in again"
	missingMaintainerError = "requires an active session or --maintainer for offline usage"
)

type SessionManager interface {
	GetConfig() (*LocalConfig, error)
	RequireSession() (*SessionData, error)
	ResolveMaintainer(inputMaintainer string) (string, error)
	SetConfig(config *LocalConfig) error
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
	Config         *tools.GlobalConfig
	OsWrapper      u.OsWrapper
}

type LocalConfig struct {
	Session               *SessionData                    `yaml:"session,omitempty"`
	Signing               *SigningConfig                  `yaml:"signing,omitempty"`
	DockerHub             *tools.DockerHubAuth            `yaml:"docker_hub,omitempty"`
	TrustedMaintainerKeys map[string]TrustedMaintainerKey `yaml:"trusted_maintainer_keys,omitempty"`
}

type SigningConfig struct {
	PublicKeyRawBase64 string `yaml:"public_key_raw_base64,omitempty"`
	PrivateKeyPath     string `yaml:"private_key_path,omitempty"`
}

type TrustedMaintainerKey struct {
	PublicKeyRawBase64       string `yaml:"public_key_raw_base64,omitempty"`
	PublicKeySignatureBase64 string `yaml:"public_key_signature_base64,omitempty"`
}

type SessionData struct {
	Maintainer     string    `yaml:"username"`
	Cookie         string    `yaml:"cookie"`
	ExpirationDate time.Time `yaml:"expiration_date"`
}

func (c *SessionManagerImpl) readLocalConfig() (*LocalConfig, error) {
	raw, err := c.OsWrapper.ReadFile(c.Config.ConfigFilePath)
	if err == nil {
		var config LocalConfig
		if err := yaml.Unmarshal(raw, &config); err != nil {
			return nil, u.Logger.NewError(err.Error())
		}
		return &config, nil
	}
	if !os.IsNotExist(err) {
		return nil, u.Logger.NewError(err.Error())
	}
	return &LocalConfig{}, nil
}

func (c *SessionManagerImpl) SetConfig(config *LocalConfig) error {
	path := c.Config.ConfigFilePath
	if err := c.OsWrapper.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return u.Logger.NewError(err.Error())
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	if err := c.OsWrapper.WriteFile(path, data, 0o600); err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (c *SessionManagerImpl) GetConfig() (*LocalConfig, error) {
	return c.readLocalConfig()
}

func (c *SessionManagerImpl) RequireSession() (*SessionData, error) {
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
		config.TrustedMaintainerKeys = map[string]TrustedMaintainerKey{}
	}
	config.TrustedMaintainerKeys[record.Maintainer] = TrustedMaintainerKey{
		PublicKeyRawBase64:       base64.StdEncoding.EncodeToString(record.PublicKeyRaw),
		PublicKeySignatureBase64: base64.StdEncoding.EncodeToString(record.PublicKeySignature),
	}
	return c.SetConfig(config)
}
