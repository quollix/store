package local

import (
	"fmt"
	"os"
	"testing"
	"time"

	"qsc/remote"

	"github.com/quollix/common/assert"
)

const sampleImageTagCachePath = "/tmp/image-tag-cache.yml"

var sampleImageTagCacheNow = time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

var imageTagCacheYAML = []byte(fmt.Sprintf(`
images:
  nginx:
    tags: ["1.27.4-alpine", "1.27.5-alpine"]
    looked_up_at: "2026-04-19T06:30:00Z"
  postgres:
    tags: ["17.5-alpine"]
    looked_up_at: "2026-04-18T22:00:00Z"
digests:
  nginx:1.27.5-alpine:
    digest: %s
    looked_up_at: "2026-04-19T06:30:00Z"
  postgres:17.5-alpine:
    digest: %s
    looked_up_at: "2026-04-18T22:00:00Z"
`, testDigestA, testDigestB))

var expectedImageTagCacheYAML = []byte(fmt.Sprintf(`images:
    nginx:
        tags:
            - 1.27.4-alpine
            - 1.27.5-alpine
        looked_up_at: "2026-04-19T12:00:00Z"
digests:
    nginx:1.27.5-alpine:
        digest: %s
        looked_up_at: "2026-04-19T12:00:00Z"
`, testDigestA))

func setupImageTagCacheTest(t *testing.T) (*ImageTagCacheImpl, *remote.OsWrapperMock) {
	osWrapperMock := remote.NewOsWrapperMock(t)
	return &ImageTagCacheImpl{
		OsWrapper: osWrapperMock,
		FilePath:  sampleImageTagCachePath,
	}, osWrapperMock
}

func TestImageTagCacheImpl_Load_MissingFileResetsToEmptyCache(t *testing.T) {
	cache, osWrapperMock := setupImageTagCacheTest(t)
	osWrapperMock.EXPECT().ReadFile(sampleImageTagCachePath).Return(nil, &os.PathError{Op: "open", Path: sampleImageTagCachePath, Err: os.ErrNotExist})

	cache.Load()

	tags, ok := cache.GetTags("nginx")
	assert.False(t, ok)
	assert.Equal(t, []string(nil), tags)
}

func TestImageTagCacheImpl_Load_PrunesExpiredEntries(t *testing.T) {
	cache, osWrapperMock := setupImageTagCacheTest(t)
	osWrapperMock.EXPECT().ReadFile(sampleImageTagCachePath).Return(imageTagCacheYAML, nil)
	osWrapperMock.EXPECT().Now().Return(sampleImageTagCacheNow)

	cache.Load()

	tags, ok := cache.GetTags("nginx")
	assert.True(t, ok)
	assert.Equal(t, []string{"1.27.4-alpine", "1.27.5-alpine"}, tags)
	expiredTags, expiredOK := cache.GetTags("postgres")
	assert.False(t, expiredOK)
	assert.Equal(t, []string(nil), expiredTags)
	digest, digestOK := cache.GetDigest("nginx", "1.27.5-alpine")
	assert.True(t, digestOK)
	assert.Equal(t, testDigestA, digest)
	expiredDigest, expiredDigestOK := cache.GetDigest("postgres", "17.5-alpine")
	assert.False(t, expiredDigestOK)
	assert.Equal(t, "", expiredDigest)
}

func TestImageTagCacheImpl_Save_WritesYamlThroughOsWrapper(t *testing.T) {
	cache, osWrapperMock := setupImageTagCacheTest(t)
	osWrapperMock.EXPECT().Now().Return(sampleImageTagCacheNow)
	cache.PutTags("nginx", []string{"1.27.4-alpine", "1.27.5-alpine"})
	cache.PutDigest("nginx", "1.27.5-alpine", testDigestA)
	osWrapperMock.EXPECT().MkdirAll("/tmp", os.FileMode(0o755)).Return(nil)
	osWrapperMock.EXPECT().WriteFile(sampleImageTagCachePath, expectedImageTagCacheYAML, os.FileMode(0o644)).Return(nil)

	cache.Save()
}
