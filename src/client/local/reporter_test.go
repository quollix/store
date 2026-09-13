package local

import (
	"testing"

	"qsc/tools"

	"github.com/quollix/common/assert"
)

func TestConvertUpdateReportToPrettyString_WithServiceUpdates(t *testing.T) {
	report := tools.FullUpdateReport{
		WasSuccessful: true,
		AppUpdateReports: []tools.AppUpdateReport{{
			AppName: "sample-app",
			ServiceUpdates: []tools.ServiceUpdate{
				{ServiceName: "nginx", OldTag: "1.0", NewTag: "1.1"},
				{ServiceName: "redis", OldTag: "7.0", NewTag: "7.1"},
			},
		}},
	}

	actual := ConvertUpdateReportToPrettyString(report)

	expected := "- " + ansiGreen + "sample-app: OK, update successful" + ansiReset + "\n" +
		"  - nginx: 1.0 -> 1.1\n" +
		"  - redis: 7.0 -> 7.1\n" +
		"summary: " + ansiGreen + "overall update successful" + ansiReset + "\n"
	assert.Equal(t, expected, actual)
}

func TestConvertUpdateReportToPrettyString_WithNoUpdates(t *testing.T) {
	report := tools.FullUpdateReport{
		WasSuccessful:    true,
		AppUpdateReports: []tools.AppUpdateReport{{AppName: "sample-app"}},
	}

	actual := ConvertUpdateReportToPrettyString(report)

	expected := "- " + ansiGreen + "sample-app: OK, no updates found" + ansiReset + "\n" +
		"summary: " + ansiGreen + "overall update successful" + ansiReset + "\n"
	assert.Equal(t, expected, actual)
}

func TestConvertUpdateReportToPrettyString_ListsFailingApps(t *testing.T) {
	report := tools.FullUpdateReport{
		WasSuccessful: false,
		AppUpdateReports: []tools.AppUpdateReport{
			{AppName: "nextcloud", ErrorMessage: "registry request failed"},
			{AppName: "vaultwarden"},
			{AppName: "wordpress", ErrorMessage: "registry request failed"},
		},
	}

	actual := ConvertUpdateReportToPrettyString(report)

	expected := "- " + ansiRed + "nextcloud: FAIL, update did not succeed, error: registry request failed" + ansiReset + "\n" +
		"- " + ansiGreen + "vaultwarden: OK, no updates found" + ansiReset + "\n" +
		"- " + ansiRed + "wordpress: FAIL, update did not succeed, error: registry request failed" + ansiReset + "\n" +
		"summary: " + ansiRed + "overall update failed" + ansiReset + "\n" +
		"failing apps: nextcloud wordpress\n"
	assert.Equal(t, expected, actual)
}
