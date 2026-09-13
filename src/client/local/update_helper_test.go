package local

import (
	"testing"

	"qsc/tools"

	"github.com/quollix/common/assert"
)

type updateHelperTestDependencies struct {
	updateHelper     *UpdateHelperImpl
	appSelector      *AppSelectorMock
	singleAppUpdater *SingleAppUpdaterMock
}

func setupUpdateHelperTest(t *testing.T) *updateHelperTestDependencies {
	appSelectorMock := NewAppSelectorMock(t)
	singleAppUpdaterMock := NewSingleAppUpdaterMock(t)
	return &updateHelperTestDependencies{
		updateHelper: &UpdateHelperImpl{
			AppSelector:      appSelectorMock,
			SingleAppUpdater: singleAppUpdaterMock,
		},
		appSelector:      appSelectorMock,
		singleAppUpdater: singleAppUpdaterMock,
	}
}

func TestUpdateHelperImpl_RunSingleAppUpdates_RunsAppsSequentially(t *testing.T) {
	deps := setupUpdateHelperTest(t)
	deps.appSelector.EXPECT().SelectApps([]string{"app1", "app2"}).Return([]string{"app1", "app2"}, nil)
	var processedApps []string
	deps.singleAppUpdater.EXPECT().ConductUpdateForSingleApp("app1").RunAndReturn(func(app string) *tools.AppUpdateReport {
		processedApps = append(processedApps, app)
		return &tools.AppUpdateReport{AppName: app}
	})
	deps.singleAppUpdater.EXPECT().ConductUpdateForSingleApp("app2").RunAndReturn(func(app string) *tools.AppUpdateReport {
		processedApps = append(processedApps, app)
		return &tools.AppUpdateReport{AppName: app}
	})

	report, err := deps.updateHelper.RunSingleAppUpdates(UpdateConfig{SelectedApps: []string{"app1", "app2"}})

	assert.Nil(t, err)
	assert.True(t, report.WasSuccessful)
	assert.Equal(t, []string{"app1", "app2"}, processedApps)
}

func TestUpdateHelperImpl_RunSingleAppUpdates_SortsReportsByAppName(t *testing.T) {
	deps := setupUpdateHelperTest(t)
	deps.appSelector.EXPECT().SelectApps([]string(nil)).Return([]string{"z-app", "a-app"}, nil)
	deps.singleAppUpdater.EXPECT().ConductUpdateForSingleApp("z-app").Return(&tools.AppUpdateReport{AppName: "z-app"})
	deps.singleAppUpdater.EXPECT().ConductUpdateForSingleApp("a-app").Return(&tools.AppUpdateReport{AppName: "a-app"})

	report, err := deps.updateHelper.RunSingleAppUpdates(UpdateConfig{})

	assert.Nil(t, err)
	assert.Equal(t, []string{"a-app", "z-app"}, []string{report.AppUpdateReports[0].AppName, report.AppUpdateReports[1].AppName})
}

func TestUpdateHelperImpl_UpdateOverallSuccess_SetsFailureWhenAnyAppFailed(t *testing.T) {
	updateHelper := &UpdateHelperImpl{}
	report := &tools.FullUpdateReport{
		WasSuccessful: true,
		AppUpdateReports: []tools.AppUpdateReport{
			{AppName: "app1"},
			{AppName: "app2", ErrorMessage: "boom"},
		},
	}

	updateHelper.UpdateOverallSuccess(report)

	assert.False(t, report.WasSuccessful)
}
