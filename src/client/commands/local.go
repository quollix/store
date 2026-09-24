package commands

import (
	"fmt"

	"qsc/tools"

	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var LocalCmd = &cobra.Command{
	Use:   "local",
	Short: "work with local apps in the apps directory",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var localDockerHubCmd = &cobra.Command{
	Use:   "docker-hub",
	Short: "manage Docker Hub authentication for local app updates",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var localDockerHubSetCmd = &cobra.Command{
	Use:   "set <username>",
	Short: "configure Docker Hub authentication for tag listing",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := Dependencies.OsWrapper.PromptSecret("Docker Hub token: ")
		if err != nil {
			return err
		}
		if err := Dependencies.SessionManager.SetDockerHubAuth(args[0], token); err != nil {
			return err
		}
		fmt.Println("Docker Hub authentication saved successfully")
		return nil
	},
}

var localDockerHubDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete Docker Hub authentication",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := Dependencies.SessionManager.DeleteDockerHubAuth(); err != nil {
			return err
		}
		fmt.Println("Docker Hub authentication deleted successfully")
		return nil
	},
}

var localDockerHubShowCmd = &cobra.Command{
	Use:   "show",
	Short: "print Docker Hub authentication for local app updates",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := Dependencies.ConfigProvider.GetConfig()
		if err != nil {
			return err
		}
		if config == nil || config.DockerHub == nil {
			fmt.Print("- not configured\n")
			return nil
		}
		fmt.Print(renderDockerHubConfig(config.DockerHub))
		return nil
	},
}

func renderDockerHubConfig(config *tools.DockerHubAuth) string {
	return fmt.Sprintf("- username: %s\n- token: %s\n", config.Username, maskSecret(config.Token))
}
