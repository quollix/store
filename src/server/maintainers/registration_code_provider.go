package maintainers

import (
	u "github.com/quollix/common/utils"
)

var (
	DefaultAccountRegistrationCode = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

type SecretGenerator interface {
	GenerateAccountRegistrationCode() (string, error)
}

type RegistrationCodeProviderImpl struct {
	AuthHelper u.AuthHelper
}

func (p *RegistrationCodeProviderImpl) GenerateAccountRegistrationCode() (string, error) {
	return p.AuthHelper.GenerateSecret()
}

type RegistrationCodeProviderMock struct{}

func (r RegistrationCodeProviderMock) GenerateAccountRegistrationCode() (string, error) {
	return DefaultAccountRegistrationCode, nil
}
