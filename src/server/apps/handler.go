package apps

import (
	"net/http"
	"server/maintainers"
	"server/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

var expectedAppCreationErrors = u.MapOf(AppNameReservedError, AppAlreadyExistsError, MaximumNumberOfAppsReachedError)

type AppsHandlerImpl struct {
	AppRepo    AppRepository
	UserRepo   maintainers.UserRepository
	AppService *AppServiceImpl
}

func (a *AppsHandlerImpl) AppCreationHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	appString, ok := validation.ReadBody[store.AppNameString](w, r)
	if !ok {
		return
	}
	err := a.AppService.CreateApp(user, appString.Value)
	if err != nil {
		u.WriteResponseError(w, expectedAppCreationErrors, err)
		return
	}
}

func (a *AppsHandlerImpl) AppDeleteHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	appName, ok := validation.ReadBody[store.AppNameString](w, r)
	if !ok {
		return
	}
	err := a.AppService.DeleteAppWithChecks(user, appName.Value)
	if err != nil {
		u.WriteResponseError(w, tools.DoesNotExistErrorMap, err)
		return
	}
}

func (a *AppsHandlerImpl) AppGetListHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	appList, err := a.AppRepo.GetAppList(user.Id)
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
	u.SendJsonResponse(w, appList)
}

func (a *AppsHandlerImpl) SearchForAppsHandler(w http.ResponseWriter, r *http.Request) {
	appSearchRequest, ok := validation.ReadBody[store.SearchRequest](w, r)
	if !ok {
		return
	}
	apps, err := a.AppRepo.SearchForApps(*appSearchRequest)
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
	u.SendJsonResponse(w, apps)
}
