//go:build terminal

package tests

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"qsc/commands"
	"qsc/configuration"
	"qsc/di"
	"qsc/local"
	"qsc/remote"
	"qsc/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
	"golang.org/x/crypto/ssh"

	"github.com/quollix/common/assert"
)

const (
	nginxComposePath                    = "../assets/sample-store/v1/nginx.yml"
	terminalAdminMaintainer             = "quollix"
	terminalAdminEmail                  = "admin@quollix.org"
	terminalAdminPassword               = "password"
	terminalNginxApp                    = "nginx"
	terminalCaddyApp                    = "caddy"
	terminalUploadedNginxVersion        = "1.0.0"
	terminalSelectedNginxVersion        = "1.0.1"
	terminalSelectedCaddyVersion        = "2.10.0"
	defaultRegistrationConfirmationCode = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	storeAccountCommand                 = "account"
	storeAppsCommand                    = "apps"
	storeVersionsCommand                = "versions"
	storeAdminCommand                   = "admin"
	storeAdminMaintainersCommand        = "admin maintainers"
	storeDevCommand                     = "dev"
	storeLocalCommand                   = "local"
	storeOnboardingCommand              = "onboarding"
	storeSessionCommand                 = "session"
)

type terminalArtifacts struct {
	adminPrivateKeyPath      string
	maintainerPrivateKeyPath string
	maintainerPublicKey      string
	appsDir                  string
	composePath              string
}

type terminalSetup struct {
	artifacts    terminalArtifacts
	dependencies *commands.ClientDependencies
	storeClient  *store.AppStoreClientImpl
}

func TestAccountSessionTerminalWorkflow(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()

	outputString := runClientCommand(t, storeAccountCommand+" sign-in %s -p %s", tools.SampleMaintainer, tools.SamplePassword)
	assert.True(t, strings.Contains(outputString, "sign-in successful"))

	config, err := setup.dependencies.ConfigProvider.GetConfig()
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleMaintainer, config.Session.Maintainer)

	outputString = runClientCommand(t, storeSessionCommand+" show")
	assert.True(t, strings.Contains(outputString, "- maintainer: "+tools.SampleMaintainer))
	assert.True(t, strings.Contains(outputString, "- access_cookie: "+maskTerminalSecret(config.Session.Cookie)))
	assert.False(t, strings.Contains(outputString, config.Session.Cookie))

	outputString = runClientCommandWithInput(t, u.OtherLocalTestingPrivateKeyPassphrase+"\n", storeOnboardingCommand+" set-private-key %s", setup.artifacts.maintainerPrivateKeyPath)
	assert.True(t, strings.Contains(outputString, "private key path saved successfully"))

	config, err = setup.dependencies.ConfigProvider.GetConfig()
	assert.Nil(t, err)
	assert.Equal(t, setup.artifacts.maintainerPrivateKeyPath, config.Signing.PrivateKeyPath)

	outputString = runClientCommand(t, storeSessionCommand+" show")
	assert.True(t, strings.Contains(outputString, "- private_key_path: "+setup.artifacts.maintainerPrivateKeyPath))
	assert.True(t, strings.Contains(outputString, "- public_key_configured: true"))

	outputString = runClientCommand(t, storeAccountCommand+" details")
	assert.True(t, strings.Contains(outputString, "User: "+tools.SampleMaintainer))
	assert.True(t, strings.Contains(outputString, "Email: "+tools.SampleEmail))

	outputString = runClientCommand(t, storeAccountCommand+" logout")
	assert.True(t, strings.Contains(outputString, "logout successful"))

	config, err = setup.dependencies.ConfigProvider.GetConfig()
	assert.Nil(t, err)
	assert.Nil(t, config.Session)

	outputString = runClientCommandWithInputExpectingError(t, "", storeAccountCommand+" details")
	assert.True(t, strings.Contains(outputString, "cookie is not set in request, please sign in"))

	outputString = runClientCommandWithInputExpectingError(t, "", storeAccountCommand+" logout")
	assert.True(t, strings.Contains(outputString, "cookie is not set in request, please sign in"))
}

func TestDevBootstrapTerminalWorkflow(t *testing.T) {
	setupTerminalArtifacts(t)
	dependencies := setupTerminalDependencies(t)

	outputString := runClientCommand(t, storeDevCommand+" bootstrap")
	assert.Equal(t, "TEST profile bootstrapped for "+terminalAdminMaintainer+"\n", outputString)

	config, err := dependencies.ConfigProvider.GetConfig()
	assert.Nil(t, err)
	assert.NotNil(t, config.Session)
	assert.Equal(t, terminalAdminMaintainer, config.Session.Maintainer)
	assert.NotEqual(t, "", config.Session.Cookie)
	assert.True(t, time.Now().UTC().Before(config.Session.ExpirationDate))
	assert.NotNil(t, config.Signing)
	expectedPrivateKeyPath := filepath.Join(filepath.Dir(dependencies.Config.ConfigFilePath), "sample-store-private-key")
	assert.Equal(t, expectedPrivateKeyPath, config.Signing.PrivateKeyPath)
	assert.Equal(t, base64.StdEncoding.EncodeToString(u.GetLocalTestingPublicKeyRaw()), config.Signing.PublicKeyRawBase64)
	assert.Equal(t, []byte(u.LocalTestingPrivateKeyOpenSSH), mustReadFile(t, expectedPrivateKeyPath))
	assert.Nil(t, dependencies.SigningKeyManager.ValidateConfiguredPrivateKey(u.LocalTestingPrivateKeyPassphrase))
	assert.NotNil(t, config.DockerHub)
	assert.Equal(t, tools.SampleDockerHubAuth.Username, config.DockerHub.Username)
	assert.Equal(t, tools.SampleDockerHubAuth.Token, config.DockerHub.Token)
	downloadedNginxContent := string(mustReadFile(t, filepath.Join(config.AppsDirectory, terminalNginxApp+".yml")))
	assert.True(t, strings.Contains(downloadedNginxContent, "image: nginx:1.0.2"))

	outputString = runClientCommand(t, storeLocalCommand+" docker-hub show")
	assert.True(t, strings.Contains(outputString, "- username: "+tools.SampleDockerHubAuth.Username))
	assert.True(t, strings.Contains(outputString, "- token: "+maskTerminalSecret(tools.SampleDockerHubAuth.Token)))
	assert.False(t, strings.Contains(outputString, tools.SampleDockerHubAuth.Token))

	outputString = runClientCommand(t, storeAppsCommand+" list")
	assert.Equal(t, terminalNginxApp+"\n", outputString)
}

