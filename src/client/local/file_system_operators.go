package local

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"qsc/tools"

	u "github.com/quollix/common/utils"
	"gopkg.in/yaml.v3"
)

const (
	updateConfigFile                  = "update.yml"
	invalidUpdateConfigEmptyImageName = "invalid update.yml: force_image contains an empty image name"
	invalidUpdateConfigEmptyForcedTag = "invalid update.yml: force_image tag must not be empty"
	invalidUpdateConfigInvalidTag     = "invalid update.yml: force_image tag is invalid"
)

type FileSystemOperator interface {
	GetAppNames(appsDir string) ([]string, error)
	GetForcedImages() (map[string]string, error)
	ParseServicesFromCompose(data []byte) ([]tools.Service, error)

	GetDockerComposeFileContent(composeFilePath string) ([]byte, error)
	WriteDockerComposeFileContent(composeFilePath string, content []byte) error
}

type FileSystemOperatorImpl struct {
	OsWrapper            u.OsWrapper
	TagSelector          TagSelector
	ImageReferenceParser ImageReferenceParser
}

type updateConfig struct {
	ForceImage map[string]string `yaml:"force_image"`
}

func (f *FileSystemOperatorImpl) GetForcedImages() (map[string]string, error) {
	updateConfigPath := getUpdateConfigPath()
	data, err := f.OsWrapper.ReadFile(updateConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			u.Logger.Info("update.yml not found, assuming no image update restrictions", "path", updateConfigPath)
			return map[string]string{}, nil
		}
		return nil, u.Logger.NewError(err.Error(), "path", updateConfigPath)
	}

	var cfg updateConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		// allow file to be empty
		if err == io.EOF {
			return map[string]string{}, nil
		}
		return nil, u.Logger.NewError(err.Error(), "path", updateConfigPath)
	}
	if err := f.validateUpdateConfig(cfg); err != nil {
		return nil, err
	}
	if cfg.ForceImage == nil {
		return map[string]string{}, nil
	}
	return cfg.ForceImage, nil
}

func getUpdateConfigPath() string {
	return filepath.Join(filepath.Dir(filepath.Clean(tools.AppsDir)), updateConfigFile)
}

func (f *FileSystemOperatorImpl) validateUpdateConfig(cfg updateConfig) error {
	updateConfigPath := getUpdateConfigPath()
	for image, tag := range cfg.ForceImage {
		if strings.TrimSpace(image) == "" {
			return u.Logger.NewError(invalidUpdateConfigEmptyImageName, "path", updateConfigPath)
		}
		if strings.TrimSpace(tag) == "" {
			return u.Logger.NewError(invalidUpdateConfigEmptyForcedTag, "path", updateConfigPath, tools.ImageField, image)
		}
		_, core, _ := tools.SplitTag(strings.TrimSpace(tag))
		if _, err := f.TagSelector.Parse(core); err != nil {
			return u.Logger.NewError(invalidUpdateConfigInvalidTag, "path", updateConfigPath, tools.ImageField, image, tools.TagField, tag)
		}
	}
	return nil
}

func (f *FileSystemOperatorImpl) ParseServicesFromCompose(data []byte) ([]tools.Service, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	svcMap, ok := doc["services"].(map[string]interface{})
	if !ok {
		return nil, u.Logger.NewError("'services' field missing in docker-compose.yml")
	}
	var services []tools.Service
	for name, raw := range svcMap {
		svc, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		imageStr, ok := svc["image"].(string)
		if !ok {
			continue
		}
		imageReference := f.ImageReferenceParser.ParseComposeImageReference(imageStr)
		services = append(services, tools.Service{
			Name:   name,
			Image:  imageReference.Image,
			Tag:    imageReference.Tag,
			Digest: imageReference.Digest,
		})
	}
	return services, nil
}

func (f *FileSystemOperatorImpl) GetAppNames(appsDir string) ([]string, error) {
	directoryEntries, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, u.Logger.NewError(err.Error(), tools.AppsDirectoryField, appsDir)
	}
	var appNames []string
	for _, e := range directoryEntries {
		if e.IsDir() || filepath.Ext(e.Name()) != tools.AppFileExtension {
			continue
		}
		appNames = append(appNames, tools.GetAppNameFromComposePath(e.Name()))
	}
	sort.Strings(appNames)
	if len(appNames) == 0 {
		u.Logger.Warn("no app files found in apps directory", tools.AppsDirectoryField, appsDir)
	}
	return appNames, nil
}

func (f *FileSystemOperatorImpl) GetDockerComposeFileContent(composeFilePath string) ([]byte, error) {
	data, err := os.ReadFile(composeFilePath) // #nosec G304 -- compose path is a trusted local workspace path
	if err != nil {
		return nil, u.Logger.NewError(err.Error(), tools.AppDirectoryField, composeFilePath)
	}
	return data, nil
}

func (f *FileSystemOperatorImpl) WriteDockerComposeFileContent(composeFilePath string, content []byte) error {
	err := os.WriteFile(composeFilePath, content, 0o600) // #nosec G703 -- compose path is a trusted local workspace path
	if err != nil {
		return u.Logger.NewError(err.Error(), tools.AppDirectoryField, composeFilePath)
	}
	return nil
}
