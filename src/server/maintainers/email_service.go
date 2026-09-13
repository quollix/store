package maintainers

import (
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

var SmtpConnectionFailedError = "Failed to connect to the SMTP server with the provided email configuration. It was not saved."

type EmailService interface {
	CheckAndSetEmailConfig(emailConfig *u.EmailConfig) error
	SendEmail(to, subject, body string) error
	SendEmails(recipients []string, subject, body string) (*store.AdminEmailSendResult, error)
}

type EmailServiceImpl struct {
	EmailConfigRepo EmailConfigRepository
	EmailClient     u.EmailClient
}

func (e *EmailServiceImpl) CheckAndSetEmailConfig(emailConfig *u.EmailConfig) error {
	emailConfig.IsEnabled = true
	err := e.EmailClient.CheckEmailServerConnection(emailConfig)
	if err != nil {
		u.Logger.Info(err.Error())
		return u.Logger.NewError(SmtpConnectionFailedError)
	}
	return e.EmailConfigRepo.SetEmailConfig(emailConfig)
}

func (e *EmailServiceImpl) SendEmail(to, subject, body string) error {
	emailConfig, err := e.getEnabledEmailConfig()
	if err != nil {
		return err
	}
	return e.EmailClient.SendEmail(emailConfig, to, subject, body)
}

func (e *EmailServiceImpl) SendEmails(recipients []string, subject, body string) (*store.AdminEmailSendResult, error) {
	emailConfig, err := e.getEnabledEmailConfig()
	if err != nil {
		return nil, err
	}

	result := &store.AdminEmailSendResult{}
	for _, recipient := range recipients {
		err := e.EmailClient.SendEmail(emailConfig, recipient, subject, body)
		addEmailSendOutcome(result, emailSendOutcome{recipient: recipient, err: err})
	}
	return result, nil
}

func (e *EmailServiceImpl) getEnabledEmailConfig() (*u.EmailConfig, error) {
	emailConfig, err := e.EmailConfigRepo.GetEmailConfig()
	if err != nil {
		return nil, err
	}
	emailConfig.IsEnabled = true
	return emailConfig, nil
}

type emailSendOutcome struct {
	recipient string
	err       error
}

func addEmailSendOutcome(result *store.AdminEmailSendResult, outcome emailSendOutcome) {
	if outcome.err == nil {
		result.SentCount++
		return
	}

	result.FailedCount++
	errorMessage := u.ExtractError(outcome.err)
	u.Logger.Warn("email send failed", "recipient", outcome.recipient, "error", errorMessage)
	result.Failures = append(result.Failures, store.AdminEmailSendFailure{
		Recipient: outcome.recipient,
		Error:     errorMessage,
	})
}
