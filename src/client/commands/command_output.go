package commands

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

func renderAppSearchTable(apps []store.AppWithLatestVersion) string {
	rows := make([][]string, 0, len(apps))
	for _, app := range apps {
		rows = append(rows, []string{
			app.Maintainer,
			app.AppName,
			app.LatestVersionName,
			formatOptionalTimestamp(app.LatestVersionCreationTimestamp),
		})
	}
	return renderTable([]string{"maintainer", "app", "latest version", "latest version timestamp"}, rows)
}

func renderIndexedVersionsTable(versions []store.LeanVersionDto) string {
	rows := make([][]string, 0, len(versions))
	for index, version := range versions {
		rows = append(rows, []string{
			strconv.Itoa(index),
			version.Name,
			formatVersionTimestamp(version.CreationTimestamp),
			formatBytes(version.SizeInBytes),
			strconv.FormatBool(version.IsMigrationCheckpoint),
			strconv.FormatInt(version.DownloadCount, 10),
		})
	}
	return renderTable([]string{"index", "version", "timestamp", "size", "migration checkpoint", "downloads"}, rows)
}

func renderMaintainersTable(maintainers []store.AdminMaintainer) string {
	rows := make([][]string, 0, len(maintainers))
	for _, maintainer := range maintainers {
		status := "pending"
		if maintainer.IsActive {
			status = "active"
		}
		rows = append(rows, []string{maintainer.Name, maintainer.Email, status, getPublicKeyFingerprint(maintainer.PublicKeyRaw)})
	}
	return renderTable([]string{"name", "email", "status", "public key fingerprint"}, rows)
}

func renderTable(headers []string, rows [][]string) string {
	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)

	for i, header := range headers {
		if i > 0 {
			writeTableOutput(writer, "\t")
		}
		writeTableOutput(writer, header)
	}
	writeTableOutput(writer, "\n")

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				writeTableOutput(writer, "\t")
			}
			writeTableOutput(writer, cell)
		}
		writeTableOutput(writer, "\n")
	}

	logTableOutputError(writer.Flush())
	return buffer.String()
}

func writeTableOutput(writer io.Writer, output string) {
	_, err := fmt.Fprint(writer, output)
	logTableOutputError(err)
}

func logTableOutputError(err error) {
	if err != nil {
		u.Logger.Error(err, "result", "failed to render command table")
	}
}

func formatOptionalTimestamp(timestamp time.Time) string {
	if timestamp.IsZero() {
		return "-"
	}
	return formatVersionTimestamp(timestamp)
}
