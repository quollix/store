package commands

import (
	"fmt"
	"path/filepath"

	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var onboardingCmd = &cobra.Command{
	Use:   "onboarding",
	Short: "complete one-time account setup tasks",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var onboardingSetupPasswordCmd = &cobra.Command{
	Use:   "setup-password <setup-token>",
	Short: "set the initial password for an app maintainer account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		password, err := promptForConfirmedPassword()
		if err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.SetupInitialPassword(args[0], password); err != nil {
			return err
		}
		fmt.Println("password setup successful")
		return nil
	},
}

var onboardingSetPrivateKeyCmd = &cobra.Command{
	Use:   "set-private-key <path>",
	Short: "configure the local private key path for signing uploads",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		passphrase, err := Dependencies.OsWrapper.PromptSecret("Private key passphrase: ")
		if err != nil {
			return err
		}
		if err := Dependencies.SigningKeyManager.SetPrivateKeyPath(args[0], passphrase); err != nil {
			return err
		}
		fmt.Println("private key path saved successfully")
		return nil
	},
}

var onboardingSetAppsDirectoryCmd = &cobra.Command{
	Use:   "set-apps-directory <path>",
	Short: "configure the local apps directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appsDirectory, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		if _, err := Dependencies.OsWrapper.ReadDir(appsDirectory); err != nil {
			return err
		}
		config, err := Dependencies.ConfigProvider.GetConfig()
		if err != nil {
			return err
		}
		config.AppsDirectory = appsDirectory
		if err := Dependencies.ConfigProvider.SetConfig(config); err != nil {
			return err
		}
		fmt.Println("apps directory saved successfully")
		return nil
	},
}