func TestLocalUpdateTerminalWorkflow(t *testing.T) {
	artifacts := setupTerminalArtifacts(t)
	setTerminalAppsDirectory(t, setupTerminalDependencies(t), artifacts.appsDir)

	outputString := runClientCommand(t, storeLocalCommand+" update nginx")
	assert.True(t, strings.Contains(outputString, "nginx: 1.0.0 -> "+local.NewNginxTag))
	assert.True(t, strings.Contains(outputString, "overall update successful"))

	currentComposeContent := string(mustReadFile(t, artifacts.composePath))
	assert.True(t, strings.Contains(currentComposeContent, "image: nginx:"+local.NewNginxTag+"@"+local.NewNginxDigest))
	assert.False(t, strings.Contains(currentComposeContent, "image: nginx:1.0.0"))
}

func TestLocalValidateTerminalWorkflow(t *testing.T) {
	artifacts := setupTerminalArtifacts(t)
	setTerminalAppsDirectory(t, setupTerminalDependencies(t), artifacts.appsDir)
	validComposeContent := terminalVersionContentForMaintainerApp(t, tools.OfficialMaintainer, terminalNginxApp, mustReadFile(t, artifacts.composePath))
	assert.Nil(t, os.WriteFile(artifacts.composePath, validComposeContent, 0o644))

	outputString := runClientCommand(t, storeLocalCommand+" validate")
	assert.True(t, strings.Contains(outputString, "consistency check successful"))

	invalidComposePath := writeTerminalComposeFile(t, artifacts.appsDir, tools.OfficialMaintainer, terminalCaddyApp, "caddy", local.OldCaddyTag)
	invalidComposeContent := []byte(strings.Replace(string(mustReadFile(t, invalidComposePath)), "container_name: quollix_caddy_caddy", "container_name: wrong_caddy_caddy", 1))
	assert.Nil(t, os.WriteFile(invalidComposePath, invalidComposeContent, 0o644))

	outputString = runClientCommand(t, storeLocalCommand+" validate nginx")
	assert.True(t, strings.Contains(outputString, "consistency check successful"))

	outputString = runClientCommandWithInputExpectingError(t, "", storeLocalCommand+" validate")
	assert.True(t, strings.Contains(outputString, "service has invalid container_name"))
}

func TestLocalListAndCacheTerminalWorkflow(t *testing.T) {
	artifacts := setupTerminalArtifacts(t)
	dependencies := setupTerminalDependencies(t)
	_ = os.Remove(dependencies.Config.ImageTagCachePath)

	outputString := runClientCommandWithInputExpectingError(t, "", storeLocalCommand+" list")
	assert.True(t, strings.Contains(outputString, "apps directory is not configured"))

	outputString = runClientCommand(t, storeOnboardingCommand+" set-apps-directory %s", artifacts.appsDir)
	assert.Equal(t, "apps directory saved successfully\n", outputString)

	outputString = runClientCommand(t, storeLocalCommand+" list")
	assert.Equal(t, "nginx\n", outputString)

	outputString = runClientCommand(t, storeLocalCommand+" cache")
	assert.Equal(t, "image tag cache is empty\n", outputString)
}

func TestLocalDockerHubTerminalWorkflow(t *testing.T) {
	setupTerminalArtifacts(t)
	dependencies := setupTerminalDependencies(t)
	token := "terminal-docker-hub-token"

	outputString := runClientCommandWithInput(t, token+"\n", storeLocalCommand+" docker-hub set terminal-user")
	assert.True(t, strings.Contains(outputString, "Docker Hub authentication saved successfully"))

	config, err := dependencies.ConfigProvider.GetConfig()
	assert.Nil(t, err)
	assert.NotNil(t, config.DockerHub)
	assert.Equal(t, "terminal-user", config.DockerHub.Username)
	assert.Equal(t, token, config.DockerHub.Token)

	outputString = runClientCommand(t, storeLocalCommand+" docker-hub show")
	assert.Equal(t, "- username: terminal-user\n- token: "+maskTerminalSecret(token)+"\n", outputString)
	assert.False(t, strings.Contains(outputString, token))

	outputString = runClientCommand(t, storeLocalCommand+" docker-hub delete")
	assert.True(t, strings.Contains(outputString, "Docker Hub authentication deleted successfully"))

	config, err = dependencies.ConfigProvider.GetConfig()
	assert.Nil(t, err)
	assert.Nil(t, config.DockerHub)

	outputString = runClientCommand(t, storeLocalCommand+" docker-hub show")
	assert.Equal(t, "- not configured\n", outputString)
}

