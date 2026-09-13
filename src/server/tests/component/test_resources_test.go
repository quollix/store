//go:build component || production

package check

import (
	"os"
	"testing"
	"time"

	"server/maintainers"
	"server/tools"

	"github.com/quollix/common/store"
	"github.com/quollix/common/validation"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

var (
	sampleMaintainer         = tools.SampleMaintainer
	sampleApp                = tools.SampleApp
	sampleVersion            = tools.SampleVersion
	sampleEmail              = tools.SampleEmail
	samplePassword           = tools.SamplePassword
	doesNotExistError        = tools.DoesNotExistError
	cookieName               = "auth"
	SampleVersionFileContent = GetValidVersionBytesOfSampleMaintainerApp()
	versionSigningService    = &store.VersionSigningServiceImpl{
		Codec:       &store.VersionSigningCodecImpl{},
		BytesSigner: &u.BytesSignerImpl{},
	}
)

func GetValidVersionBytesOfSampleMaintainerApp() []byte {
	assetsDir, err := u.FindDir("assets")
	if err != nil {
		panic("Failed to find assets directory")
	}
	sampleAppDir := assetsDir + "/" + sampleApp + "/docker-compose.yml"
	versionBytes, err := os.ReadFile(sampleAppDir) // #nosec G304 (CWE-22): Potential file inclusion via variable
	if err != nil {
		panic("Failed to read sample version file")
	}
	return append([]byte(validation.AppDefinitionLicenseNotice+"\n"), versionBytes...)
}

func GetStoreClientAndLogin(t *testing.T) *store.AppStoreClientImpl {
	storeClient := GetStoreClient()
	err := CreateMaintainerByAdminAndSetPassword(t, sampleMaintainer, samplePassword, sampleEmail, u.GetOtherLocalTestingPublicKeyRaw())
	assert.Nil(t, err)
	err = storeClient.Login(sampleMaintainer, samplePassword)
	assert.Nil(t, err)
	return storeClient
}

func GetStoreClient() *store.AppStoreClientImpl {
	return &store.AppStoreClientImpl{
		Parent: u.ComponentClient{
			SetCookieHeader: true,
			RootUrl:         maintainers.DefaultAppStoreHost,
		},
	}
}

func GetAdminStoreClientAndLogin(t *testing.T) *store.AppStoreClientImpl {
	adminClient := GetStoreClient()
	assert.Nil(t, adminClient.Login(maintainers.QuollixAdminUsername, maintainers.QuollixAdminPassword))
	return adminClient
}

func CreateMaintainerByAdminAndSetPassword(t *testing.T, userName string, password string, email string, publicKey []byte) error {
	adminClient := GetAdminStoreClientAndLogin(t)
	err := createMaintainerByAdmin(t, adminClient, userName, email, publicKey)
	if err != nil {
		return err
	}
	return GetStoreClient().SetupInitialPassword(maintainers.DefaultAccountRegistrationCode, password)
}

func NewSignedVersionUpload(appName, versionName string, content []byte, creationTimestamp time.Time) (time.Time, []byte, error) {
	privateKey, err := u.DecodeEd25519PrivateKeyOpenSSH([]byte(u.OtherLocalTestingPrivateKeyOpenSSH), []byte(u.OtherLocalTestingPrivateKeyPassphrase))
	if err != nil {
		return time.Time{}, nil, err
	}
	signature, err := versionSigningService.SignVersion(privateKey, &store.Version{
		Maintainer:               sampleMaintainer,
		AppName:                  appName,
		VersionName:              versionName,
		Content:                  content,
		VersionCreationTimestamp: creationTimestamp.UTC(),
	})
	if err != nil {
		return time.Time{}, nil, err
	}
	return creationTimestamp.UTC(), signature, nil
}

func getOtherTestingPublicKey() []byte {
	publicKey, err := u.DecodeAuthorizedEd25519PublicKey([]byte(store.AppStoreOfficialMaintainerPublicKeyOpenSSH))
	if err != nil {
		panic(err)
	}
	return publicKey
}

func getQuollixAdminPublicKey() []byte {
	publicKey, err := u.DecodeAuthorizedEd25519PublicKey(u.LocalTestingPublicKeyOpenSSHBytes)
	if err != nil {
		panic(err)
	}
	return publicKey
}

func signMaintainerPublicKey(t *testing.T, maintainer string, publicKey []byte) []byte {
	privateKey, err := u.DecodeEd25519PrivateKeyOpenSSH([]byte(u.LocalTestingPrivateKeyOpenSSH), []byte(u.LocalTestingPrivateKeyPassphrase))
	assert.Nil(t, err)
	signature, err := store.SignMaintainerPublicKey(privateKey, maintainer, publicKey)
	assert.Nil(t, err)
	return signature
}

func createMaintainerByAdmin(t *testing.T, client *store.AppStoreClientImpl, name string, email string, publicKeyRaw []byte) error {
	setTestEmailConfig(t, client)
	return client.CreateMaintainerByAdmin(name, email, publicKeyRaw, signMaintainerPublicKey(t, name, publicKeyRaw))
}

func setTestEmailConfig(t *testing.T, client *store.AppStoreClientImpl) {
	testEmailConfig := u.SampleEmailConfig
	assert.Nil(t, client.SetEmailConfig(&testEmailConfig))
}

func UploadSignedVersion(client *store.AppStoreClientImpl, appName, versionName string, content []byte) error {
	_, err := UploadSignedVersionAndReturnCreatedVersion(client, appName, versionName, content)
	return err
}

func UploadSignedVersionAndReturnCreatedVersion(client *store.AppStoreClientImpl, appName, versionName string, content []byte) (*store.CreatedVersionResponse, error) {
	creationTimestamp, signature, err := NewSignedVersionUpload(appName, versionName, content, time.Now().Add(-5*time.Minute))
	if err != nil {
		return nil, err
	}
	return client.UploadVersionAndReturnCreatedVersion(appName, versionName, creationTimestamp, content, signature)
}
