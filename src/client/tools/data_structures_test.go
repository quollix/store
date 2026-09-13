package tools

import (
	"testing"

	"github.com/quollix/common/assert"
)

func TestAppUpdateReport_IsSuccessfulSoFar(t *testing.T) {
	report := AppUpdateReport{}

	assert.True(t, report.IsSuccessfulSoFar())

	report.ErrorMessage = "some error"
	assert.False(t, report.IsSuccessfulSoFar())
}

func TestAppUpdateReport_HasUpdates(t *testing.T) {
	report := AppUpdateReport{}

	assert.False(t, report.HasUpdates())

	report.ServiceUpdates = []ServiceUpdate{{ServiceName: "web", OldTag: "1.0", NewTag: "1.1"}}
	assert.True(t, report.HasUpdates())
}