func TestAccountMaintenanceTerminalWorkflow(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()

	newPassword := "newpassword"
	newEmail := "new-" + tools.SampleEmail

	outputString := runClientCommandWithInput(t, tools.SamplePassword+"\n"+newPassword+"\n"+newPassword+"\n", storeAccountCommand+" change-password")
	assert.True(t, strings.Contains(outputString, "password change successful"))

	freshClient := setupTerminalDependencies(t).AppStoreClient
	assert.Nil(t, freshClient.Login(tools.SampleMaintainer, newPassword))

	outputString = runClientCommand(t, storeAccountCommand+" sign-in %s -p %s", tools.SampleMaintainer, newPassword)
	assert.True(t, strings.Contains(outputString, "sign-in successful"))

	outputString = runClientCommand(t, storeAccountCommand+" change-email %s", newEmail)
	assert.True(t, strings.Contains(outputString, "email change successful"))

	freshClient = setupTerminalDependencies(t).AppStoreClient
	assert.Nil(t, freshClient.Login(tools.SampleMaintainer, newPassword))
	accountDetails, err := freshClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, newEmail, accountDetails.Email)
}

func TestAccountDeletionTerminalWorkflow(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()

	assert.Nil(t, uploadSignedTerminalVersion(setup.dependencies, "nginx", "1.27.4", mustReadFile(t, setup.artifacts.composePath)))

	outputString := runClientCommandWithInput(t, "DELETE\n", storeAccountCommand+" delete-account")
	assert.True(t, strings.Contains(outputString, "account deletion successful"))

	publicClient := setupTerminalDependencies(t).AppStoreClient
	apps, err := publicClient.SearchForApps(tools.SampleMaintainer, "nginx", true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
	assert.NotNil(t, publicClient.Login(tools.SampleMaintainer, tools.SamplePassword))
}

func TestAppsTerminalWorkflow(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()

	apps, err := setup.storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))

	outputString := runClientCommand(t, storeAppsCommand+" create nginx")
	assert.True(t, strings.Contains(outputString, "app created successfully"))

	outputString = runClientCommand(t, storeAppsCommand+" list")
	assert.Equal(t, "nginx\n", outputString)

	outputString = runClientCommandWithInput(t, "DELETE\n", storeAppsCommand+" delete nginx")
	assert.True(t, strings.Contains(outputString, "app deleted successfully"))

	apps, err = setup.storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
}

func TestVersionsTerminalWorkflow(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()
	assert.Nil(t, setup.storeClient.CreateApp(terminalNginxApp))
	assert.Nil(t, setup.storeClient.CreateApp("emptyapp"))

	// The local file deliberately contains CRLF line endings; CLI upload must normalize before signing.
	localContentWithCarriageReturns := bytes.ReplaceAll(mustReadFile(t, setup.artifacts.composePath), []byte("\n"), []byte("\r\n"))
	assert.True(t, bytes.Contains(localContentWithCarriageReturns, []byte("\r")))
	assert.Nil(t, os.WriteFile(setup.artifacts.composePath, localContentWithCarriageReturns, 0o644))
	normalizedContent := bytes.ReplaceAll(localContentWithCarriageReturns, []byte("\r\n"), []byte("\n"))

	outputString := runClientCommandWithInput(t, u.OtherLocalTestingPrivateKeyPassphrase+"\n", storeLocalCommand+" upload")
	assert.True(t, strings.Contains(outputString, terminalNginxApp+": uploaded version "+terminalUploadedNginxVersion))

	outputString = runClientCommand(t, storeAppsCommand+" search '' nginx")
	assert.True(t, strings.Contains(outputString, "maintainer"))
	assert.True(t, strings.Contains(outputString, "app"))
	assert.True(t, regexp.MustCompile(regexp.QuoteMeta(tools.SampleMaintainer)+` +nginx`).MatchString(outputString))

	outputString = runClientCommand(t, storeVersionsCommand+" list %s nginx", tools.SampleMaintainer)
	assert.True(t, strings.Contains(outputString, "index  version"))
	assert.True(t, strings.Contains(outputString, terminalUploadedNginxVersion))
	assert.True(t, regexp.MustCompile(`false +0`).MatchString(outputString))

	versions, err := setup.storeClient.ListVersions(tools.SampleMaintainer, terminalNginxApp)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(versions))
	assert.Equal(t, terminalUploadedNginxVersion, versions[0].Name)
	uploadedVersion, err := setup.storeClient.DownloadVersionByID(versions[0].VersionId)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleMaintainer, uploadedVersion.Maintainer)
	assert.Equal(t, terminalNginxApp, uploadedVersion.AppName)
	assert.Equal(t, terminalUploadedNginxVersion, uploadedVersion.VersionName)
	assert.False(t, bytes.Contains(uploadedVersion.Content, []byte("\r")))
	assert.Equal(t, normalizedContent, uploadedVersion.Content)

	outputString = runClientCommandWithInput(t, "0\n", storeVersionsCommand+" checkpoint mark nginx")
	assert.True(t, strings.Contains(outputString, "migration checkpoint marked successfully"))

	outputString = runClientCommand(t, storeVersionsCommand+" list %s nginx", tools.SampleMaintainer)
	assert.True(t, regexp.MustCompile(`true +1`).MatchString(outputString))

	outputString = runClientCommandWithInput(t, "0\n", storeVersionsCommand+" checkpoint unmark nginx")
	assert.True(t, strings.Contains(outputString, "migration checkpoint unmarked successfully"))

	downloadDir := t.TempDir()
	outputString = runClientCommandWithInput(t, "DELETE\n", storeVersionsCommand+" clone %s", downloadDir)
	assert.True(t, strings.Contains(outputString, "versions cloned successfully: 1 apps"))

	outputString = runClientCommand(t, storeVersionsCommand+" list %s nginx", tools.SampleMaintainer)
	assert.True(t, strings.Contains(outputString, terminalUploadedNginxVersion))
	assert.True(t, regexp.MustCompile(`false +2`).MatchString(outputString))

	downloadedContent, err := os.ReadFile(filepath.Join(downloadDir, "nginx.yml"))
	assert.Nil(t, err)
	assert.Equal(t, normalizedContent, downloadedContent)
	assertTerminalFileDoesNotExist(t, filepath.Join(downloadDir, "emptyapp.yml"))

	outputString = runClientCommandWithInput(t, "0\nDELETE\n", storeVersionsCommand+" delete nginx")
	assert.True(t, strings.Contains(outputString, "version deleted successfully"))

	versions, err = setup.storeClient.ListVersions(tools.SampleMaintainer, terminalNginxApp)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(versions))
}

