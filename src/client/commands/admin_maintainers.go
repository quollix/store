package commands

import (
	"fmt"

	"qsc/remote"

	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var adminMaintainerCmd = &cobra.Command{
	Use:   "maintainer",
	Short: "manage app maintainer accounts",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var createMaintainerCmd = &cobra.Command{
	Use:   "create <name> <email> <ssh-public-key>",
	Short: "create an app maintainer account",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, email, publicKey := args[0], args[1], args[2]
		publicKeyRaw, err := remote.DecodeAuthorizedEd25519PublicKey([]byte(publicKey))
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
		privateKeyOpenSSH, err := Dependencies.OsWrapper.ReadFile(privateKeyPath)
		if err != nil {
			return u.Logger.NewError(err.Error())
		}
		publicKeySignature, err := remote.SignMaintainerPublicKey(privateKeyOpenSSH, privateKeyPassphrase, name, publicKeyRaw)
		if err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.CreateMaintainerByAdmin(name, email, publicKeyRaw, publicKeySignature); err != nil {
			return err
		}
		fmt.Println("app maintainer created successfully")
		return nil
	},
}

var deleteMaintainerCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "delete an app maintainer account and remove their apps",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := promptMaintainerToDeletionConfirmOperation("app maintainer deletion"); err != nil {
			return err
		}
		if err := Dependencies.AppStoreClient.DeleteMaintainerByAdmin(args[0]); err != nil {
			return err
		}
		fmt.Println("app maintainer deletion successful")
		return nil
	},
}
