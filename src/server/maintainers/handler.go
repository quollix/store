package maintainers

import (
	"net/http"
	"server/tools"

	"github.com/quollix/common/store"

	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

var (
	expectedIncorrectUsernameOrPasswordErrors = u.MapOf(IncorrectUsernameOrPasswordError)
	expectedSetupInitialPasswordErrors        = u.MapOf(RegistrationCodeNotFoundError, RegistrationCodeExpiredError)
	expectedAdminMaintainerErrors             = u.MapOf(MaintainerAlreadyExistsError, EmailAlreadyExistsError, PublicKeyAlreadyExistsError, InvalidPublicKeyError, InvalidPublicKeySignatureError, MaintainerNotFoundError, AdminMaintainerDeleteError)
	expectedEmailAlreadyExistsErrors          = u.MapOf(EmailAlreadyExistsError)
)

type UserHandler struct {
	UserRepo                   UserRepository
	Config                     *tools.Config
	UserService                *UserServiceImpl
	DatabaseSnapshotRepository u.DatabaseSnapshotRepository
}

func (h *UserHandler) WipeData(w http.ResponseWriter, r *http.Request) {
	if err := h.DatabaseSnapshotRepository.ResetDatabaseToSnapshot(); err != nil {
		u.WriteResponseError(w, nil, err)
	}
}

func (h *UserHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	creds, ok := validation.ReadBody[store.LoginCredentials](w, r)
	if !ok {
		return
	}
	cookie, err := h.UserService.Login(creds)
	if err != nil {
		u.WriteResponseError(w, expectedIncorrectUsernameOrPasswordErrors, err)
		return
	}
	http.SetCookie(w, cookie)
}

func (h *UserHandler) UserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	err := h.UserRepo.DeleteUser(user.Id)
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
}

func (h *UserHandler) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	form, ok := validation.ReadBody[store.ChangePasswordForm](w, r)
	if !ok {
		return
	}
	err := h.UserService.ChangePassword(user, form)
	if err != nil {
		u.WriteResponseError(w, expectedIncorrectUsernameOrPasswordErrors, err)
		return
	}
}

func (h *UserHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	err := h.UserRepo.Logout(user.Id)
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
}

func (h *UserHandler) SetupInitialPasswordHandler(w http.ResponseWriter, r *http.Request) {
	form, ok := validation.ReadBody[store.SetupInitialPasswordForm](w, r)
	if !ok {
		return
	}
	err := h.UserService.SetupInitialPassword(form)
	if err != nil {
		u.WriteResponseError(w, expectedSetupInitialPasswordErrors, err)
		return
	}
}

func (h *UserHandler) AdminCreateMaintainerHandler(w http.ResponseWriter, r *http.Request) {
	admin, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	form, ok := validation.ReadBody[store.AdminMaintainerCreateForm](w, r)
	if !ok {
		return
	}
	err := h.UserService.CreateMaintainerByAdmin(form, admin)
	if err != nil {
		u.WriteResponseError(w, expectedAdminMaintainerErrors, err)
		return
	}
}

func (h *UserHandler) MaintainerPublicKeyHandler(w http.ResponseWriter, r *http.Request) {
	form, ok := validation.ReadBody[store.MaintainerNameString](w, r)
	if !ok {
		return
	}
	record, err := h.UserRepo.GetMaintainerPublicKeyRecord(form.Value)
	if err != nil {
		u.WriteResponseError(w, u.MapOf(MaintainerNotFoundError, MaintainerPublicKeySignatureNotFoundError), err)
		return
	}
	u.SendJsonResponse(w, record)
}

func (h *UserHandler) AdminDeleteMaintainerHandler(w http.ResponseWriter, r *http.Request) {
	form, ok := validation.ReadBody[store.MaintainerNameString](w, r)
	if !ok {
		return
	}
	err := h.UserService.DeleteMaintainerByAdmin(form.Value)
	if err != nil {
		u.WriteResponseError(w, expectedAdminMaintainerErrors, err)
		return
	}
}

func (h *UserHandler) GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	accountDetails := store.AccountDetails{
		Name:                 user.Name,
		Email:                user.Email,
		PublicKeyRaw:         user.PublicKeyRaw,
		CookieExpirationDate: *user.ExpirationDate,
		UsedSpaceInBytes:     user.UsedSpaceInBytes,
		StorageLimitInBytes:  user.StorageLimitInBytes,
		IsAdmin:              user.IsAdmin,
	}
	u.SendJsonResponse(w, accountDetails)
}

func (h *UserHandler) ChangeEmailHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	emailString, ok := validation.ReadBody[store.EmailString](w, r)
	if !ok {
		return
	}
	err := h.UserService.ChangeEmail(user.Id, emailString.Value)
	if err != nil {
		u.WriteResponseError(w, expectedEmailAlreadyExistsErrors, err)
		return
	}
}
