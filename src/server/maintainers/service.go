package maintainers

import (
	"crypto/ed25519"
	"net/http"
	"server/tools"
	"time"

	"github.com/quollix/common/bootstrap"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

var (
	IncorrectUsernameOrPasswordError          = "incorrect username or password"
	MaintainerAlreadyExistsError              = "maintainer already exists"
	EmailAlreadyExistsError                   = "email already exists"
	PublicKeyAlreadyExistsError               = "public key already exists"
	InvalidCookieError                        = "invalid cookie, please try to sign in again"
	CookieExpiredError                        = "cookie expired, please try to sign in again"
	CookieNotFoundError                       = "cookie not found, please try to sign in again"
	MaximumStorageExceededError               = "maximum storage exceeded"
	InvalidPublicKeyError                     = "invalid ed25519 public key"
	InvalidPublicKeySignatureError            = "invalid maintainer public key signature"
	MaintainerPublicKeySignatureNotFoundError = "maintainer public key signature not found"
	MaintainerNotFoundError                   = "maintainer not found"
	AdminMaintainerDeleteError                = "admin maintainer cannot be deleted"
	StorageLimitMustBeNonNegativeError        = "storage limit must not be negative"
	RegistrationCodeNotFoundError             = "registration code not found"
	RegistrationCodeExpiredError              = "registration code expired"
)

const (
	QuollixAdminUsername            = "quollix"
	QuollixAdminEmail               = "admin@quollix.org"
	QuollixAdminPassword            = "password"
	AuthCookieValidDays             = 30
	AccountSetupTokenValidityInDays = 7
)

type UserServiceImpl struct {
	UserRepo        UserRepository
	Config          *tools.Config
	SecretGenerator SecretGenerator
	AuthHelper      u.AuthHelper
	EmailService    EmailService
}

const MaintainerSetupEmailSubject = "Quollix App Store account setup"

func applyAuthCookieSecurity(cookie *http.Cookie) {
	cookie.Name = tools.CookieName
	cookie.Path = "/"
	cookie.SameSite = http.SameSiteStrictMode
	cookie.HttpOnly = true
	cookie.Secure = true
}

func getAuthCookieExpiration() time.Time {
	return time.Now().UTC().Add(AuthCookieValidDays * 24 * time.Hour)
}

func (s *UserServiceImpl) isPasswordCorrect(userName string, password string) (bool, error) {
	user, err := s.UserRepo.GetUserByName(userName)
	if err != nil {
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	return err == nil, nil
}

func (s *UserServiceImpl) saveCookie(userName, cookie string, expirationDate time.Time) error {
	user, err := s.UserRepo.GetUserByName(userName)
	if err != nil {
		return err
	}

	hashedCookieValue := s.AuthHelper.GetSHA256Hash(cookie)
	user.HashedCookieValue = &hashedCookieValue
	user.ExpirationDate = &expirationDate
	return s.UserRepo.UpdateUser(user)
}

func (s *UserServiceImpl) isCookieExpired(cookie string) (bool, error) {
	hashedCookieValue := s.AuthHelper.GetSHA256Hash(cookie)
	user, err := s.UserRepo.GetUserViaHashedCookie(hashedCookieValue)
	if err != nil {
		return true, err
	}
	return time.Now().UTC().After(*user.ExpirationDate), nil
}

func (s *UserServiceImpl) Login(creds *store.LoginCredentials) (*http.Cookie, error) {
	isCorrect, err := s.isPasswordCorrect(creds.User, creds.Password)
	if err != nil {
		return nil, err
	}

	if !isCorrect {
		return nil, u.Logger.NewError(IncorrectUsernameOrPasswordError)
	}

	cookie, err := s.AuthHelper.GenerateCookie()
	if err != nil {
		return nil, err
	}
	cookie.Expires = getAuthCookieExpiration()
	applyAuthCookieSecurity(cookie)

	err = s.saveCookie(creds.User, cookie.Value, cookie.Expires)
	if err != nil {
		return nil, err
	}
	return cookie, nil
}

func (s *UserServiceImpl) CreateMaintainerByAdmin(form *store.AdminMaintainerCreateForm, admin *tools.User) error {
	if err := s.validateMaintainerDoesNotExist(form.Name, form.Email, form.PublicKeyRaw); err != nil {
		return err
	}
	if err := validateMaintainerPublicKeySignature(form, admin.PublicKeyRaw); err != nil {
		return err
	}

	setupToken, err := s.SecretGenerator.GenerateAccountRegistrationCode()
	if err != nil {
		return err
	}
	expirationDate := time.Now().UTC().AddDate(0, 0, AccountSetupTokenValidityInDays)
	hashedSetupToken := s.AuthHelper.GetSHA256Hash(setupToken)
	if err = s.UserRepo.CreateUserWithSetupToken(form.Name, form.Email, form.PublicKeyRaw, form.PublicKeySignature, tools.UserStorageLimitInBytes, hashedSetupToken, expirationDate); err != nil {
		return err
	}
	if err = s.EmailService.SendEmail(form.Email, MaintainerSetupEmailSubject, formatMaintainerSetupEmailBody(form.Name, setupToken, expirationDate)); err != nil {
		if rollbackErr := s.deleteCreatedMaintainerAfterEmailFailure(form.Name); rollbackErr != nil {
			return u.Logger.AddContext(err, "rollback_error", u.ExtractError(rollbackErr))
		}
		return err
	}
	return nil
}

func formatMaintainerSetupEmailBody(name string, setupToken string, expirationDate time.Time) string {
	return "Hello " + name + ",\n\n" +
		"your Quollix App Store maintainer account has been created.\n\n" +
		"Setup token: " + setupToken + "\n" +
		"This token expires at: " + expirationDate.UTC().Format("2006-01-02 15:04:05") + " UTC\n\n" +
		"Next steps:\n" +
		"https://quollix.org/docs/project/app-store/join/\n"
}

func (s *UserServiceImpl) deleteCreatedMaintainerAfterEmailFailure(name string) error {
	user, err := s.UserRepo.GetUserByName(name)
	if err != nil {
		return err
	}
	return s.UserRepo.DeleteUser(user.Id)
}

func (s *UserServiceImpl) validateMaintainerDoesNotExist(name string, email string, publicKeyRaw []byte) error {
	if err := validateEd25519PublicKeyRaw(publicKeyRaw); err != nil {
		return err
	}

	doesUserExist, err := s.UserRepo.DoesUserExist(name)
	if err != nil {
		return err
	}
	if doesUserExist {
		return u.Logger.NewError(MaintainerAlreadyExistsError)
	}

	doesEmailExist, err := s.UserRepo.DoesEmailExist(email)
	if err != nil {
		return err
	}
	if doesEmailExist {
		return u.Logger.NewError(EmailAlreadyExistsError)
	}

	doesPublicKeyExist, err := s.UserRepo.DoesPublicKeyRawExist(publicKeyRaw)
	if err != nil {
		return err
	}
	if doesPublicKeyExist {
		return u.Logger.NewError(PublicKeyAlreadyExistsError)
	}
	return nil
}

func validateMaintainerPublicKeySignature(form *store.AdminMaintainerCreateForm, adminPublicKeyRaw []byte) error {
	ok, err := store.VerifyMaintainerPublicKeySignature(ed25519.PublicKey(adminPublicKeyRaw), &store.MaintainerPublicKeyRecord{
		Maintainer:         form.Name,
		PublicKeyRaw:       form.PublicKeyRaw,
		PublicKeySignature: form.PublicKeySignature,
	})
	if err != nil {
		return u.Logger.NewError(InvalidPublicKeySignatureError)
	}
	if !ok {
		return u.Logger.NewError(InvalidPublicKeySignatureError)
	}
	return nil
}

func (s *UserServiceImpl) SetupInitialPassword(form *store.SetupInitialPasswordForm) error {
	hashedCode := s.AuthHelper.GetSHA256Hash(form.Token)
	user, err := s.UserRepo.GetUserBySetupTokenHash(hashedCode)
	if err != nil {
		return err
	}
	if user.SetupTokenExpiresAt == nil || time.Now().UTC().After(user.SetupTokenExpiresAt.UTC()) {
		return u.Logger.NewError(RegistrationCodeExpiredError)
	}
	if user.HashedPassword != "" {
		return u.Logger.NewError(RegistrationCodeNotFoundError)
	}

	hashedPassword, err := s.AuthHelper.SaltAndHash(form.Password)
	if err != nil {
		return err
	}
	return s.UserRepo.SetInitialPassword(user.Id, hashedPassword)
}

func (s *UserServiceImpl) DeleteMaintainerByAdmin(name string) error {
	user, err := s.getMaintainerByNameForAdmin(name)
	if err != nil {
		return err
	}
	if user.IsAdmin {
		return u.Logger.NewError(AdminMaintainerDeleteError)
	}
	return s.UserRepo.DeleteUser(user.Id)
}

func (s *UserServiceImpl) SetMaintainerStorageLimitByAdmin(name string, storageLimitInBytes int64) error {
	if storageLimitInBytes < 0 {
		return u.Logger.NewError(StorageLimitMustBeNonNegativeError)
	}
	user, err := s.getMaintainerByNameForAdmin(name)
	if err != nil {
		return err
	}
	user.StorageLimitInBytes = storageLimitInBytes
	return s.UserRepo.UpdateUser(user)
}

func (s *UserServiceImpl) getMaintainerByNameForAdmin(name string) (*tools.User, error) {
	user, err := s.UserRepo.GetUserByName(name)
	if err != nil {
		if u.ExtractError(err) == IncorrectUsernameOrPasswordError {
			return nil, u.Logger.NewError(MaintainerNotFoundError)
		}
		return nil, err
	}
	return user, nil
}

func (s *UserServiceImpl) CheckAuthenticationAndUpdateCookie(cookie *http.Cookie) (*tools.User, error) { // #nosec G124 (CWE-614): http.Cookie missing or has insecure Secure, HttpOnly, or SameSite attribute (Confidence: HIGH, Severity: MEDIUM)

	if err := validation.Validate("Cookie", validation.FieldSecret, cookie.Value); err != nil {
		return nil, u.Logger.NewError(InvalidCookieError)
	}

	hashedCookieValue := s.AuthHelper.GetSHA256Hash(cookie.Value)
	user, err := s.UserRepo.GetUserViaHashedCookie(hashedCookieValue)
	if err != nil {
		return nil, err
	}

	isExpired, err := s.isCookieExpired(cookie.Value)
	if err != nil {
		return nil, err
	}
	if isExpired {
		return nil, u.Logger.NewError(CookieExpiredError)
	}

	cookie.Expires = getAuthCookieExpiration()
	applyAuthCookieSecurity(cookie)
	err = s.saveCookie(user.Name, cookie.Value, cookie.Expires)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserServiceImpl) ChangePassword(user *tools.User, form *store.ChangePasswordForm) error {
	isCorrect, err := s.isPasswordCorrect(user.Name, form.OldPassword)
	if err != nil {
		return err
	}
	if !isCorrect {
		return u.Logger.NewError(IncorrectUsernameOrPasswordError)
	}
	err = s.UserRepo.ChangePassword(user.Id, form.NewPassword)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserServiceImpl) InitializeAdminAccountIfNotExisting() error {
	doesExist, err := s.UserRepo.DoesAdminAccountExist()
	if err != nil {
		return err
	}

	if doesExist {
		return nil
	}

	publicKey, err := s.getInitialAdminPublicKeyRaw()
	if err != nil {
		return err
	}

	adminName, adminPassword, err := bootstrap.GetInitialAdminCredentials(QuollixAdminUsername)
	if err != nil {
		return err
	}

	hashedPassword, err := s.AuthHelper.SaltAndHash(adminPassword)
	if err != nil {
		return err
	}
	return s.UserRepo.CreateUser(
		adminName,
		hashedPassword,
		QuollixAdminEmail,
		publicKey,
		tools.AdminStorageLimitInBytes,
		true,
	)

}

func (s *UserServiceImpl) getInitialAdminPublicKeyRaw() ([]byte, error) {
	if s.Config.UseSampleDataForTesting {
		return u.GetLocalTestingPublicKeyRaw(), nil
	}
	return u.DecodeAuthorizedEd25519PublicKey([]byte(store.AppStoreOfficialMaintainerPublicKeyOpenSSH))
}

func validateEd25519PublicKeyRaw(publicKeyRaw []byte) error {
	if _, err := ssh.NewPublicKey(ed25519.PublicKey(publicKeyRaw)); err != nil {
		return u.Logger.NewError(InvalidPublicKeyError)
	}
	return nil
}

func (s *UserServiceImpl) ChangeEmail(userId int, newEmail string) error {
	exists, err := s.UserRepo.DoesEmailExist(newEmail)
	if err != nil {
		return err
	}
	if exists {
		return u.Logger.NewError(EmailAlreadyExistsError)
	}

	user, err := s.UserRepo.GetUserById(userId)
	if err != nil {
		return err
	}
	user.Email = newEmail
	return s.UserRepo.UpdateUser(user)
}
