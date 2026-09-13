package commands

import (
	"fmt"

	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var AppsCmd = &cobra.Command{
	Use:   "apps",
	Short: "manage apps in the Quollix App Store",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var createAppCmd = &cobra.Command{
	Use:   "create <app-name>",
	Short: "create a new app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		if err := Dependencies.AppStoreClient.CreateApp(appName); err != nil {
			return err
		}
		fmt.Println("app created successfully")
		return nil
	},
}

var listRemoteAppsCmd = &cobra.Command{
	Use:   "list",
	Short: "lists all apps in the remote app store",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apps, err := Dependencies.AppStoreClient.ListOwnApps()
		if err != nil {
			return err
		}
		if len(apps) == 0 {
			fmt.Println("no apps found")
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

var deleteRemoteAppCmd = &cobra.Command{
	Use:   "delete <app-name>",
	Short: "deletes an app from the remote app store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		operation := fmt.Sprintf("deletion of app '%s'", appName)
		if err := promptMaintainerToDeletionConfirmOperation(operation); err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.DeleteApp(appName); err != nil {
			return err
		}
		fmt.Println("app deleted successfully")
		return nil
	},
}

var searchRemoteAppsCmd = &cobra.Command{
	Use:   "search <maintainer-search-term> <app-search-term>",
	Short: "searches for apps in the remote app store; use \"\" as search term to match all",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		maintainerSearchTerm, appSearchTerm := args[0], args[1]
		apps, err := Dependencies.AppStoreClient.SearchForApps(maintainerSearchTerm, appSearchTerm, true)
		if err != nil {
			return err
		}
		if len(apps) == 0 {
			fmt.Println("no apps found")
			return nil
		}
		fmt.Print(renderAppSearchTable(apps))
		return nil
	},
}
