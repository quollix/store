package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "manage app versions",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var uploadVersionCmd = &cobra.Command{
	Use:   "upload [app-name...]",
	Short: "uploads app compose files as new versions",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := Dependencies.SessionManager.RequireSession()
		if err != nil {
			return err
		}
		privateKeyPath, err := Dependencies.SigningKeyManager.GetPrivateKeyPath()
		if err != nil {
			return err
		}
		privateKeyPassphrase, err := Dependencies.OsWrapper.PromptSecret("Private key passphrase: ")
		if err != nil {
			return err
		}
		if err := Dependencies.SigningKeyManager.ValidateConfiguredPrivateKey(privateKeyPassphrase); err != nil {
			return err
		}
		uploadResults, err := uploadVersionsFromAppsDirectory(versionUploadConfig{
			Maintainer:           session.Maintainer,
			PrivateKeyPath:       privateKeyPath,
			PrivateKeyPassphrase: privateKeyPassphrase,
			SelectedApps:         args,
		})
		for _, result := range uploadResults {
			fmt.Printf("%s: %s\n", result.AppName, result.Message)
		}
		return err
	},
}

var deleteVersionCmd = &cobra.Command{
	Use:   "delete <app-name>",
	Short: "deletes a version of an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]

		session, err := Dependencies.SessionManager.RequireSession()
		if err != nil {
			return err
		}
		selectedVersion, err := selectSingleVersion(session.Maintainer, appName, "Enter version index to delete: ")
		if err != nil {
			return err
		}
		versionID := selectedVersion.VersionId

		operationName := fmt.Sprintf("deletion of app '%s' version '%s' from %s", appName, selectedVersion.Name, formatVersionTimestamp(selectedVersion.CreationTimestamp))
		if err := promptMaintainerToDeletionConfirmOperation(operationName); err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.DeleteVersionByID(versionID); err != nil {
			return err
		}
		fmt.Println("version deleted successfully")
		return nil
	},
}

var listVersionsCmd = &cobra.Command{
	Use:   "list <maintainer-name> <app-name>",
	Short: "lists all versions of an app",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		maintainerName := args[0]
		appName := args[1]

		versions, err := Dependencies.AppStoreClient.ListVersions(maintainerName, appName)
		if err != nil {
			return err
		}
		printIndexedVersionsTable(orderVersionsOldestFirst(versions))
		return nil
	},
}

var versionCheckpointCmd = &cobra.Command{
	Use:   "checkpoint",
	Short: "marks or unmarks migration checkpoints",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var markVersionCheckpointCmd = &cobra.Command{
	Use:   "mark <app-name>",
	Short: "marks a version as a migration checkpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]

		session, err := Dependencies.SessionManager.RequireSession()
		if err != nil {
			return err
		}
		selectedVersion, err := selectSingleVersion(session.Maintainer, appName, "Enter version index to mark as migration checkpoint: ")
		if err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.SetVersionMigrationCheckpoint(selectedVersion.VersionId, true); err != nil {
			return err
		}
		fmt.Println("migration checkpoint marked successfully")
		return nil
	},
}

var unmarkVersionCheckpointCmd = &cobra.Command{
	Use:   "unmark <app-name>",
	Short: "unmarks a version as a migration checkpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]

		session, err := Dependencies.SessionManager.RequireSession()
		if err != nil {
			return err
		}
		selectedVersion, err := selectSingleVersion(session.Maintainer, appName, "Enter version index to unmark as migration checkpoint: ")
		if err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.SetVersionMigrationCheckpoint(selectedVersion.VersionId, false); err != nil {
			return err
		}
		fmt.Println("migration checkpoint unmarked successfully")
		return nil
	},
}

var cloneVersionsCmd = &cobra.Command{
	Use:   "clone <download-path>",
	Short: "downloads the latest version of every app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		downloadPath := args[0]
		session, err := Dependencies.SessionManager.RequireSession()
		if err != nil {
			return err
		}
		appCount, err := cloneLatestVersions(session.Maintainer, downloadPath)
		if err != nil {
			return err
		}
		fmt.Printf("versions cloned successfully: %d apps\n", appCount)
		return nil
	},
}