func TestUploadVersionsTerminalSelectsAppsByArgument(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()
	nginxComposePath := writeTerminalComposeFile(t, setup.artifacts.appsDir, tools.SampleMaintainer, terminalNginxApp, "nginx", terminalSelectedNginxVersion)
	caddyComposePath := writeTerminalComposeFile(t, setup.artifacts.appsDir, tools.SampleMaintainer, terminalCaddyApp, "caddy", local.OldCaddyTag)
	writeTerminalComposeFile(t, setup.artifacts.appsDir, tools.SampleMaintainer, "skipped", "nginx", "1.0.2")

	outputString := runClientCommandWithInput(t, u.OtherLocalTestingPrivateKeyPassphrase+"\n", storeLocalCommand+" upload nginx caddy")
	assert.True(t, strings.Contains(outputString, terminalNginxApp+": uploaded version "+terminalSelectedNginxVersion))
	assert.True(t, strings.Contains(outputString, terminalCaddyApp+": uploaded version "+terminalSelectedCaddyVersion))
	assert.False(t, strings.Contains(outputString, "skipped"))

	apps, err := setup.storeClient.ListOwnApps()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(apps))
	assert.True(t, containsTerminalString(apps, terminalNginxApp))
	assert.True(t, containsTerminalString(apps, terminalCaddyApp))
	assert.False(t, containsTerminalString(apps, "skipped"))

	assertTerminalUploadedVersion(t, setup.storeClient, terminalNginxApp, terminalSelectedNginxVersion, mustReadFile(t, nginxComposePath))
	assertTerminalUploadedVersion(t, setup.storeClient, terminalCaddyApp, terminalSelectedCaddyVersion, mustReadFile(t, caddyComposePath))

	assert.Nil(t, os.WriteFile(caddyComposePath, []byte("services:"), 0o644))
	outputString = runClientCommandWithInputExpectingError(t, u.OtherLocalTestingPrivateKeyPassphrase+"\n", storeLocalCommand+" upload nginx caddy")
	assert.True(t, strings.Contains(outputString, terminalCaddyApp+": upload failed"))
	assert.True(t, strings.Contains(outputString, terminalNginxApp+": store already has this content"))
	assert.True(t, strings.Contains(outputString, "app upload failed"))
	assert.True(t, strings.Contains(outputString, "one or more app uploads failed"))
}

func TestDeleteVersionTerminalSelectsDuplicateVersionByIndex(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()
	assert.Nil(t, setup.storeClient.CreateApp("nginx"))

	baseTimestamp := time.Now().UTC().Add(-5 * time.Minute)
	_, err := uploadSignedTerminalVersionAt(setup.dependencies, "nginx", "1.27.4", baseTimestamp, nginxComposeContentWithTag(t, "1.27.4"))
	assert.Nil(t, err)
	newerVersion, err := uploadSignedTerminalVersionAt(setup.dependencies, "nginx", "1.27.4", baseTimestamp.Add(time.Minute), nginxComposeContentWithTag(t, "1.27.5"))
	assert.Nil(t, err)

	outputString := runClientCommandWithInput(t, "0\nDELETE\n", storeVersionsCommand+" delete nginx")
	assert.True(t, strings.Contains(outputString, "index  version"))
	assert.True(t, regexp.MustCompile(`0 +1\.27\.4`).MatchString(outputString))
	assert.True(t, regexp.MustCompile(`1 +1\.27\.4`).MatchString(outputString))
	assert.True(t, strings.Contains(outputString, fmt.Sprintf("Type DELETE to confirm deletion of app 'nginx' version '1.27.4' from %s:", formatTerminalTimestamp(baseTimestamp))))
	assert.True(t, strings.Contains(outputString, "version deleted successfully"))

	versions, err := setup.storeClient.ListVersions(tools.SampleMaintainer, "nginx")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(versions))
	assert.Equal(t, newerVersion.VersionId, versions[0].VersionId)
}

func TestDeleteVersionTerminalRejectsInvalidDuplicateVersionIndex(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()
	assert.Nil(t, setup.storeClient.CreateApp("nginx"))

	baseTimestamp := time.Now().UTC().Add(-5 * time.Minute)
	_, err := uploadSignedTerminalVersionAt(setup.dependencies, "nginx", "1.27.4", baseTimestamp, nginxComposeContentWithTag(t, "1.27.4"))
	assert.Nil(t, err)
	_, err = uploadSignedTerminalVersionAt(setup.dependencies, "nginx", "1.27.4", baseTimestamp.Add(time.Minute), nginxComposeContentWithTag(t, "1.27.5"))
	assert.Nil(t, err)

	outputString := runClientCommandWithInputExpectingError(t, "999999\n", storeVersionsCommand+" delete nginx")
	assert.True(t, strings.Contains(outputString, "selected version index does not match listed versions"))

	versions, err := setup.storeClient.ListVersions(tools.SampleMaintainer, "nginx")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(versions))
}

