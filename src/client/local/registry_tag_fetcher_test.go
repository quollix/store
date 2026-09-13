package local

import (
	"testing"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

type registryTagFetcherTestDependencies struct {
	registryTagFetcher   *RegistryTagFetcherImpl
	dockerHubRegistry    *DockerHubRegistryMock
	ociRegistry          *OCIRegistryMock
	tagSelector          *TagSelectorMock
	imageTagCache        *ImageTagCacheMock
	imageReferenceParser *ImageReferenceParserMock
}

func setupRegistryTagFetcherTest(t *testing.T) *registryTagFetcherTestDependencies {
	dockerHubRegistryMock := NewDockerHubRegistryMock(t)
	ociRegistryMock := NewOCIRegistryMock(t)
	tagSelectorMock := NewTagSelectorMock(t)
	imageTagCacheMock := NewImageTagCacheMock(t)
	imageReferenceParserMock := NewImageReferenceParserMock(t)
	return &registryTagFetcherTestDependencies{
		registryTagFetcher: &RegistryTagFetcherImpl{
			DockerHubRegistry:    dockerHubRegistryMock,
			OCIRegistry:          ociRegistryMock,
			TagSelector:          tagSelectorMock,
			ImageTagCache:        imageTagCacheMock,
			ImageReferenceParser: imageReferenceParserMock,
		},
		dockerHubRegistry:    dockerHubRegistryMock,
		ociRegistry:          ociRegistryMock,
		tagSelector:          tagSelectorMock,
		imageTagCache:        imageTagCacheMock,
		imageReferenceParser: imageReferenceParserMock,
	}
}

func TestRegistryTagFetcherImpl_GetLatestTag_UsesCachedTags(t *testing.T) {
	deps := setupRegistryTagFetcherTest(t)
	tags := []string{"1.27.4-alpine", "1.27.5-alpine"}
	deps.imageTagCache.EXPECT().GetTags("nginx").Return(tags, true)
	deps.tagSelector.EXPECT().SelectLatestTag("1.27.4-alpine", tags).Return("1.27.5-alpine", true, nil)

	latestTag, found, err := deps.registryTagFetcher.GetLatestTag("nginx", "1.27.4-alpine")

	assert.Nil(t, err)
	assert.True(t, found)
	assert.Equal(t, "1.27.5-alpine", latestTag)
}

func TestRegistryTagFetcherImpl_GetLatestTag_FetchesDockerHubTagsAndCachesThem(t *testing.T) {
	deps := setupRegistryTagFetcherTest(t)
	tags := []string{"1.27.4-alpine", "1.27.5-alpine"}
	deps.imageTagCache.EXPECT().GetTags("nginx").Return(nil, false)
	deps.imageReferenceParser.EXPECT().ParseRegistryImageReference("nginx").Return(imageReference{provider: providerDockerHub, repoPath: "library/nginx"}, nil)
	deps.dockerHubRegistry.EXPECT().FetchTags("library/nginx").Return(tags, nil)
	deps.imageTagCache.EXPECT().PutTags("nginx", tags)
	deps.imageTagCache.EXPECT().Save()
	deps.tagSelector.EXPECT().SelectLatestTag("1.27.4-alpine", tags).Return("1.27.5-alpine", true, nil)

	latestTag, found, err := deps.registryTagFetcher.GetLatestTag("nginx", "1.27.4-alpine")

	assert.Nil(t, err)
	assert.True(t, found)
	assert.Equal(t, "1.27.5-alpine", latestTag)
}

func TestRegistryTagFetcherImpl_GetDigest_UsesCachedDigest(t *testing.T) {
	deps := setupRegistryTagFetcherTest(t)
	deps.imageTagCache.EXPECT().GetDigest("ghcr.io/example/app", "1.2.3").Return(testDigestA, true)

	digest, err := deps.registryTagFetcher.GetDigest("ghcr.io/example/app", "1.2.3")

	assert.Nil(t, err)
	assert.Equal(t, testDigestA, digest)
}

func TestRegistryTagFetcherImpl_GetDigest_FetchesOCIDigestAndCachesIt(t *testing.T) {
	deps := setupRegistryTagFetcherTest(t)
	deps.imageTagCache.EXPECT().GetDigest("ghcr.io/example/app", "1.2.3").Return("", false)
	deps.imageReferenceParser.EXPECT().ParseRegistryImageReference("ghcr.io/example/app").Return(imageReference{provider: providerOCIRegistry, host: "ghcr.io", repoPath: "example/app"}, nil)
	deps.ociRegistry.EXPECT().FetchDigest("ghcr.io", "example/app", "1.2.3").Return(testDigestA, nil)
	deps.imageTagCache.EXPECT().PutDigest("ghcr.io/example/app", "1.2.3", testDigestA)
	deps.imageTagCache.EXPECT().Save()

	digest, err := deps.registryTagFetcher.GetDigest("ghcr.io/example/app", "1.2.3")

	assert.Nil(t, err)
	assert.Equal(t, testDigestA, digest)
}

func TestRegistryTagFetcherImpl_GetDigest_ReturnsErrorWhenImageReferenceIsInvalid(t *testing.T) {
	deps := setupRegistryTagFetcherTest(t)
	deps.imageTagCache.EXPECT().GetDigest("", "1.2.3").Return("", false)
	deps.imageReferenceParser.EXPECT().ParseRegistryImageReference("").Return(imageReference{}, u.Logger.NewError("image must not be empty"))

	digest, err := deps.registryTagFetcher.GetDigest("", "1.2.3")

	assert.Equal(t, "", digest)
	assert.Equal(t, "image must not be empty", u.ExtractError(err))
}