var contentVersionCmd = &cobra.Command{
	Use:   "content <maintainer-name> <app-name>",
	Short: "prints the content of a version",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		maintainerName := args[0]
		appName := args[1]

		selectedVersion, err := selectSingleVersion(maintainerName, appName, "Enter version index to print content: ")
		if err != nil {
			return err
		}
		content, err := fetchVerifiedVersionContentByID(selectedVersion.VersionId)
		if err != nil {
			return err
		}
		fmt.Print(string(content))
		return nil
	},
}

var diffVersionsCmd = &cobra.Command{
	Use:   "diff <maintainer-name> <app-name>",
	Short: "prints a local git diff between two versions",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		maintainerName := args[0]
		appName := args[1]

		leftVersion, rightVersion, err := selectTwoVersionsForDiff(maintainerName, appName)
		if err != nil {
			return err
		}

		leftContent, err := fetchVerifiedVersionContentByID(leftVersion.VersionId)
		if err != nil {
			return err
		}
		rightContent, err := fetchVerifiedVersionContentByID(rightVersion.VersionId)
		if err != nil {
			return err
		}
		diffLines, err := DiffLines(SplitDiffLines(string(leftContent)), SplitDiffLines(string(rightContent)))
		if err != nil {
			return err
		}
		fmt.Print(RenderDiff(diffLines))
		return nil
	},
}

func selectSingleVersion(maintainerName, appName, prompt string) (*store.LeanVersionDto, error) {
	versions, err := Dependencies.AppStoreClient.ListVersions(maintainerName, appName)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, u.Logger.NewError("version does not exist")
	}

	selectableVersions := orderVersionsOldestFirst(versions)
	printIndexedVersionsTable(selectableVersions)
	selectedIndex, err := promptForVersionIndex(prompt, len(selectableVersions))
	if err != nil {
		return nil, err
	}
	return &selectableVersions[selectedIndex], nil
}

func selectTwoVersionsForDiff(maintainerName, appName string) (*store.LeanVersionDto, *store.LeanVersionDto, error) {
	versions, err := Dependencies.AppStoreClient.ListVersions(maintainerName, appName)
	if err != nil {
		return nil, nil, err
	}
	if len(versions) < 2 {
		return nil, nil, u.Logger.NewError("at least two versions are required")
	}

	selectableVersions := orderVersionsOldestFirst(versions)
	printIndexedVersionsTable(selectableVersions)
	input, err := Dependencies.OsWrapper.PromptUser("Enter two space-separated version indices to diff, e.g. '0 1': ")
	if err != nil {
		return nil, nil, err
	}
	leftIndex, rightIndex, err := parseTwoVersionIndices(input, len(selectableVersions))
	if err != nil {
		return nil, nil, err
	}
	return &selectableVersions[leftIndex], &selectableVersions[rightIndex], nil
}

func promptForVersionIndex(prompt string, versionCount int) (int, error) {
	input, err := Dependencies.OsWrapper.PromptUser(prompt)
	if err != nil {
		return 0, err
	}
	selectedIndex, err := strconv.Atoi(input)
	if err != nil {
		return 0, u.Logger.NewError("selected version index must be a number")
	}
	if selectedIndex < 0 || selectedIndex >= versionCount {
		return 0, u.Logger.NewError("selected version index does not match listed versions")
	}
	return selectedIndex, nil
}

func parseTwoVersionIndices(input string, versionCount int) (int, int, error) {
	parts := strings.Fields(input)
	if len(parts) != 2 {
		return 0, 0, u.Logger.NewError("exactly two version indices are required")
	}
	leftIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, u.Logger.NewError("selected version indices must be numbers")
	}
	rightIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, u.Logger.NewError("selected version indices must be numbers")
	}
	if leftIndex == rightIndex {
		return 0, 0, u.Logger.NewError("selected version indices must be different")
	}
	if leftIndex < 0 || leftIndex >= versionCount || rightIndex < 0 || rightIndex >= versionCount {
		return 0, 0, u.Logger.NewError("selected version index does not match listed versions")
	}
	return leftIndex, rightIndex, nil
}

func printIndexedVersionsTable(versions []store.LeanVersionDto) {
	fmt.Print(renderIndexedVersionsTable(versions))
}

func formatVersionTimestamp(timestamp time.Time) string {
	return timestamp.Format("2006-01-02 15:04:05")
}
