package commands

import (
	"fmt"
	"os"

	"qsc/local"
	"qsc/tools"

	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "updates local app compose files",
	RunE: func(cmd *cobra.Command, args []string) error {
		u.Logger.Info("starting update process")
		updateReport, err := Dependencies.Updater.PerformUpdate(local.UpdateConfig{
			SelectedApps: args,
		})
		if err != nil {
			return err
		}
		fmt.Print(local.ConvertUpdateReportToPrettyString(*updateReport))
		return nil
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "validates local app compose files for consistency",
	RunE: func(cmd *cobra.Command, args []string) error {
		found, err := Dependencies.ConsistencyChecker.PerformConsistencyChecks(args)
		if err != nil {
			return err
		}
		if !found {
			fmt.Printf("no apps found in apps directory: %s\n", tools.AppsDir)
			return nil
		}
		fmt.Println("consistency check successful")
		return nil
	},
}

var imageTagCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "prints the current image tag cache file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, err := getImageTagCacheOutput(Dependencies.Config.ImageTagCachePath, Dependencies.OsWrapper)
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	},
}

func getImageTagCacheOutput(cachePath string, osWrapper u.OsWrapper) (string, error) {
	data, err := osWrapper.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "image tag cache is empty\n", nil
		}
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	return string(data) + "\n", nil
}

var listAppsCommand = &cobra.Command{
	Use:   "list",
	Short: "lists apps from the apps directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apps, err := Dependencies.AppSelector.SelectApps(nil)
		if err != nil {
			return err
		}
		if len(apps) == 0 {
			fmt.Printf("no apps found in apps directory: %s\n", tools.AppsDir)
			return nil
		}

		var output string
		for _, app := range apps {
			output += app + "\n"
		}
		fmt.Print(output)
		return nil
	},
}
