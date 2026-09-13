package remote

import (
	"crypto/ed25519"
	"time"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

func DecodeAuthorizedEd25519PublicKey(authorizedKey []byte) ([]byte, error) {
	return u.DecodeAuthorizedEd25519PublicKey(authorizedKey)
}

func GetPublicKeyRawFromPrivateKey(privateKeyOpenSSH []byte, passphrase string) ([]byte, error) {
	privateKey, err := decodeEd25519PrivateKey(privateKeyOpenSSH, passphrase)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), privateKey.Public().(ed25519.PublicKey)...), nil
}

func SignVersionPayload(versionSigningService store.VersionSigningService, privateKeyOpenSSH []byte, passphrase string, maintainer, appName, versionName string, creationTimestamp time.Time, content []byte) ([]byte, error) {
	privateKey, err := decodeEd25519PrivateKey(privateKeyOpenSSH, passphrase)
	if err != nil {
		return nil, err
	}
	return versionSigningService.SignVersion(privateKey, &store.Version{
		Maintainer:               maintainer,
		AppName:                  appName,
		VersionName:              versionName,
		Content:                  content,
		VersionCreationTimestamp: creationTimestamp.UTC(),
	})
}

func SignMaintainerPublicKey(privateKeyOpenSSH []byte, passphrase string, maintainer string, publicKeyRaw []byte) ([]byte, error) {
	privateKey, err := decodeEd25519PrivateKey(privateKeyOpenSSH, passphrase)
	if err != nil {
		return nil, err
	}
	return store.SignMaintainerPublicKey(privateKey, maintainer, publicKeyRaw)
}

func decodeEd25519PrivateKey(privateKeyOpenSSH []byte, passphrase string) (ed25519.PrivateKey, error) {
	return u.DecodeEd25519PrivateKeyOpenSSH(privateKeyOpenSSH, []byte(passphrase))
}
