package commands

import (
	"crypto/ed25519"
	"fmt"
	"qsc/configuration"
	"time"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var Account = &cobra.Command{
	Use:   "account",
	Short: "manage your account",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var signInCmd = &cobra.Command{
	Use:   "sign-in <username>",
	Short: "sign in to the Quollix App Store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		}
		if password == "" {
			password, err = Dependencies.OsWrapper.PromptSecret("Password: ")
			if err != nil {
				return err
			}
		}
		if err := Dependencies.AppStoreClient.Login(username, password); err != nil {
			return err
		}
		details, err := Dependencies.AppStoreClient.GetAccountDetails()
		if err != nil {
			return err
		}
		config, err := Dependencies.ConfigProvider.GetConfig()
		if err != nil {
			return err
		}
		session := configuration.SessionData{
			Maintainer:     username,
			Cookie:         Dependencies.AppStoreClient.Parent.Cookie.Value,
			ExpirationDate: Dependencies.AppStoreClient.Parent.Cookie.Expires,
		}
		config.Session = &session
		if err := Dependencies.ConfigProvider.SetConfig(config); err != nil {
			return err
		}
		if err := Dependencies.SigningKeyManager.StorePublicKeyRaw(details.PublicKeyRaw); err != nil {
			return err
		}
		fmt.Println("sign-in successful")
		return nil
	},
}

func promptForConfirmedPassword() (string, error) {
	password, err := Dependencies.OsWrapper.PromptSecret("Password: ")
	if err != nil {
		return "", err
	}
	confirmation, err := Dependencies.OsWrapper.PromptSecret("Confirm password: ")
	if err != nil {
		return "", err
	}
	if password != confirmation {
		return "", u.Logger.NewError("password confirmation did not match")
	}
	return password, nil
}

func promptForPasswordChange() (string, string, error) {
	oldPassword, err := Dependencies.OsWrapper.PromptSecret("Current password: ")
	if err != nil {
		return "", "", err
	}
	newPassword, err := Dependencies.OsWrapper.PromptSecret("New password: ")
	if err != nil {
		return "", "", err
	}
	confirmation, err := Dependencies.OsWrapper.PromptSecret("Confirm new password: ")
	if err != nil {
		return "", "", err
	}
	if newPassword != confirmation {
		return "", "", u.Logger.NewError("password confirmation did not match")
	}
	return oldPassword, newPassword, nil
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "logs out and clears the locally stored session cookie",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := Dependencies.AppStoreClient.Logout(); err != nil {
			return err
		}
		if err := Dependencies.SessionManager.DeleteSession(); err != nil {
			return err
		}
		fmt.Println("logout successful")
		return nil
	},
}

var deleteOwnAccountCmd = &cobra.Command{
	Use:   "delete-account",
	Short: "deletes your own account, including all associated apps and versions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := promptMaintainerToDeletionConfirmOperation("account deletion"); err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.DeleteOwnAccount(); err != nil {
			return err
		}
		fmt.Println("account deletion successful")
		return nil
	},
}

var changePasswordCmd = &cobra.Command{
	Use:   "change-password",
	Short: "change your account password",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		oldPassword, newPassword, err := promptForPasswordChange()
		if err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.ChangePassword(oldPassword, newPassword); err != nil {
			return err
		}
		fmt.Println("password change successful")
		return nil
	},
}

var accountDetailsCmd = &cobra.Command{
	Use:   "details",
	Short: "get details about your account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		details, err := Dependencies.AppStoreClient.GetAccountDetails()
		if err != nil {
			return err
		}
		if err := Dependencies.SessionManager.PersistCurrentSession(); err != nil {
			return err
		}
		if err := Dependencies.SigningKeyManager.StorePublicKeyRaw(details.PublicKeyRaw); err != nil {
			return err
		}
		printUserDetails(details)
		return nil
	},
}

func printUserDetails(details *store.AccountDetails) {
	spaceUsedPercentage := 0.0
	if details.StorageLimitInBytes > 0 {
		spaceUsedPercentage = float64(details.UsedSpaceInBytes) / float64(details.StorageLimitInBytes) * 100
	}
	remainingSpaceInBytes := max(details.StorageLimitInBytes-details.UsedSpaceInBytes, 0)
	role := "maintainer"
	if details.IsAdmin {
		role = "admin"
	}
	s := fmt.Sprintf(
		`User: %s
Email: %s
Role: %s
Session expires: %s (UTC, %s)
Space used: %.2f%% (%s / %s, %s remaining)
Signing key fingerprint: %s`,
		details.Name,
		details.Email,
		role,
		details.CookieExpirationDate.UTC().Format("2006-01-02 15:04:05"),
		u.FormatRelativeDuration(time.Now().UTC(), details.CookieExpirationDate),
		spaceUsedPercentage,
		formatBytes(details.UsedSpaceInBytes),
		formatBytes(details.StorageLimitInBytes),
		formatBytes(remainingSpaceInBytes),
		getPublicKeyFingerprint(details.PublicKeyRaw),
	)
	fmt.Println(s)
}

func getPublicKeyFingerprint(publicKey []byte) string {
	sshPublicKey, err := ssh.NewPublicKey(ed25519.PublicKey(publicKey))
	if err != nil {
		return "invalid public key"
	}
	return ssh.FingerprintSHA256(sshPublicKey)
}

func formatBytes(numberOfBytes int64) string {
	const kb = 1024.0
	const mb = kb * 1024.0
	displayValue := float64(numberOfBytes)
	switch {
	case displayValue < kb:
		return fmt.Sprintf("%.0f B", displayValue)
	case displayValue < mb:
		return fmt.Sprintf("%.2f KB", displayValue/kb)
	default:
		return fmt.Sprintf("%.2f MB", displayValue/mb)
	}
}

var changeEmailCmd = &cobra.Command{
	Use:   "change-email <new-email>",
	Short: "change your account email address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newEmail := args[0]
		if err := Dependencies.AppStoreClient.ChangeEmail(newEmail); err != nil {
			return err
		}
		fmt.Println("email change successful")
		return nil
	},
}
