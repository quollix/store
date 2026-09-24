package setup

import (
	"context"
	"io"
	"net/http"

	"server/apps"
	"server/maintainers"
	"server/tools"
	"server/versions"

	"github.com/quollix/common/deploy"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

var (
	CookieNotSetInRequest            = "cookie is not set in request, please sign in"
	AccessDeniedForNonAdminUserError = "access denied, user is not admin"
	expectedCookieValidationErrors   = u.MapOf(maintainers.InvalidCookieError, maintainers.CookieExpiredError, maintainers.CookieNotFoundError)
)

type Role int

const (
	AdminRole Role = iota
	UserRole
	AnonymousRole
)

type Route struct {
	path    string
	handler http.HandlerFunc
	role    Role
}

type HandlerInitializer struct {
	AppsHandler     *apps.AppsHandlerImpl
	VersionsHandler *versions.VersionsHandler
	UserHandler     *maintainers.UserHandler
	Config          *tools.Config
	Mux             *http.ServeMux
	UserService     *maintainers.UserServiceImpl
	EmailHandler    *maintainers.EmailHandler
}

func (h *HandlerInitializer) InitializeHandlers() {
	routes := []Route{
		{store.LoginPath, h.UserHandler.LoginHandler, AnonymousRole},
		{store.SetupInitialPasswordPath, h.UserHandler.SetupInitialPasswordHandler, AnonymousRole},
		{store.DownloadPath, h.VersionsHandler.VersionDownloadHandler, AnonymousRole},
		{store.DownloadByIDPath, h.VersionsHandler.VersionDownloadByIDHandler, AnonymousRole},
		{store.DownloadNextVersionForUpdatePath, h.VersionsHandler.DownloadNextVersionForUpdateHandler, AnonymousRole},
		{store.GetVersionsPath, h.VersionsHandler.GetVersionsHandler, AnonymousRole},
		{store.SearchAppsPath, h.AppsHandler.SearchForAppsHandler, AnonymousRole},
		{store.MaintainerPublicKeyPath, h.UserHandler.MaintainerPublicKeyHandler, AnonymousRole},
		{deploy.HealthPath, HealthCheckHandler, AnonymousRole},
		{"/", FrontendPlaceholderHandler, AnonymousRole},

		{store.ChangePasswordPath, h.UserHandler.ChangePasswordHandler, UserRole},
		{store.DeleteUserPath, h.UserHandler.UserDeleteHandler, UserRole},
		{store.LogoutPath, h.UserHandler.LogoutHandler, UserRole},
		{store.AccountDetailsPath, h.UserHandler.GetStatusHandler, UserRole},
		{store.EmailChangePath, h.UserHandler.ChangeEmailHandler, UserRole},
		{store.VersionUploadPath, h.VersionsHandler.VersionsUploadHandler, UserRole},
		{store.VersionDeletePath, h.VersionsHandler.VersionDeleteHandler, UserRole},
		{store.VersionMigrationCheckpointPath, h.VersionsHandler.VersionMigrationCheckpointHandler, UserRole},
		{store.AppCreationPath, h.AppsHandler.AppCreationHandler, UserRole},
		{store.AppGetListPath, h.AppsHandler.AppGetListHandler, UserRole},
		{store.AppDeletePath, h.AppsHandler.AppDeleteHandler, UserRole},

		{store.EmailConfigReadPath, h.EmailHandler.GetEmailHandler, AdminRole},
		{store.EmailConfigWritePath, h.EmailHandler.SetEmailHandler, AdminRole},
		{store.AdminEmailTestPath, h.EmailHandler.SendTestEmailHandler, AdminRole},
		{store.AdminEmailMaintainersPath, h.EmailHandler.SendMaintainerEmailHandler, AdminRole},
		{store.AdminMaintainerCreatePath, h.UserHandler.AdminCreateMaintainerHandler, AdminRole},
		{store.AdminMaintainerListPath, h.UserHandler.AdminListMaintainersHandler, AdminRole},
		{store.AdminMaintainerDeletePath, h.UserHandler.AdminDeleteMaintainerHandler, AdminRole},
		{store.AdminMaintainerSetSpacePath, h.UserHandler.AdminSetMaintainerStorageLimitHandler, AdminRole},
	}

	if h.Config.OpenWipeEndpoint {
		u.Logger.Warn("opening unprotected full data wipe endpoint meant for testing only")
		routes = append(routes, Route{store.WipeDataPath, h.UserHandler.WipeData, AnonymousRole})
	}

	h.registerRoutes(routes)
}

func (h *HandlerInitializer) registerRoutes(routes []Route) {
	for _, r := range routes {
		h.Mux.Handle(r.path, h.authMiddleware(r.handler, r.role))
	}
}

func (h *HandlerInitializer) authMiddleware(next http.Handler, role Role) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if role == AnonymousRole {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(tools.CookieName)
		if err != nil {
			u.Logger.Debug(CookieNotSetInRequest, "remote_addr", r.RemoteAddr, "path", r.URL.Path)
			http.Error(w, CookieNotSetInRequest, http.StatusBadRequest)
			return
		}
		user, err := h.UserService.CheckAuthenticationAndUpdateCookie(cookie)
		if err != nil {
			u.WriteResponseError(w, expectedCookieValidationErrors, err)
			return
		}
		http.SetCookie(w, cookie)

		if role == AdminRole && !user.IsAdmin {
			u.Logger.Info(AccessDeniedForNonAdminUserError, tools.UserField, user.Name)
			http.Error(w, AccessDeniedForNonAdminUserError, http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), tools.UserCtxKey, user)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

type healthInfo struct {
	Status string `json:"status"`
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	u.SendJsonResponse(w, healthInfo{Status: "ok"})
}

const defaultFrontendContent = `
<h1>Quollix App Store</h1>
<p>You cannot interact with this server via a web interface.</p>
<p>Please visit the 
	<a href="https://quollix.org/docs/project/app-store/" target="_blank">usage tutorial on our website</a>.
</p>
`

var PlaceHolderFrontend = u.RenderDefaultPage(defaultFrontendContent)

func FrontendPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := io.WriteString(w, PlaceHolderFrontend)
	if err != nil {
		u.Logger.Error(err.Error())
		return
	}
}