func TestShowAndDiffVersionTerminal(t *testing.T) {
	setup := setupAuthenticatedTerminalMaintainer(t)
	defer setup.storeClient.WipeData()
	assert.Nil(t, setup.storeClient.CreateApp("nginx"))

	baseTimestamp := time.Now().UTC().Add(-5 * time.Minute)
	_, err := uploadSignedTerminalVersionAt(setup.dependencies, "nginx", "1.27.4", baseTimestamp, nginxComposeContentWithTag(t, "1.27.4"))
	assert.Nil(t, err)
	_, err = uploadSignedTerminalVersionAt(setup.dependencies, "nginx", "1.27.4", baseTimestamp.Add(time.Minute), nginxComposeContentWithTag(t, "1.27.5"))
	assert.Nil(t, err)
	_, err = uploadSignedTerminalVersionAt(setup.dependencies, "nginx", "1.27.4", baseTimestamp.Add(2*time.Minute), nginxComposeContentWithTag(t, "1.27.6"))
	assert.Nil(t, err)

	outputString := runClientCommandWithInput(t, "2\n", storeVersionsCommand+" content %s nginx", tools.SampleMaintainer)
	assert.True(t, strings.Contains(outputString, "index  version"))
	assert.True(t, regexp.MustCompile(`0 +1\.27\.4`).MatchString(outputString))
	assert.True(t, strings.Contains(outputString, "Enter version index to print content:"))
	assert.True(t, strings.Contains(outputString, validation.AppDefinitionLicenseNotice))
	assert.True(t, strings.Contains(outputString, "services:"))
	assert.True(t, strings.Contains(outputString, "image: nginx:1.27.6-alpine"))

	outputString = runClientCommandWithInput(t, "2 1\n", storeVersionsCommand+" diff %s nginx", tools.SampleMaintainer)
	assert.True(t, regexp.MustCompile(`1 +1\.27\.4`).MatchString(outputString))
	assert.True(t, regexp.MustCompile(`2 +1\.27\.4`).MatchString(outputString))
	assert.True(t, strings.Contains(outputString, fmt.Sprintf(`%s-        image: nginx:1.27.6-alpine%s
%s+        image: nginx:1.27.5-alpine%s
`, tools.AnsiRed, tools.AnsiReset, tools.AnsiGreen, tools.AnsiReset)))
	assert.True(t, strings.Contains(outputString, "...\n"))
}

func TestAdminEmailTerminalWorkflow(t *testing.T) {
	dependencies := setupTerminalDependencies(t)
	storeClient := dependencies.AppStoreClient
	defer storeClient.WipeData()
	assert.Nil(t, loginWithoutSigning(dependencies.ConfigProvider, storeClient, terminalAdminMaintainer, terminalAdminPassword))

	sampleEmailConfig := u.SampleEmailConfig
	outputString := runClientCommandWithInput(
		t,
		sampleEmailConfig.EmailAccountPassword+"\n",
		storeAdminCommand+" email set-config %s %s %s %s",
		sampleEmailConfig.SMTPHost,
		sampleEmailConfig.SMTPPort,
		sampleEmailConfig.FromEmailAddress,
		sampleEmailConfig.EmailAccountUsername,
	)
	assert.True(t, strings.Contains(outputString, "email config set successfully"))

	outputString = runClientCommand(t, storeAdminCommand+" email get-config")
	assert.Equal(t, fmt.Sprintf("smtp-host: %s\nsmtp-port: %s\nemail-address: %s\nemail-account-username: %s\n", sampleEmailConfig.SMTPHost, sampleEmailConfig.SMTPPort, sampleEmailConfig.FromEmailAddress, sampleEmailConfig.EmailAccountUsername), outputString)

	testEmailBodyFile := writeTerminalEmailBodyFile(t, "test-email-body.txt", u.SampleTestEmail.Body)
	outputString = runClientCommand(t, storeAdminCommand+" email send-test %q %q", u.SampleTestEmail.Subject, testEmailBodyFile)
	assert.Equal(t, "emails sent: 1\nemails failed: 0\n", outputString)

	err := createMaintainerByAdmin(storeClient, tools.SampleMaintainer, tools.SampleEmail, u.GetOtherLocalTestingPublicKeyRaw())
	assert.Nil(t, err)

	maintainersEmailBodyFile := writeTerminalEmailBodyFile(t, "maintainers-email-body.txt", u.SampleMaintainersEmail.Body)
	outputString = runClientCommand(t, storeAdminCommand+" email send-to-maintainers %q %q", u.SampleMaintainersEmail.Subject, maintainersEmailBodyFile)
	assert.Equal(t, "emails sent: 2\nemails failed: 0\n", outputString)

	failingMaintainerPublicKeyRaw, err := u.DecodeAuthorizedEd25519PublicKey([]byte(store.AppStoreOfficialMaintainerPublicKeyOpenSSH))
	assert.Nil(t, err)
	err = createMaintainerByAdmin(storeClient, "failingmaintainer", u.SampleEmailFailingRecipient, failingMaintainerPublicKeyRaw)
	assert.Nil(t, err)

	failureEmailBodyFile := writeTerminalEmailBodyFile(t, "failing-maintainers-email-body.txt", u.SampleMaintainersFailureEmail.Body)
	outputString = runClientCommandWithInputExpectingError(t, "", storeAdminCommand+" email send-to-maintainers %q %q", u.SampleMaintainersFailureEmail.Subject, failureEmailBodyFile)
	assert.True(t, strings.Contains(outputString, "emails sent: 2"))
	assert.True(t, strings.Contains(outputString, "emails failed: 1"))
	assert.True(t, strings.Contains(outputString, "failed recipients:"))
	assert.True(t, strings.Contains(outputString, "- "+u.SampleEmailFailingRecipient+": "+u.SampleEmailSendFailedError))
}

