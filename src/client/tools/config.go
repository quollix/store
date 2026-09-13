package tools

import (
	"path/filepath"
)

type GlobalConfig struct {
	AppStoreRootURL                            string
	ConfigFilePath                             string
	ImageTagCachePath                          string
	VerifyAppStoreCertificate                  bool
	EnableImageTagCache                        bool
	UseTestRegistryTagFetcher                  bool
	UseLocalTestingOfficialMaintainerPublicKey bool
}

func InitGlobalConfig(profileEnv string, userConfigDir string) *GlobalConfig {
	if profileEnv == "TEST" {
		return &GlobalConfig{
			AppStoreRootURL:                            "http://localhost:8080",
			ConfigFilePath:                             filepath.Join(userConfigDir, "qsc-test", "config.yml"),
			ImageTagCachePath:                          filepath.Join(userConfigDir, "qsc-test", "image-tag-cache.yml"),
			VerifyAppStoreCertificate:                  false,
			EnableImageTagCache:                        false,
			UseTestRegistryTagFetcher:                  true,
			UseLocalTestingOfficialMaintainerPublicKey: true,
		}
	}
	return &GlobalConfig{
		AppStoreRootURL:                            "https://store.quollix.org",
		ConfigFilePath:                             filepath.Join(userConfigDir, "qsc", "config.yml"),
		ImageTagCachePath:                          filepath.Join(userConfigDir, "qsc", "image-tag-cache.yml"),
		VerifyAppStoreCertificate:                  true,
		EnableImageTagCache:                        true,
		UseTestRegistryTagFetcher:                  false,
		UseLocalTestingOfficialMaintainerPublicKey: false,
	}
}
