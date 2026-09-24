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

	expected := "- " + tools.AnsiGreen + "sample-app: OK, update successful" + tools.AnsiReset + "\n" +
		"  - nginx: 1.0 -> 1.1\n" +
		"  - redis: 7.0 -> 7.1\n" +
		"summary: " + tools.AnsiGreen + "overall update successful" + tools.AnsiReset + "\n"
	assert.Equal(t, expected, actual)
}

func TestConvertUpdateReportToPrettyString_WithNoUpdates(t *testing.T) {
	report := tools.FullUpdateReport{
		WasSuccessful:    true,
		AppUpdateReports: []tools.AppUpdateReport{{AppName: "sample-app"}},
	}

	actual := ConvertUpdateReportToPrettyString(report)

	expected := "- " + tools.AnsiGreen + "sample-app: OK, no updates found" + tools.AnsiReset + "\n" +
		"summary: " + tools.AnsiGreen + "overall update successful" + tools.AnsiReset + "\n"
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

	expected := "- " + tools.AnsiRed + "nextcloud: FAIL, update did not succeed, error: registry request failed" + tools.AnsiReset + "\n" +
		"- " + tools.AnsiGreen + "vaultwarden: OK, no updates found" + tools.AnsiReset + "\n" +
		"- " + tools.AnsiRed + "wordpress: FAIL, update did not succeed, error: registry request failed" + tools.AnsiReset + "\n" +
		"summary: " + tools.AnsiRed + "overall update failed" + tools.AnsiReset + "\n" +
		"failing apps: nextcloud wordpress\n"
	assert.Equal(t, expected, actual)
}
