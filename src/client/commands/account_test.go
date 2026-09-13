package commands

import (
	"qsc/remote"
	"testing"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestGetPublicKeyFingerprint(t *testing.T) {
	assert.Equal(t, "invalid public key", getPublicKeyFingerprint([]byte("not-a-key")))
	publicKeyBytes, err := remote.DecodeAuthorizedEd25519PublicKey(u.LocalTestingPublicKeyOpenSSHBytes)
	assert.Nil(t, err)
	assert.Equal(t, u.GetLocalTestingPublicKeyFingerprintSHA256(), getPublicKeyFingerprint(publicKeyBytes))
}
