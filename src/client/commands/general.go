package commands

import (
	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var Dependencies *ClientDependencies

func InitializeCobraCommands(deps *ClientDependencies) {
	Dependencies = deps
	if signInCmd.Flags().Lookup("password") == nil {
		signInCmd.Flags().StringP("password", "p", "", "password to use for sign-in; prompted securely when omitted")
	}
	sessionCmd.AddCommand(sessionShowCmd, sessionDeleteCmd)
	onboardingCmd.AddCommand(onboardingSetupPasswordCmd, onboardingSetPrivateKeyCmd, onboardingSetAppsDirectoryCmd)
	localDockerHubCmd.AddCommand(localDockerHubSetCmd, localDockerHubDeleteCmd, localDockerHubShowCmd)
	LocalCmd.AddCommand(updateCmd, validateCmd, imageTagCacheCmd, listAppsCommand, localDockerHubCmd, uploadVersionCmd)
	Account.AddCommand(signInCmd, logoutCmd, deleteOwnAccountCmd, changePasswordCmd, accountDetailsCmd, changeEmailCmd)
	adminMaintainersCmd.AddCommand(createMaintainerCmd, listMaintainersCmd, deleteMaintainerCmd, setMaintainerSpaceCmd)
	adminEmailCmd.AddCommand(getEmailConfigCmd, setEmailConfigCmd, sendTestEmailToMyselfCmd, sendEmailToAllMaintainersCmd)
	adminCmd.AddCommand(adminMaintainersCmd, adminEmailCmd)
	AppsCmd.AddCommand(createAppCmd, listRemoteAppsCmd, deleteRemoteAppCmd, searchRemoteAppsCmd)
	versionCheckpointCmd.AddCommand(markVersionCheckpointCmd, unmarkVersionCheckpointCmd)
	versionsCmd.AddCommand(deleteVersionCmd, listVersionsCmd, cloneVersionsCmd, contentVersionCmd, diffVersionsCmd, versionCheckpointCmd)
	devCmd.AddCommand(devBootstrapCmd)
	RootCmd.AddCommand(Account, adminCmd, AppsCmd, onboardingCmd, sessionCmd, versionsCmd, devCmd, LocalCmd)
	RootCmd.CompletionOptions = cobra.CompletionOptions{DisableDefaultCmd: true}
	RootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
}

var RootCmd = &cobra.Command{
	Use:           "qsc",
	Short:         "CLI for managing local apps and the Quollix App Store",
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

func promptMaintainerToDeletionConfirmOperation(operationName string) error {
	return promptForExactValue("Type DELETE to confirm "+operationName+": ", "DELETE")
}

func promptForExactValue(prompt string, expectedValue string) error {
	input, err := Dependencies.OsWrapper.PromptUser(prompt)
	if err != nil {
		return err
	}
	if input != expectedValue {
		return u.Logger.NewError("confirmation did not match; aborting")
	}
	return nil
}
