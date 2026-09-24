package local

import (
	"qsc/configuration"
	"qsc/tools"

	u "github.com/quollix/common/utils"
)

type SingleAppUpdater interface {
	ConductUpdateForSingleApp(app string) *tools.AppUpdateReport
}

type SingleAppUpdaterImpl struct {
	FileSystemOperator    FileSystemOperator
	AppUpdater            UpdateFetcher
	ComposeContentUpdater ComposeContentUpdater
	ConfigProvider        configuration.Provider
}

func (d *SingleAppUpdaterImpl) ConductUpdateForSingleApp(app string) *tools.AppUpdateReport {
	report := getEmptyAppUpdateReport(app)
	updateReport, err := d.conductUpdateWithErrorHandling(report)
	if err != nil {
		err = u.Logger.AddContext(err, "app_name", app)
		u.Logger.Error(err, "result", "app update failed")
		if updateReport.ErrorMessage == "" {
			updateReport.ErrorMessage = u.ExtractError(err)
		}
		return updateReport
	}
	return report
}

func getEmptyAppUpdateReport(app string) *tools.AppUpdateReport {
	return &tools.AppUpdateReport{
		AppName:        app,
		ServiceUpdates: nil,
		ErrorMessage:   "",
	}
}

func (d *SingleAppUpdaterImpl) conductUpdateWithErrorHandling(report *tools.AppUpdateReport) (*tools.AppUpdateReport, error) {
	config, err := d.ConfigProvider.GetConfig()
	if err != nil {
		return report, err
	}
	appsDirectory, err := config.RequireAppsDirectory()
	if err != nil {
		return report, err
	}
	sourceComposePath := tools.GetAppComposePath(appsDirectory, report.AppName)
	u.Logger.Info("reading compose file", tools.AppField, report.AppName, tools.ComposeFilePathField, sourceComposePath)
	composeContent, err := d.FileSystemOperator.GetDockerComposeFileContent(sourceComposePath)
	if err != nil {
		return report, err
	}

	u.Logger.Info("fetching app updates", tools.AppField, report.AppName)
	if err = d.AppUpdater.FetchUpdateAndWriteToReport(report, composeContent); err != nil {
		return report, err
	}
	u.Logger.Info("app updates fetched", tools.AppField, report.AppName)
	if !report.HasUpdates() {
		u.Logger.Info("no app updates found", tools.AppField, report.AppName)
		return report, nil
	}
	u.Logger.Info("applying service updates to compose content", tools.AppField, report.AppName, tools.ComposeFilePathField, sourceComposePath)
	updatedComposeContent, err := d.ComposeContentUpdater.UpdateComposeTags(composeContent, report.ServiceUpdates)
	if err != nil {
		return report, err
	}
	u.Logger.Info("service updates applied to compose content", tools.AppField, report.AppName, tools.ComposeFilePathField, sourceComposePath)
	u.Logger.Info("writing updated compose file", tools.AppField, report.AppName, tools.ComposeFilePathField, sourceComposePath)
	if err = d.FileSystemOperator.WriteDockerComposeFileContent(sourceComposePath, updatedComposeContent); err != nil {
		return report, err
	}
	u.Logger.Info("updated compose file written", tools.AppField, report.AppName, tools.ComposeFilePathField, sourceComposePath)
	return report, nil
}
