package commands

import (
	"fmt"
	"math"
	"strconv"

	"qsc/remote"

	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

const bytesPerMegaByte = 1024 * 1024

var adminMaintainersCmd = &cobra.Command{
	Use:   "maintainers",
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

var listMaintainersCmd = &cobra.Command{
	Use:   "list",
	Short: "list app maintainer accounts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		maintainers, err := Dependencies.AppStoreClient.ListMaintainersByAdmin()
		if err != nil {
			return err
		}
		fmt.Print(renderMaintainersTable(maintainers))
		return nil
	},
}

var setMaintainerSpaceCmd = &cobra.Command{
	Use:   "set-space <name> <megabytes>",
	Short: "set an app maintainer's storage limit",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		megaBytes, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil || megaBytes > math.MaxInt64/bytesPerMegaByte {
			return u.Logger.NewError("storage limit must be a non-negative whole number of megabytes")
		}
		if err := Dependencies.AppStoreClient.SetMaintainerStorageLimitByAdmin(args[0], int64(megaBytes)*bytesPerMegaByte); err != nil {
			return err
		}
		fmt.Println("app maintainer storage limit updated successfully")
		return nil
	},
}
