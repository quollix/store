package apps

import (
	"server/maintainers"
	"server/tools"

	u "github.com/quollix/common/utils"
)

var (
	AppNameReservedError            = "app name is reserved"
	AppAlreadyExistsError           = "app already exists"
	MaximumNumberOfAppsReachedError = "you have reached the maximum number of apps"
)

type AppServiceImpl struct {
	AppRepo  AppRepository
	UserRepo maintainers.UserRepository
}

func (a *AppServiceImpl) CreateApp(user *tools.User, appName string) error {
	if u.IsSystemApp(appName) {
		return u.Logger.NewError(AppNameReservedError)
	}

	numberOfExistingApps, err := a.AppRepo.GetNumberOfApps(user.Id)
	if err != nil {
		return err
	}
	if numberOfExistingApps >= 100 {
		return u.Logger.NewError(MaximumNumberOfAppsReachedError)
	}

	doesExist, err := a.AppRepo.DoesAppExistByMaintainerID(user.Id, appName)
	if err != nil {
		return err
	}
	if doesExist {
		return u.Logger.NewError(AppAlreadyExistsError)
	}

	err = a.AppRepo.CreateApp(user.Id, appName)
	if err != nil {
		return err
	}
	return nil
}

func (a *AppServiceImpl) DeleteAppWithChecks(user *tools.User, appName string) error {
	app, err := a.AppRepo.GetAppByName(user.Id, appName)
	if err != nil {
		return err
	}

	numberOfBytesToBeFreedUpAfterDeletion, err := a.AppRepo.SumUpBytesOfAllAppVersions(app.AppId)
	if err != nil {
		return err
	}

	user, err = a.UserRepo.GetUserById(app.MaintainerId)
	if err != nil {
		return err
	}

	err = a.AppRepo.DeleteApp(user.Id, appName)
	if err != nil {
		return err
	}

	user.UsedSpaceInBytes -= numberOfBytesToBeFreedUpAfterDeletion
	err = a.UserRepo.UpdateUser(user)
	if err != nil {
		return err
	}

	return nil
}

func (a *AppServiceImpl) AssertAppIsPresent(userName, appName string) error {
	exists, err := a.AppRepo.DoesAppExistByMaintainerName(userName, appName)
	if err != nil {
		return err
	}
	if !exists {
		return u.Logger.NewError(tools.DoesNotExistError)
	}
	return nil
}
