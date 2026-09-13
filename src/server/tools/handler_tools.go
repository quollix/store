package tools

import (
	"net/http"

	u "github.com/quollix/common/utils"
)

var (
	DoesNotExistError                = "does not exist"
	DoesNotExistErrorMap             = u.MapOf(DoesNotExistError)
	UserMissingInRequestContextError = "authenticated user missing from request context"
)

type ContextKey string

const UserCtxKey ContextKey = "user"

func GetUserFromContext(r *http.Request) (*User, bool) {
	user, ok := r.Context().Value(UserCtxKey).(*User)
	if !ok || user == nil {
		return nil, false
	}
	return user, true
}

func GetUserFromContextOrWriteError(w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := GetUserFromContext(r)
	if !ok {
		u.WriteResponseErrorAlways(w, u.Logger.NewError(UserMissingInRequestContextError))
		return nil, false
	}
	return user, true
}
