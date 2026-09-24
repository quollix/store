package local

import (
	"qsc/configuration"
	"qsc/tools"

	"github.com/quollix/common/validation"
)

type ConsistencyChecker interface {
	PerformConsistencyChecks(selectedApps []string) (bool, error)
}

type ConsistencyCheckerImpl struct {
	AppSelector        AppSelector
	ConfigProvider     configuration.Provider
	FileSystemOperator FileSystemOperator
	ComposeValidator   validation.VersionValidator
}

func (c *ConsistencyCheckerImpl) PerformConsistencyChecks(selectedApps []string) (bool, error) {
	apps, err := c.AppSelector.SelectApps(selectedApps)
	if err != nil {
		return false, err
	}
	if len(apps) == 0 {
		return false, nil
	}
	config, err := c.ConfigProvider.GetConfig()
	if err != nil {
		return false, err
	}
	appsDirectory, err := config.RequireAppsDirectory()
	if err != nil {
		return false, err
	}

	for _, app := range apps {
		appPath := tools.GetAppComposePath(appsDirectory, app)
		content, err := c.FileSystemOperator.GetDockerComposeFileContent(appPath)
		if err != nil {
			return true, err
		}
		if err := c.ComposeValidator.Validate(content, tools.OfficialMaintainer, app); err != nil {
			return true, err
		}
	}
	return true, nil
}
