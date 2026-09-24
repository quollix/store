package configuration

import (
	"os"
	"path/filepath"
	"time"

	"qsc/tools"

	u "github.com/quollix/common/utils"
	"gopkg.in/yaml.v3"
)

const MissingAppsDirectoryError = "apps directory is not configured"

type Config struct {
	AppsDirectory         string                          `yaml:"apps_directory,omitempty"`
	Session               *SessionData                    `yaml:"session,omitempty"`
	Signing               *SigningConfig                  `yaml:"signing,omitempty"`
	DockerHub             *tools.DockerHubAuth            `yaml:"docker_hub,omitempty"`
	TrustedMaintainerKeys map[string]TrustedMaintainerKey `yaml:"trusted_maintainer_keys,omitempty"`
}

func (c *Config) RequireAppsDirectory() (string, error) {
	if c.AppsDirectory == "" {
		return "", u.Logger.NewError(MissingAppsDirectoryError)
	}
	return c.AppsDirectory, nil
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

type Provider interface {
	GetConfig() (*Config, error)
	SetConfig(config *Config) error
}

type ProviderImpl struct {
	Config    *tools.GlobalConfig
	OsWrapper u.OsWrapper
}

func (p *ProviderImpl) GetConfig() (*Config, error) {
	raw, err := p.OsWrapper.ReadFile(p.Config.ConfigFilePath)
	if err == nil {
		var config Config
		if err := yaml.Unmarshal(raw, &config); err != nil {
			return nil, u.Logger.NewError(err.Error())
		}
		return &config, nil
	}
	if !os.IsNotExist(err) {
		return nil, u.Logger.NewError(err.Error())
	}
	return &Config{}, nil
}

func (p *ProviderImpl) SetConfig(config *Config) error {
	path := p.Config.ConfigFilePath
	if err := p.OsWrapper.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return u.Logger.NewError(err.Error())
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	if err := p.OsWrapper.WriteFile(path, data, 0o600); err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}
