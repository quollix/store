package commands

import (
	"fmt"
	"os"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "manage the Quollix App Store (admins only)",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var adminEmailCmd = &cobra.Command{
	Use:   "email",
	Short: "manage admin email settings and sending",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var sendTestEmailToMyselfCmd = &cobra.Command{
	Use:   "send-test <subject> <body-file>",
	Short: "send a test email to the authenticated admin",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject, bodyFile := args[0], args[1]
		body, err := readAdminEmailBody(bodyFile)
		if err != nil {
			return err
		}
		result, err := Dependencies.AppStoreClient.SendTestEmailByAdmin(subject, body)
		if err != nil {
			return err
		}
		return printAdminEmailResult(result)
	},
}

var sendEmailToAllMaintainersCmd = &cobra.Command{
	Use:   "send-to-maintainers <subject> <body-file>",
	Short: "send an email to all app maintainers",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject, bodyFile := args[0], args[1]
		body, err := readAdminEmailBody(bodyFile)
		if err != nil {
			return err
		}
		result, err := Dependencies.AppStoreClient.SendEmailToMaintainersByAdmin(subject, body)
		if err != nil {
			return err
		}
		return printAdminEmailResult(result)
	},
}

func readAdminEmailBody(bodyFile string) (string, error) {
	body, err := os.ReadFile(bodyFile) // #nosec G304 -- admin explicitly provides the email body file path.
	if err != nil {
		return "", u.Logger.NewError(err.Error())
	}
	return string(body), nil
}

var setEmailConfigCmd = &cobra.Command{
	Use:   "set-config <smtp-host> <smtp-port> <email-address> <email-account-username>",
	Short: "set the server email config",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		password, err := Dependencies.OsWrapper.PromptSecret("Email account password: ")
		if err != nil {
			return err
		}
		ec := &u.EmailConfig{}
		ec.SMTPHost, ec.SMTPPort, ec.FromEmailAddress, ec.EmailAccountUsername, ec.EmailAccountPassword = args[0], args[1], args[2], args[3], password
		if err := Dependencies.AppStoreClient.SetEmailConfig(ec); err != nil {
			return err
		}
		fmt.Println("email config set successfully")
		return nil
	},
}

func printAdminEmailResult(result *store.AdminEmailSendResult) error {
	fmt.Println("emails sent:", result.SentCount)
	fmt.Println("emails failed:", result.FailedCount)
	if len(result.Failures) > 0 {
		fmt.Println("failed recipients:")
		for _, failure := range result.Failures {
			if failure.Recipient == "" {
				fmt.Println("-", failure.Error)
				continue
			}
			fmt.Printf("- %s: %s\n", failure.Recipient, failure.Error)
		}
	}
	if result.FailedCount > 0 {
		return u.Logger.NewError("email sending failed for one or more recipients")
	}
	return nil
}

var getEmailConfigCmd = &cobra.Command{
	Use:   "get-config",
	Short: "get the server email config",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ec, err := Dependencies.AppStoreClient.GetEmailConfig()
		if err != nil {
			return err
		}
		fmt.Println("smtp-host:", ec.SMTPHost)
		fmt.Println("smtp-port:", ec.SMTPPort)
		fmt.Println("email-address:", ec.FromEmailAddress)
		fmt.Println("email-account-username:", ec.EmailAccountUsername)
		return nil
	},
}
