package commands

import (
	"encoding/base64"
	"fmt"
	"strings"

	"qsc/remote"
	"qsc/tools"

	u "github.com/quollix/common/utils"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "manage the locally stored auth session",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete the locally stored auth session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := Dependencies.SessionManager.DeleteSession(); err != nil {
			return err
		}
		fmt.Println("local auth session deleted successfully")
		return nil
	},
}

var sessionShowCmd = &cobra.Command{
	Use:   "show",
	Short: "print the locally stored auth session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := Dependencies.SessionManager.GetConfig()
		if err != nil {
			return err
		}
		fmt.Print(renderSessionConfig(config, Dependencies.Config))
		return nil
	},
}

func renderSessionConfig(config *remote.LocalConfig, globalConfig *tools.GlobalConfig) string {
	var builder strings.Builder
	if globalConfig != nil {
		fmt.Fprintf(&builder, "- config_file: %s\n", globalConfig.ConfigFilePath)
		fmt.Fprintf(&builder, "- server_url: %s\n", globalConfig.AppStoreRootURL)
	}
	builder.WriteString("- session:\n")
	if config.Session == nil {
		builder.WriteString("  - not configured\n")
	} else {
		fmt.Fprintf(&builder, "  - maintainer: %s\n", config.Session.Maintainer)
		fmt.Fprintf(&builder, "  - expiration_date: %s\n", config.Session.ExpirationDate.UTC().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&builder, "  - access_cookie: %s\n", maskSecret(config.Session.Cookie))
	}
	renderSigningConfig(&builder, config)
	renderDockerHubConfigSection(&builder, config)
	fmt.Fprintf(&builder, "- trusted_maintainer_keys:\n  - count: %d\n", trustedMaintainerKeyCount(config))
	return builder.String()
}

func renderSigningConfig(builder *strings.Builder, config *remote.LocalConfig) {
	builder.WriteString("- signing:\n")
	if config.Signing == nil {
		builder.WriteString("  - not configured\n")
		return
	}
	fmt.Fprintf(builder, "  - public_key_configured: %t\n", config.Signing.PublicKeyRawBase64 != "")
	fmt.Fprintf(builder, "  - public_key_fingerprint: %s\n", getPublicKeyFingerprintFromBase64(config.Signing.PublicKeyRawBase64))
	fmt.Fprintf(builder, "  - private_key_path: %s\n", config.Signing.PrivateKeyPath)
}

func renderDockerHubConfigSection(builder *strings.Builder, config *remote.LocalConfig) {
	builder.WriteString("- docker_hub:\n")
	if config.DockerHub == nil {
		builder.WriteString("  - not configured\n")
		return
	}
	fmt.Fprintf(builder, "  - username: %s\n", config.DockerHub.Username)
	fmt.Fprintf(builder, "  - token: %s\n", maskSecret(config.DockerHub.Token))
}

func getPublicKeyFingerprintFromBase64(publicKeyRawBase64 string) string {
	if publicKeyRawBase64 == "" {
		return ""
	}
	publicKeyRaw, err := base64.StdEncoding.DecodeString(publicKeyRawBase64)
	if err != nil {
		return "invalid public key"
	}
	return getPublicKeyFingerprint(publicKeyRaw)
}

func trustedMaintainerKeyCount(config *remote.LocalConfig) int {
	return len(config.TrustedMaintainerKeys)
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
}
