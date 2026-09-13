//go:build component

package check

import (
	"testing"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestAdminCanSendTestEmail(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()
	setTestEmailConfig(t, adminClient)

	result, err := adminClient.SendTestEmailByAdmin(u.SampleTestEmail.Subject, u.SampleTestEmail.Body)
	assert.Nil(t, err)
	assert.Equal(t, 1, result.SentCount)
	assert.Equal(t, 0, result.FailedCount)
	assert.Equal(t, 0, len(result.Failures))
}

func TestAdminCanSendEmailToMaintainers(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()
	setTestEmailConfig(t, adminClient)
	assert.Nil(t, createAndActivateMaintainer(t, adminClient, sampleMaintainer, sampleEmail, getOtherTestingPublicKey()))

	result, err := adminClient.SendEmailToMaintainersByAdmin(u.SampleMaintainersEmail.Subject, u.SampleMaintainersEmail.Body)
	assert.Nil(t, err)
	assert.Equal(t, 2, result.SentCount)
	assert.Equal(t, 0, result.FailedCount)
	assert.Equal(t, 0, len(result.Failures))
}

func TestAdminMaintainerEmailReturnsFailureSummary(t *testing.T) {
	adminClient := GetAdminStoreClientAndLogin(t)
	defer adminClient.WipeData()
	setTestEmailConfig(t, adminClient)
	assert.Nil(t, createAndActivateMaintainer(t, adminClient, "failingmaintainer", u.SampleEmailFailingRecipient, getOtherTestingPublicKey()))

	result, err := adminClient.SendEmailToMaintainersByAdmin(u.SampleMaintainersFailureEmail.Subject, u.SampleMaintainersFailureEmail.Body)
	assert.Nil(t, err)
	assert.Equal(t, 1, result.SentCount)
	assert.Equal(t, 1, result.FailedCount)
	assert.Equal(t, 1, len(result.Failures))
	assert.Equal(t, u.SampleEmailFailingRecipient, result.Failures[0].Recipient)
	assert.Equal(t, u.SampleEmailSendFailedError, result.Failures[0].Error)
}
