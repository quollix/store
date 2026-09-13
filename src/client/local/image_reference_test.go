package local

import (
	"testing"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

const (
	testDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	testDigestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	testDigestD = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

var imageReferenceParser = &ImageReferenceParserImpl{}

func TestImageReferenceParserImpl_ParseComposeImageReference_WithDigest(t *testing.T) {
	ref := imageReferenceParser.ParseComposeImageReference("registry.example.com:5000/sample/app:1.2.3@" + testDigestA)

	assert.Equal(t, "registry.example.com:5000/sample/app", ref.Image)
	assert.Equal(t, "1.2.3", ref.Tag)
	assert.Equal(t, testDigestA, ref.Digest)
}

func TestImageReferenceParserImpl_ParseComposeImageReference_WithoutTag(t *testing.T) {
	ref := imageReferenceParser.ParseComposeImageReference("sample/app@" + testDigestA)

	assert.Equal(t, "sample/app", ref.Image)
	assert.Equal(t, "", ref.Tag)
	assert.Equal(t, testDigestA, ref.Digest)
}

func TestImageReferenceParserImpl_ParseRegistryImageReference_DockerHubOfficialImage(t *testing.T) {
	ref, err := imageReferenceParser.ParseRegistryImageReference("nginx")

	assert.Nil(t, err)
	assert.Equal(t, providerDockerHub, ref.provider)
	assert.Equal(t, "", ref.host)
	assert.Equal(t, "library/nginx", ref.repoPath)
}

func TestImageReferenceParserImpl_ParseRegistryImageReference_DockerHubRepository(t *testing.T) {
	ref, err := imageReferenceParser.ParseRegistryImageReference("vaultwarden/server")

	assert.Nil(t, err)
	assert.Equal(t, providerDockerHub, ref.provider)
	assert.Equal(t, "", ref.host)
	assert.Equal(t, "vaultwarden/server", ref.repoPath)
}

func TestImageReferenceParserImpl_ParseRegistryImageReference_OCIRegistry(t *testing.T) {
	ref, err := imageReferenceParser.ParseRegistryImageReference("ghcr.io/jitsi/web")

	assert.Nil(t, err)
	assert.Equal(t, providerOCIRegistry, ref.provider)
	assert.Equal(t, "ghcr.io", ref.host)
	assert.Equal(t, "jitsi/web", ref.repoPath)
}

func TestImageReferenceParserImpl_ParseRegistryImageReference_ReturnsErrorWhenImageIsEmpty(t *testing.T) {
	ref, err := imageReferenceParser.ParseRegistryImageReference("")

	assert.Equal(t, imageReference{}, ref)
	assert.Equal(t, "image must not be empty", u.ExtractError(err))
}

func TestImageReferenceParserImpl_ParseRegistryImageReference_ReturnsErrorWhenRegistryPathIsMissing(t *testing.T) {
	ref, err := imageReferenceParser.ParseRegistryImageReference("ghcr.io")

	assert.Equal(t, imageReference{}, ref)
	assert.Equal(t, "registry image must include repository path", u.ExtractError(err))
}

func TestComposeImageReference_String_WithDigest(t *testing.T) {
	ref := composeImageReference{Image: "sample/app", Tag: "1.2.3", Digest: testDigestA}

	assert.Equal(t, "sample/app:1.2.3@"+testDigestA, ref.String())
}
