package maintainers

import (
	"fmt"
	"reflect"
	"strings"

	"server/tools"

	u "github.com/quollix/common/utils"
)

const (
	testEmailUnexpectedConfigError      = "test email client received unexpected email config"
	testEmailUnexpectedMessageError     = "test email client received unexpected email message"
	testEmailUnexpectedRecipientError   = "test email client received unexpected recipient"
	testEmailUnexpectedEnabledFlagError = "test email client received disabled email config"
)

type TestEmailClient struct{}

func (e *TestEmailClient) SendEmail(emailConfig *u.EmailConfig, to, subject, body string) error {
	if err := assertExpectedTestEmailConfig(emailConfig); err != nil {
		return err
	}
	if !isExpectedTestEmailMessage(subject, body) {
		return u.Logger.NewError(testEmailUnexpectedMessageError, "subject", subject, "body", body)
	}
	if !isExpectedTestEmailRecipient(to) {
		return u.Logger.NewError(testEmailUnexpectedRecipientError, "recipient", to)
	}
	if to == u.SampleEmailFailingRecipient && subject == u.SampleMaintainersFailureEmail.Subject {
		return u.Logger.NewError(u.SampleEmailSendFailedError)
	}
	return nil
}

func (e *TestEmailClient) CheckEmailServerConnection(emailConfig *u.EmailConfig) error {
	return assertExpectedTestEmailConfig(emailConfig)
}

func assertExpectedTestEmailConfig(emailConfig *u.EmailConfig) error {
	expectedEmailConfig := u.SampleEmailConfig
	expectedEmailConfig.IsEnabled = true
	if !emailConfig.IsEnabled {
		return u.Logger.NewError(testEmailUnexpectedEnabledFlagError)
	}
	if !reflect.DeepEqual(expectedEmailConfig, *emailConfig) {
		return u.Logger.NewError(testEmailUnexpectedConfigError, "expected", fmt.Sprintf("%+v", expectedEmailConfig), "actual", fmt.Sprintf("%+v", *emailConfig))
	}
	return nil
}

func isExpectedTestEmailMessage(subject, body string) bool {
	return (subject == u.SampleTestEmail.Subject && body == u.SampleTestEmail.Body) ||
		(subject == u.SampleMaintainersEmail.Subject && body == u.SampleMaintainersEmail.Body) ||
		(subject == u.SampleMaintainersFailureEmail.Subject && body == u.SampleMaintainersFailureEmail.Body) ||
		(subject == MaintainerSetupEmailSubject &&
			strings.Contains(body, "Setup token: "+DefaultAccountRegistrationCode))
}

func isExpectedTestEmailRecipient(to string) bool {
	switch to {
	case QuollixAdminEmail, tools.SampleEmail, u.SampleEmailFailingRecipient:
		return true
	default:
		return false
	}
}
