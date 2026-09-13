package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"qsc/remote"
	"qsc/tools"

	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

const (
	devBootstrapAdminUsername = "quollix"
	devBootstrapAdminPassword = "password"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "manage local development helpers",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var devBootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "initialize qsc TEST profile with sample credentials",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("PROFILE") != "TEST" {
			return u.Logger.NewError("dev bootstrap is only available with PROFILE=TEST")
		}

		configDir := filepath.Dir(Dependencies.Config.ConfigFilePath)
		if err := Dependencies.OsWrapper.RemoveAll(configDir); err != nil {
			return err
		}
		if err := Dependencies.OsWrapper.MkdirAll(configDir, 0o700); err != nil {
			return err
		}

		privateKeyPath := filepath.Join(configDir, "sample-store-private-key")
		if err := Dependencies.OsWrapper.WriteFile(privateKeyPath, []byte(u.LocalTestingPrivateKeyOpenSSH), 0o600); err != nil {
			return err
		}

		if err := Dependencies.AppStoreClient.Login(devBootstrapAdminUsername, devBootstrapAdminPassword); err != nil {
			return err
		}
		details, err := Dependencies.AppStoreClient.GetAccountDetails()
		if err != nil {
			return err
		}
		config := &remote.LocalConfig{
			Session: &remote.SessionData{
				Maintainer:     devBootstrapAdminUsername,
				Cookie:         Dependencies.AppStoreClient.Parent.Cookie.Value,
				ExpirationDate: Dependencies.AppStoreClient.Parent.Cookie.Expires,
			},
			DockerHub: &tools.DockerHubAuth{
				Username: tools.SampleDockerHubAuth.Username,
				Token:    tools.SampleDockerHubAuth.Token,
			},
		}
		if err := Dependencies.SessionManager.SetConfig(config); err != nil {
			return err
		}
		if err := Dependencies.SigningKeyManager.StorePublicKeyRaw(details.PublicKeyRaw); err != nil {
			return err
		}
		if err := Dependencies.SigningKeyManager.SetPrivateKeyPath(privateKeyPath, u.LocalTestingPrivateKeyPassphrase); err != nil {
			return err
		}

		fmt.Println("TEST profile bootstrapped for " + devBootstrapAdminUsername)
		return nil
	},
}