func TestAdminMaintainerTerminalWorkflow(t *testing.T) {
	artifacts := setupTerminalArtifacts(t)
	dependencies := setupTerminalDependencies(t)
	storeClient := dependencies.AppStoreClient
	defer storeClient.WipeData()
	assert.Nil(t, loginWithSigning(dependencies.ConfigProvider, dependencies.SigningKeyManager, storeClient, terminalAdminMaintainer, terminalAdminPassword, artifacts.adminPrivateKeyPath, u.LocalTestingPrivateKeyPassphrase))
	sampleEmailConfig := u.SampleEmailConfig
	assert.Nil(t, storeClient.SetEmailConfig(&sampleEmailConfig))

	maintainerPassword := "newpassword"
	outputString := runClientCommandWithInput(t, u.LocalTestingPrivateKeyPassphrase+"\n", storeAdminMaintainersCommand+" create %s %s %q", tools.SampleMaintainer, tools.SampleEmail, artifacts.maintainerPublicKey)
	assert.Equal(t, "Private key passphrase: app maintainer created successfully\n", outputString)

	outputString = runClientCommand(t, storeAdminMaintainersCommand+" list")
	assert.Equal(t, fmt.Sprintf(`name     email              status   public key fingerprint
quollix  admin@quollix.org  active   %s
sample   sample@sample.com  pending  %s
`, u.GetLocalTestingPublicKeyFingerprintSHA256(), u.OtherLocalTestingPublicKeyFingerprintSHA256), outputString)

	freshClient := setupTerminalDependencies(t).AppStoreClient
	assert.NotNil(t, freshClient.Login(tools.SampleMaintainer, maintainerPassword))

	outputString = runClientCommandWithInput(t, maintainerPassword+"\n"+maintainerPassword+"\n", storeOnboardingCommand+" setup-password %s", defaultRegistrationConfirmationCode)
	assert.True(t, strings.Contains(outputString, "password setup successful"))

	freshClient = setupTerminalDependencies(t).AppStoreClient
	assert.Nil(t, freshClient.Login(tools.SampleMaintainer, maintainerPassword))
	accountDetails, err := freshClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.False(t, accountDetails.IsAdmin)
	assert.Equal(t, tools.SampleEmail, accountDetails.Email)

	appName := "nginx"
	assert.Nil(t, freshClient.CreateApp(appName))
	content := terminalVersionContentForMaintainerApp(t, tools.SampleMaintainer, appName, mustReadFile(t, artifacts.composePath))
	assert.Nil(t, loginWithSigning(dependencies.ConfigProvider, dependencies.SigningKeyManager, freshClient, tools.SampleMaintainer, maintainerPassword, artifacts.maintainerPrivateKeyPath, u.OtherLocalTestingPrivateKeyPassphrase))
	assert.Nil(t, uploadSignedTerminalVersionForMaintainer(freshClient, dependencies, tools.SampleMaintainer, appName, "1.0.0", content))
	apps, err := setupTerminalDependencies(t).AppStoreClient.SearchForApps(tools.SampleMaintainer, appName, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))

	assert.Nil(t, loginWithSigning(dependencies.ConfigProvider, dependencies.SigningKeyManager, storeClient, terminalAdminMaintainer, terminalAdminPassword, artifacts.adminPrivateKeyPath, u.LocalTestingPrivateKeyPassphrase))
	outputString = runClientCommandWithInput(t, "DELETE\n", storeAdminMaintainersCommand+" delete %s", tools.SampleMaintainer)
	assert.True(t, strings.Contains(outputString, "app maintainer deletion successful"))
	apps, err = setupTerminalDependencies(t).AppStoreClient.SearchForApps(tools.SampleMaintainer, appName, true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
	err = freshClient.Login(tools.SampleMaintainer, maintainerPassword)
	u.AssertDeepStackErrorFromRequest(t, err, "incorrect username or password")

	outputString = runClientCommandWithInputExpectingError(t, "DELETE\n", storeAdminMaintainersCommand+" delete %s", tools.SampleMaintainer)
	assert.True(t, strings.Contains(outputString, "maintainer not found"))

	publicClient := setupTerminalDependencies(t).AppStoreClient
	assert.NotNil(t, publicClient.Login(tools.SampleMaintainer, maintainerPassword))
}

func TestAdminMaintainerSetSpaceTerminal(t *testing.T) {
	dependencies := setupTerminalDependencies(t)
	storeClient := dependencies.AppStoreClient
	defer storeClient.WipeData()
	assert.Nil(t, loginWithoutSigning(dependencies.ConfigProvider, storeClient, terminalAdminMaintainer, terminalAdminPassword))

	outputString := runClientCommand(t, storeAdminMaintainersCommand+" set-space %s 20", terminalAdminMaintainer)
	assert.Equal(t, "app maintainer storage limit updated successfully\n", outputString)

	accountDetails, err := storeClient.GetAccountDetails()
	assert.Nil(t, err)
	assert.Equal(t, int64(20*1024*1024), accountDetails.StorageLimitInBytes)
}

