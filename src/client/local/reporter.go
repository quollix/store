package local

import (
	"fmt"
	"strings"

	"qsc/tools"
)

const (
	ansiRed   = "\033[31m"
	ansiGreen = "\033[32m"
	ansiReset = "\033[0m"
)

func ConvertUpdateReportToPrettyString(updateReport tools.FullUpdateReport) string {
	var builder strings.Builder
	for _, appUpdateReport := range updateReport.AppUpdateReports {
		addUpdateReportLine(appUpdateReport, &builder)
	}
	if updateReport.WasSuccessful {
		fmt.Fprintf(&builder, "summary: %soverall update successful%s\n", ansiGreen, ansiReset)
	} else {
		fmt.Fprintf(&builder, "summary: %soverall update failed%s\n", ansiRed, ansiReset)
	}
	addFailingAppsLine(getFailingUpdateApps(updateReport), &builder)
	return builder.String()
}

func addUpdateReportLine(report tools.AppUpdateReport, builder *strings.Builder) {
	if report.IsSuccessfulSoFar() && !report.HasUpdates() {
		fmt.Fprintf(builder, "- %s\n", colorizeUpdateLine(report.AppName+": OK, no updates found", ansiGreen))
	} else if report.IsSuccessfulSoFar() {
		fmt.Fprintf(builder, "- %s\n", colorizeUpdateLine(report.AppName+": OK, update successful", ansiGreen))
	} else {
		fmt.Fprintf(builder, "- %s\n", colorizeUpdateLine(report.AppName+": FAIL, update did not succeed, error: "+report.ErrorMessage, ansiRed))
	}
	for _, s := range report.ServiceUpdates {
		fmt.Fprintf(builder, "  - %s: %s -> %s\n", s.ServiceName, s.OldTag, s.NewTag)
	}
}

func colorizeUpdateLine(text, color string) string {
	return color + text + ansiReset
}

func addFailingAppsLine(failingApps []string, builder *strings.Builder) {
	if len(failingApps) == 0 {
		return
	}
	fmt.Fprintf(builder, "failing apps: %s\n", strings.Join(failingApps, " "))
}

func getFailingUpdateApps(report tools.FullUpdateReport) []string {
	var failingApps []string
	for _, appReport := range report.AppUpdateReports {
		if !appReport.IsSuccessfulSoFar() {
			failingApps = append(failingApps, appReport.AppName)
		}
	}
	return failingApps
}
