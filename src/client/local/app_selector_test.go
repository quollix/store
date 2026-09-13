package local

import (
	"testing"

	"qsc/tools"

	"github.com/quollix/common/assert"
)

func setupAppSelectorTest(t *testing.T) (*AppSelectorImpl, *FileSystemOperatorMock) {
	fileSystemOperatorMock := NewFileSystemOperatorMock(t)
	return &AppSelectorImpl{FileSystemOperator: fileSystemOperatorMock}, fileSystemOperatorMock
}

func TestAppSelectorImpl_SelectApps_ReturnsAllAppsWhenNoSelectionWasPassed(t *testing.T) {
	appSelector, fileSystemOperatorMock := setupAppSelectorTest(t)
	fileSystemOperatorMock.EXPECT().GetAppNames(tools.AppsDir).Return([]string{"app1", "app2"}, nil)

	selectedApps, err := appSelector.SelectApps(nil)

	assert.Nil(t, err)
	assert.Equal(t, []string{"app1", "app2"}, selectedApps)
}

func TestAppSelectorImpl_SelectApps_ReturnsOnlySelectedAppsInDirectoryOrder(t *testing.T) {
	appSelector, fileSystemOperatorMock := setupAppSelectorTest(t)
	fileSystemOperatorMock.EXPECT().GetAppNames(tools.AppsDir).Return([]string{"app1", "app2", "app3"}, nil)

	selectedApps, err := appSelector.SelectApps([]string{"app3", "app1"})

	assert.Nil(t, err)
	assert.Equal(t, []string{"app1", "app3"}, selectedApps)
}

func TestAppSelectorImpl_SelectApps_ReturnsEmptySliceWhenNoAppsExist(t *testing.T) {
	appSelector, fileSystemOperatorMock := setupAppSelectorTest(t)
	fileSystemOperatorMock.EXPECT().GetAppNames(tools.AppsDir).Return([]string{}, nil)

	selectedApps, err := appSelector.SelectApps(nil)

	assert.Nil(t, err)
	assert.Equal(t, []string{}, selectedApps)
}
