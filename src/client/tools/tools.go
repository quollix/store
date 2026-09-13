package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/quollix/common/assert"
	"gopkg.in/yaml.v3"
)

var (
	AppsDir            = "apps"
	OfficialMaintainer = "quollix"
	SampleMaintainer   = "sample"
	SampleAppName      = "sampleapp"
	SampleVersion      = "0.0.1"
	SampleEmail        = "sample@sample.com"
	SamplePassword     = "password"
)

const (
	AppFileExtension         = ".yml"
	ImageField               = "image"
	TagField                 = "tag"
	StatusCodeField          = "status_code"
	UrlField                 = "url"
	AppsDirectoryField       = "apps_directory"
	AppDirectoryField        = "app_directory"
	AppField                 = "app"
	ReportErrorMessageField  = "report_error_message"
	ServiceNameField         = "service_name"
	OldServiceTagField       = "service_tag"
	NewerServiceTagField     = "newer_service_tag"
	MainServiceField         = "main_service"
	WasSuccessulField        = "was_successful"
	ComposeFilePathField     = "compose_file_path"
	MainServiceNotFoundError = "main service not found"
)

func GetAppComposePath(appsDir, appName string) string {
	return filepath.Join(appsDir, appName+AppFileExtension)
}

func GetAppNameFromComposePath(path string) string {
	base := filepath.Base(filepath.Clean(path))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func EqualYaml(t *testing.T, a, b []byte) {
	var v1, v2 interface{}
	assert.Nil(t, yaml.Unmarshal(a, &v1))
	assert.Nil(t, yaml.Unmarshal(b, &v2))
	assert.Equal(t, v2, v1)
}

func SplitTag(tag string) (string, string, string) {
	prefix := ""
	if strings.HasPrefix(tag, "v") {
		prefix = "v"
		tag = strings.TrimPrefix(tag, "v")
	} else if strings.HasPrefix(tag, "stable-") {
		prefix = "stable-"
		tag = strings.TrimPrefix(tag, prefix)
	}
	suffix := ""
	if idx := strings.Index(tag, "-"); idx != -1 {
		suffix = tag[idx:]
		tag = tag[:idx]
	}
	return prefix, tag, suffix
}
