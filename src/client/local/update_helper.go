package local

import (
	"sort"

	"qsc/tools"

	u "github.com/quollix/common/utils"
)

type UpdateHelperImpl struct {
	AppSelector      AppSelector
	SingleAppUpdater SingleAppUpdater
}

func (d *UpdateHelperImpl) RunSingleAppUpdates(config UpdateConfig) (*tools.FullUpdateReport, error) {
	apps, err := d.AppSelector.SelectApps(config.SelectedApps)
	if err != nil {
		return nil, err
	}

	report := &tools.FullUpdateReport{
		WasSuccessful: true,
	}
	for _, app := range apps {
		u.Logger.Info("performing update for app", tools.AppField, app)
		appUpdateReport := d.SingleAppUpdater.ConductUpdateForSingleApp(app)
		report.AppUpdateReports = append(report.AppUpdateReports, *appUpdateReport)
	}
	sort.Slice(report.AppUpdateReports, func(i, j int) bool {
		return report.AppUpdateReports[i].AppName < report.AppUpdateReports[j].AppName
	})
	return report, nil
}

func (d *UpdateHelperImpl) UpdateOverallSuccess(report *tools.FullUpdateReport) {
	for _, appReport := range report.AppUpdateReports {
		if !appReport.IsSuccessfulSoFar() {
			report.WasSuccessful = false
			return
		}
	}
}
