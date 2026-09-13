package setup

import (
	"crypto/ed25519"
	"crypto/sha256"
	"time"

	"server/maintainers"
	"server/versions"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

const (
	sampleApp     = "nginx"
	sampleVersion = "1.0"
)

var sampleVersionContents = []string{
	`services:
    nginx:
        container_name: quollix_nginx_nginx
        image: nginx:1.0.0
        labels:
            quollix.port: 8080
`,
	`services:
    nginx:
        container_name: quollix_nginx_nginx
        image: nginx:1.0.1
        labels:
            quollix.port: 8080
`,
	`services:
    nginx:
        container_name: quollix_nginx_nginx
        image: nginx:1.0.2
        labels:
            quollix.port: 8080
`,
}

func SeedSampleData(deps *InitializerDependencies) error {
	sampleEmailConfig := u.SampleEmailConfig
	if err := deps.EmailConfigRepo.SetEmailConfig(&sampleEmailConfig); err != nil {
		return err
	}

	user, err := deps.UserRepo.GetUserByName(maintainers.QuollixAdminUsername)
	if err != nil {
		return err
	}
	appExists, err := deps.AppRepo.DoesAppExistByMaintainerID(user.Id, sampleApp)
	if err != nil {
		return err
	}
	if !appExists {
		if err := deps.AppRepo.CreateApp(user.Id, sampleApp); err != nil {
			return err
		}
	}
	app, err := deps.AppRepo.GetAppByName(user.Id, sampleApp)
	if err != nil {
		return err
	}

	versionsExist, err := deps.VersionRepo.DoesVersionNameExistByAppID(app.AppId, sampleVersion)
	if err != nil {
		return err
	}
	if versionsExist {
		return nil
	}

	privateKey, err := u.DecodeEd25519PrivateKeyOpenSSH([]byte(u.LocalTestingPrivateKeyOpenSSH), []byte(u.LocalTestingPrivateKeyPassphrase))
	if err != nil {
		return err
	}
	baseTimestamp := time.Now().UTC().Add(-3 * time.Minute).Round(time.Microsecond)
	for i, content := range sampleVersionContents {
		creationTimestamp := baseTimestamp.Add(time.Duration(i) * time.Minute)
		signature, err := signSampleVersion(privateKey, []byte(content), creationTimestamp)
		if err != nil {
			return err
		}
		contentHash := versions.VersionContentHash(sha256.Sum256([]byte(content)))
		if _, err := deps.VersionRepo.CreateVersion(app.AppId, sampleVersion, creationTimestamp, []byte(content), contentHash, signature); err != nil {
			return err
		}
	}
	return nil
}

func signSampleVersion(privateKey ed25519.PrivateKey, content []byte, creationTimestamp time.Time) ([]byte, error) {
	payloadBytes, err := (&store.VersionSigningCodecImpl{}).EncodeVersion(&store.Version{
		Maintainer:               maintainers.QuollixAdminUsername,
		AppName:                  sampleApp,
		VersionName:              sampleVersion,
		Content:                  content,
		VersionCreationTimestamp: creationTimestamp,
	})
	if err != nil {
		return nil, err
	}
	return (&u.BytesSignerImpl{}).SignBytes(privateKey, payloadBytes), nil
}
