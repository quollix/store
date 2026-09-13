package local

import (
	"qsc/tools"
)

type AppSelector interface {
	SelectApps(selectedApps []string) ([]string, error)
}

type AppSelectorImpl struct {
	FileSystemOperator FileSystemOperator
}

func (a *AppSelectorImpl) SelectApps(selectedApps []string) ([]string, error) {
	appNames, err := a.FileSystemOperator.GetAppNames(tools.AppsDir)
	if err != nil {
		return nil, err
	}
	if len(selectedApps) > 0 {
		appNames = a.filterForSelectedAppNames(selectedApps, appNames)
	}
	return appNames, nil
}

func (a *AppSelectorImpl) filterForSelectedAppNames(selectedApps []string, appNames []string) []string {
	appSet := make(map[string]struct{}, len(selectedApps))
	for _, app := range selectedApps {
		appSet[app] = struct{}{}
	}

	var selectedAppNames []string
	for _, app := range appNames {
		if _, ok := appSet[app]; ok {
			selectedAppNames = append(selectedAppNames, app)
		}
	}
	return selectedAppNames
}