func setupTerminalArtifacts(t *testing.T) terminalArtifacts {
	userConfigDir, err := os.UserConfigDir()
	assert.Nil(t, err)
	configPath := tools.InitGlobalConfig(os.Getenv("PROFILE"), userConfigDir).ConfigFilePath
	_ = os.Remove(configPath)
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	tempDir := t.TempDir()
	adminPrivateKeyPath := tempDir + "/admin-test-key"
	assert.Nil(t, os.WriteFile(adminPrivateKeyPath, []byte(u.LocalTestingPrivateKeyOpenSSH), 0o600))

	maintainerPrivateKeyPath := tempDir + "/maintainer-test-key"
	assert.Nil(t, os.WriteFile(maintainerPrivateKeyPath, []byte(u.OtherLocalTestingPrivateKeyOpenSSH), 0o600))

	publicKey, err := ssh.NewPublicKey(ed25519.PublicKey(u.GetOtherLocalTestingPublicKeyRaw()))
	assert.Nil(t, err)
	publicKeyString := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey)))
	sourceComposeContent, err := os.ReadFile(nginxComposePath)
	assert.Nil(t, err)
	composePath := tempDir + "/nginx.yml"
	assert.Nil(t, os.WriteFile(composePath, sourceComposeContent, 0o644))
	return terminalArtifacts{
		adminPrivateKeyPath:      adminPrivateKeyPath,
		maintainerPrivateKeyPath: maintainerPrivateKeyPath,
		maintainerPublicKey:      publicKeyString,
		appsDir:                  tempDir,
		composePath:              composePath,
	}
}

func setupTerminalDependencies(t *testing.T) *commands.ClientDependencies {
	userConfigDir, err := os.UserConfigDir()
	assert.Nil(t, err)
	config := tools.InitGlobalConfig(os.Getenv("PROFILE"), userConfigDir)
	return di.BuildDependencyGraph(config)
}

