package maintainers

import (
	"server/tools"

	u "github.com/quollix/common/utils"
)

var DefaultEmailConfig = u.EmailConfig{
	SMTPHost:             defaultSMTPHost,
	SMTPPort:             defaultSMTPPort,
	FromEmailAddress:     defaultEmail,
	EmailAccountUsername: defaultEmailUser,
	EmailAccountPassword: defaultEmailPassword,
}

type EmailConfigRepository interface {
	GetEmailConfig() (*u.EmailConfig, error)
	SetEmailConfig(cfg *u.EmailConfig) error
}

const (
	smtpHostKey      = "EMAIL_SMTP_HOST"
	smtpPortKey      = "EMAIL_SMTP_PORT"
	emailKey         = "EMAIL_ADDRESS"
	emailUserKey     = "EMAIL_ACCOUNT_USERNAME"
	emailPasswordKey = "EMAIL_ACCOUNT_PASSWORD"

	DefaultAppStoreHost  = "http://localhost:8080"
	defaultSMTPHost      = "smtps.sample.com"
	defaultSMTPPort      = "465"
	defaultEmail         = "sample@sample.com"
	defaultEmailUser     = "sample"
	defaultEmailPassword = "password"
)

type EmailConfigRepositoryImpl struct {
	DatabaseProvider *tools.DatabaseProviderImpl
}

func (s *EmailConfigRepositoryImpl) GetEmailConfig() (*u.EmailConfig, error) {
	d := DefaultEmailConfig
	var cfg u.EmailConfig
	err := s.DatabaseProvider.GetDb().QueryRow(`
		SELECT
			COALESCE(MAX(CASE WHEN key = $1  THEN value END), $2),
			COALESCE(MAX(CASE WHEN key = $3  THEN value END), $4),
			COALESCE(MAX(CASE WHEN key = $5  THEN value END), $6),
			COALESCE(MAX(CASE WHEN key = $7  THEN value END), $8),
			COALESCE(MAX(CASE WHEN key = $9  THEN value END), $10)
		FROM configs
		WHERE key IN ($1,$3,$5,$7,$9)
	`,
		smtpHostKey, d.SMTPHost,
		smtpPortKey, d.SMTPPort,
		emailKey, d.FromEmailAddress,
		emailUserKey, d.EmailAccountUsername,
		emailPasswordKey, d.EmailAccountPassword,
	).Scan(
		&cfg.SMTPHost,
		&cfg.SMTPPort,
		&cfg.FromEmailAddress,
		&cfg.EmailAccountUsername,
		&cfg.EmailAccountPassword,
	)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	return &cfg, nil
}

func (s *EmailConfigRepositoryImpl) SetEmailConfig(config *u.EmailConfig) error {
	if _, err := s.DatabaseProvider.GetDb().Exec(`
		INSERT INTO configs(key, value) VALUES
			($1,$2),($3,$4),($5,$6),($7,$8),($9,$10)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`,
		smtpHostKey, config.SMTPHost,
		smtpPortKey, config.SMTPPort,
		emailKey, config.FromEmailAddress,
		emailUserKey, config.EmailAccountUsername,
		emailPasswordKey, config.EmailAccountPassword,
	); err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}
