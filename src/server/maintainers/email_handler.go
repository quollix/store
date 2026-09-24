package maintainers

import (
	"net/http"
	"server/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

var expectedSmtpConnectionFailedErrors = u.MapOf(SmtpConnectionFailedError)

type EmailHandler struct {
	EmailConfigStore EmailConfigRepository
	EmailServiceImpl *EmailServiceImpl
	UserRepo         UserRepository
}

func (e *EmailHandler) GetEmailHandler(w http.ResponseWriter, r *http.Request) {
	config, err := e.EmailConfigStore.GetEmailConfig()
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
	u.SendJsonResponse(w, config)
}

func (e *EmailHandler) SetEmailHandler(w http.ResponseWriter, r *http.Request) {
	config, ok := validation.ReadBody[u.EmailConfig](w, r)
	if !ok {
		return
	}
	err := e.EmailServiceImpl.CheckAndSetEmailConfig(config)
	if err != nil {
		u.WriteResponseError(w, expectedSmtpConnectionFailedErrors, err)
		return
	}
}

func (e *EmailHandler) SendTestEmailHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	emailRequest, ok := validation.ReadBody[store.AdminEmailRequest](w, r)
	if !ok {
		return
	}
	result, err := e.EmailServiceImpl.SendEmails([]string{user.Email}, emailRequest.Subject, emailRequest.Body)
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
	u.SendJsonResponse(w, result)
}

func (e *EmailHandler) SendMaintainerEmailHandler(w http.ResponseWriter, r *http.Request) {
	emailRequest, ok := validation.ReadBody[store.AdminEmailRequest](w, r)
	if !ok {
		return
	}
	maintainerList, err := e.UserRepo.ListMaintainers()
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
	recipients := make([]string, 0, len(maintainerList))
	for _, maintainer := range maintainerList {
		recipients = append(recipients, maintainer.Email)
	}
	result, err := e.EmailServiceImpl.SendEmails(recipients, emailRequest.Subject, emailRequest.Body)
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
	u.SendJsonResponse(w, result)
}