func writeTerminalEmailBodyFile(t *testing.T, fileName string, body string) string {
	path := filepath.Join(t.TempDir(), fileName)
	assert.Nil(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func writeTerminalComposeFile(t *testing.T, appsDir string, maintainer string, appName string, imageName string, imageTag string) string {
	path := filepath.Join(appsDir, appName+".yml")
	content := validation.AppDefinitionLicenseNotice + fmt.Sprintf(`
services:
    %s:
        container_name: %s_%s_%s
        image: %s:%s
        labels:
            quollix.port: 8080
`, appName, maintainer, appName, appName, imageName, imageTag)
	assert.Nil(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func assertTerminalUploadedVersion(t *testing.T, client *store.AppStoreClientImpl, appName string, versionName string, expectedContent []byte) {
	versions, err := client.ListVersions(tools.SampleMaintainer, appName)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(versions))
	assert.Equal(t, versionName, versions[0].Name)
	uploadedVersion, err := client.DownloadVersionByID(versions[0].VersionId)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleMaintainer, uploadedVersion.Maintainer)
	assert.Equal(t, appName, uploadedVersion.AppName)
	assert.Equal(t, versionName, uploadedVersion.VersionName)
	tools.EqualYaml(t, expectedContent, uploadedVersion.Content)
}

func assertTerminalFileDoesNotExist(t *testing.T, path string) {
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func containsTerminalString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func formatTerminalTimestamp(timestamp time.Time) string {
	return timestamp.Format("2006-01-02 15:04:05")
}

func maskTerminalSecret(value string) string {
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
}

func setupAuthenticatedTerminalMaintainer(t *testing.T) terminalSetup {
	artifacts := setupTerminalArtifacts(t)
	dependencies := setupTerminalDependencies(t)
	storeClient := dependencies.AppStoreClient

	assert.Nil(t, loginWithoutSigning(dependencies.ConfigProvider, storeClient, terminalAdminMaintainer, terminalAdminPassword))
	sampleEmailConfig := u.SampleEmailConfig
	assert.Nil(t, storeClient.SetEmailConfig(&sampleEmailConfig))
	err := createMaintainerByAdmin(storeClient, tools.SampleMaintainer, tools.SampleEmail, u.GetOtherLocalTestingPublicKeyRaw())
	assert.Nil(t, err)
	assert.Nil(t, storeClient.SetupInitialPassword(defaultRegistrationConfirmationCode, tools.SamplePassword))
	assert.Nil(t, loginWithSigning(dependencies.ConfigProvider, dependencies.SigningKeyManager, storeClient, tools.SampleMaintainer, tools.SamplePassword, artifacts.maintainerPrivateKeyPath, u.OtherLocalTestingPrivateKeyPassphrase))
	setTerminalAppsDirectory(t, dependencies, artifacts.appsDir)

	return terminalSetup{
		artifacts:    artifacts,
		dependencies: dependencies,
		storeClient:  storeClient,
	}
}

func setTerminalAppsDirectory(t *testing.T, dependencies *commands.ClientDependencies, appsDirectory string) {
	config, err := dependencies.ConfigProvider.GetConfig()
	assert.Nil(t, err)
	config.AppsDirectory = appsDirectory
	assert.Nil(t, dependencies.ConfigProvider.SetConfig(config))
}

func loginWithSigning(configProvider configuration.Provider, signingKeyManager remote.SigningKeyManager, storeClient *store.AppStoreClientImpl, username string, password string, privateKeyPath string, privateKeyPassphrase string) error {
	if err := storeClient.Login(username, password); err != nil {
		return err
	}
	accountDetails, err := storeClient.GetAccountDetails()
	if err != nil {
		return err
	}
	if storeClient.Parent.Cookie == nil {
		return u.Logger.NewError("sign-in did not provide auth cookie")
	}
	if err = configProvider.SetConfig(&configuration.Config{
		Session: &configuration.SessionData{
			Maintainer:     username,
			Cookie:         storeClient.Parent.Cookie.Value,
			ExpirationDate: storeClient.Parent.Cookie.Expires,
		},
	}); err != nil {
		return err
	}
	if err = signingKeyManager.StorePublicKeyRaw(accountDetails.PublicKeyRaw); err != nil {
		return err
	}
	return signingKeyManager.SetPrivateKeyPath(privateKeyPath, privateKeyPassphrase)
}

func createMaintainerByAdmin(client *store.AppStoreClientImpl, name string, email string, publicKeyRaw []byte) error {
	signature, err := remote.SignMaintainerPublicKey([]byte(u.LocalTestingPrivateKeyOpenSSH), u.LocalTestingPrivateKeyPassphrase, name, publicKeyRaw)
	if err != nil {
		return err
	}
	return client.CreateMaintainerByAdmin(name, email, publicKeyRaw, signature)
}

func loginWithoutSigning(configProvider configuration.Provider, storeClient *store.AppStoreClientImpl, username string, password string) error {
	if err := storeClient.Login(username, password); err != nil {
		return err
	}
	if storeClient.Parent.Cookie == nil {
		return u.Logger.NewError("sign-in did not provide auth cookie")
	}
	return configProvider.SetConfig(&configuration.Config{
		Session: &configuration.SessionData{
			Maintainer:     username,
			Cookie:         storeClient.Parent.Cookie.Value,
			ExpirationDate: storeClient.Parent.Cookie.Expires,
		},
	})
}

func uploadSignedTerminalVersion(dependencies *commands.ClientDependencies, appName string, versionName string, content []byte) error {
	creationTimestamp := time.Now().UTC().Add(-5 * time.Minute)
	_, err := uploadSignedTerminalVersionAt(dependencies, appName, versionName, creationTimestamp, content)
	return err
}

func uploadSignedTerminalVersionAt(dependencies *commands.ClientDependencies, appName string, versionName string, creationTimestamp time.Time, content []byte) (*store.CreatedVersionResponse, error) {
	return uploadSignedTerminalVersionAtForMaintainer(dependencies.AppStoreClient, dependencies, tools.SampleMaintainer, appName, versionName, creationTimestamp, content)
}

func uploadSignedTerminalVersionForMaintainer(client *store.AppStoreClientImpl, dependencies *commands.ClientDependencies, maintainer string, appName string, versionName string, content []byte) error {
	creationTimestamp := time.Now().UTC().Add(-5 * time.Minute)
	_, err := uploadSignedTerminalVersionAtForMaintainer(client, dependencies, maintainer, appName, versionName, creationTimestamp, content)
	return err
}

func uploadSignedTerminalVersionAtForMaintainer(client *store.AppStoreClientImpl, dependencies *commands.ClientDependencies, maintainer string, appName string, versionName string, creationTimestamp time.Time, content []byte) (*store.CreatedVersionResponse, error) {
	signature, err := remote.SignVersionPayload(dependencies.VersionSigningService, []byte(u.OtherLocalTestingPrivateKeyOpenSSH), u.OtherLocalTestingPrivateKeyPassphrase, maintainer, appName, versionName, creationTimestamp, content)
	if err != nil {
		return nil, err
	}
	return client.UploadVersionAndReturnCreatedVersion(appName, versionName, creationTimestamp, content, signature)
}

func terminalVersionContentForMaintainerApp(t *testing.T, maintainer string, appName string, content []byte) []byte {
	expectedContainerNamePrefix := tools.SampleMaintainer + "_" + appName + "_"
	assert.True(t, strings.Contains(string(content), "container_name: "+expectedContainerNamePrefix))
	return []byte(strings.ReplaceAll(string(content), expectedContainerNamePrefix, maintainer+"_"+appName+"_"))
}

func runClientCommand(t *testing.T, format string, args ...any) string {
	command := fmt.Sprintf(format, args...)
	cmd := exec.Command("bash", "-lc", "go run . "+command)
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	assert.Nil(t, err)
	return string(output)
}

func runClientCommandWithInput(t *testing.T, input string, format string, args ...any) string {
	command := fmt.Sprintf(format, args...)
	cmd := exec.Command("bash", "-lc", "go run . "+command)
	cmd.Dir = ".."
	cmd.Stdin = bytes.NewBufferString(input)
	output, err := cmd.CombinedOutput()
	assert.Nil(t, err)
	return string(output)
}

func runClientCommandWithInputExpectingError(t *testing.T, input string, format string, args ...any) string {
	command := fmt.Sprintf(format, args...)
	cmd := exec.Command("bash", "-lc", "go run . "+command)
	cmd.Dir = ".."
	cmd.Stdin = bytes.NewBufferString(input)
	output, err := cmd.CombinedOutput()
	assert.NotNil(t, err)
	return string(output)
}

func mustReadFile(t *testing.T, path string) []byte {
	content, err := os.ReadFile(path)
	assert.Nil(t, err)
	return content
}

func nginxComposeContentWithTag(t *testing.T, tag string) []byte {
	content := string(mustReadFile(t, nginxComposePath))
	return []byte(strings.Replace(content, "nginx:1.0.0", "nginx:"+tag+"-alpine", 1))
}
