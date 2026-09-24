package local

import (
	"testing"

	"qsc/configuration"

	"github.com/quollix/common/assert"
)

const testAppsDirectory = "apps"

func setupAppSelectorTest(t *testing.T) (*AppSelectorImpl, *FileSystemOperatorMock) {
	fileSystemOperatorMock := NewFileSystemOperatorMock(t)
	configProviderMock := configuration.NewProviderMock(t)
	configProviderMock.EXPECT().GetConfig().Return(&configuration.Config{AppsDirectory: testAppsDirectory}, nil)
	return &AppSelectorImpl{FileSystemOperator: fileSystemOperatorMock, ConfigProvider: configProviderMock}, fileSystemOperatorMock
}

func TestAppSelectorImpl_SelectApps_ReturnsAllAppsWhenNoSelectionWasPassed(t *testing.T) {
	appSelector, fileSystemOperatorMock := setupAppSelectorTest(t)
	fileSystemOperatorMock.EXPECT().GetAppNames(testAppsDirectory).Return([]string{"app1", "app2"}, nil)

	selectedApps, err := appSelector.SelectApps(nil)

	assert.Nil(t, err)
	assert.Equal(t, []string{"app1", "app2"}, selectedApps)
}

func TestAppSelectorImpl_SelectApps_ReturnsOnlySelectedAppsInDirectoryOrder(t *testing.T) {
	appSelector, fileSystemOperatorMock := setupAppSelectorTest(t)
	fileSystemOperatorMock.EXPECT().GetAppNames(testAppsDirectory).Return([]string{"app1", "app2", "app3"}, nil)

	selectedApps, err := appSelector.SelectApps([]string{"app3", "app1"})

	assert.Nil(t, err)
	assert.Equal(t, []string{"app1", "app3"}, selectedApps)
}

func TestAppSelectorImpl_SelectApps_ReturnsEmptySliceWhenNoAppsExist(t *testing.T) {
	appSelector, fileSystemOperatorMock := setupAppSelectorTest(t)
	fileSystemOperatorMock.EXPECT().GetAppNames(testAppsDirectory).Return([]string{}, nil)

	selectedApps, err := appSelector.SelectApps(nil)

	assert.Nil(t, err)
	assert.Equal(t, []string{}, selectedApps)
}
