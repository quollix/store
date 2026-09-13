package maintainers

import (
	"net/http"
	"testing"
	"time"

	"server/tools"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/deepstack"
)

type testObjectsType struct {
	userService     UserServiceImpl
	userRepoMock    *UserRepositoryMock
	secretGenerator *SecretGeneratorMock
	authHelperMock  *tools.AuthHelperMock
}

func setupTestObjects(t *testing.T) *testObjectsType {
	testObjects := &testObjectsType{}
	testObjects.userRepoMock = NewUserRepositoryMock(t)
	testObjects.secretGenerator = NewSecretGeneratorMock(t)
	testObjects.authHelperMock = tools.NewAuthHelperMock(t)

	testObjects.userService.UserRepo = testObjects.userRepoMock
	testObjects.userService.SecretGenerator = testObjects.secretGenerator
	testObjects.userService.AuthHelper = testObjects.authHelperMock
	testObjects.userService.Config = &tools.Config{}
	return testObjects
}

func TestCookieExpirationThrowsError(t *testing.T) {
	testObjects := setupTestObjects(t)

	cookie := &http.Cookie{
		Value:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Expires: time.Now().UTC().Add(10 * time.Minute),
	}

	hashedCookieValue := "hashed-cookie"
	testObjects.authHelperMock.EXPECT().GetSHA256Hash(cookie.Value).Return(hashedCookieValue).Twice()

	expiredAt := time.Now().UTC().Add(-1 * time.Minute)
	user := &tools.User{ExpirationDate: &expiredAt}

	testObjects.userRepoMock.EXPECT().GetUserViaHashedCookie(hashedCookieValue).Return(user, nil).Twice()

	_, err := testObjects.userService.CheckAuthenticationAndUpdateCookie(cookie)
	deepstack.AssertDeepStackError(t, err, CookieExpiredError)
}

func TestSetupInitialPasswordRejectsExpiredToken(t *testing.T) {
	testObjects := setupTestObjects(t)

	setupToken := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hashedSetupToken := "hashed-setup-token"
	expiredAt := time.Now().UTC().Add(-1 * time.Minute)
	user := &tools.User{SetupTokenExpiresAt: &expiredAt}

	testObjects.authHelperMock.EXPECT().GetSHA256Hash(setupToken).Return(hashedSetupToken)
	testObjects.userRepoMock.EXPECT().GetUserBySetupTokenHash(hashedSetupToken).Return(user, nil)

	err := testObjects.userService.SetupInitialPassword(&store.SetupInitialPasswordForm{
		Token:    setupToken,
		Password: "password",
	})
	assert.Equal(t, RegistrationCodeExpiredError, u.ExtractError(err))
}
