package local

import (
	"qsc/tools"
)

type Updater struct {
	UpdateHelper  UpdateHelper
	ImageTagCache ImageTagCache
}

type UpdateHelper interface {
	RunSingleAppUpdates(config UpdateConfig) (*tools.FullUpdateReport, error)
	UpdateOverallSuccess(report *tools.FullUpdateReport)
}

type UpdateConfig struct {
	SelectedApps []string
}

func (d *Updater) PerformUpdate(config UpdateConfig) (*tools.FullUpdateReport, error) {
	d.ImageTagCache.Load()
	defer d.ImageTagCache.Save()

	report, err := d.UpdateHelper.RunSingleAppUpdates(config)
	if err != nil {
		return nil, err
	}

	d.UpdateHelper.UpdateOverallSuccess(report)
	return report, nil
}
