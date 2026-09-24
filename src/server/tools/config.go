package tools

import (
	"log/slog"
	"os"

	u "github.com/quollix/common/utils"
	"github.com/quollix/deepstack"
)

const (
	OneMegaByteInBytes        int64 = 1024 * 1024
	VersionUploadLimitInBytes int64 = OneMegaByteInBytes
	UserStorageLimitInBytes   int64 = 10 * OneMegaByteInBytes
	AdminStorageLimitInBytes  int64 = 10 * 1024 * OneMegaByteInBytes

	Port       = "8080"
	CookieName = "auth"
)

type Config struct {
	UseMailMockClient       bool
	UseSampleDataForTesting bool
	SeedSampleData          bool
	OpenWipeEndpoint        bool
}

func NewConfig() *Config {
	config := &Config{}
	if os.Getenv("PROFILE") == "TEST" {
		config.UseMailMockClient = true
		config.UseSampleDataForTesting = true
		config.SeedSampleData = os.Getenv("SAMPLE_DATA") == "true"
		config.OpenWipeEndpoint = true
		u.Logger = deepstack.NewDeepStackLogger(deepstack.NewRawConsoleHandler(slog.LevelDebug))
	} else {
		config.UseMailMockClient = false
		config.UseSampleDataForTesting = false
		config.SeedSampleData = false
		config.OpenWipeEndpoint = false
		u.Logger = deepstack.NewDeepStackLogger(deepstack.NewJsonConsoleHandler(slog.LevelInfo))
	}

	return config
}
