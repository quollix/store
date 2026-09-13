package local

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	u "github.com/quollix/common/utils"
	"gopkg.in/yaml.v3"
)

const imageTagCacheTTL = 12 * time.Hour

type imageTagCacheFile struct {
	Images  map[string]imageTagCacheFileEntry  `yaml:"images"`
	Digests map[string]digestTagCacheFileEntry `yaml:"digests,omitempty"`
}

type imageTagCacheFileEntry struct {
	Tags       []string `yaml:"tags"`
	LookedUpAt string   `yaml:"looked_up_at"`
}

type digestTagCacheFileEntry struct {
	Digest     string `yaml:"digest"`
	LookedUpAt string `yaml:"looked_up_at"`
}

type imageTagCacheEntry struct {
	Tags       []string
	LookedUpAt time.Time
}

type digestTagCacheEntry struct {
	Digest     string
	LookedUpAt time.Time
}

type ImageTagCache interface {
	Load()
	Save()
	GetTags(image string) ([]string, bool)
	PutTags(image string, tags []string)
	GetDigest(image, tag string) (string, bool)
	PutDigest(image, tag, digest string)
}

type NoOpImageTagCache struct{}

func (c *NoOpImageTagCache) Load() {}

func (c *NoOpImageTagCache) Save() {}

func (c *NoOpImageTagCache) GetTags(image string) ([]string, bool) {
	return nil, false
}

func (c *NoOpImageTagCache) PutTags(image string, tags []string) {}

func (c *NoOpImageTagCache) GetDigest(image, tag string) (string, bool) {
	return "", false
}

func (c *NoOpImageTagCache) PutDigest(image, tag, digest string) {}

type ImageTagCacheImpl struct {
	OsWrapper u.OsWrapper
	FilePath  string

	mu      sync.RWMutex
	images  map[string]imageTagCacheEntry
	digests map[string]digestTagCacheEntry
}

func (c *ImageTagCacheImpl) Load() {
	data, err := c.OsWrapper.ReadFile(c.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.reset()
			return
		}
		u.Logger.Warn("failed to load image tag cache", "path", c.FilePath, "details", u.ExtractError(err))
		c.reset()
		return
	}

	cacheFile, err := c.decode(data)
	if err != nil {
		u.Logger.Warn("failed to decode image tag cache", "path", c.FilePath, "details", u.ExtractError(err))
		c.reset()
		return
	}

	images := make(map[string]imageTagCacheEntry, len(cacheFile.Images))
	digests := make(map[string]digestTagCacheEntry, len(cacheFile.Digests))
	now := c.nowUTC()
	for image, entry := range cacheFile.Images {
		lookedUpAt, err := time.Parse(time.RFC3339, entry.LookedUpAt)
		if err != nil {
			u.Logger.Warn("failed to parse image tag cache timestamp", "path", c.FilePath, "image", image, "timestamp", entry.LookedUpAt)
			continue
		}
		lookedUpAt = lookedUpAt.UTC()
		if now.Sub(lookedUpAt) > imageTagCacheTTL {
			continue
		}
		images[image] = imageTagCacheEntry{
			Tags:       slices.Clone(entry.Tags),
			LookedUpAt: lookedUpAt,
		}
	}
	for key, entry := range cacheFile.Digests {
		lookedUpAt, err := time.Parse(time.RFC3339, entry.LookedUpAt)
		if err != nil {
			u.Logger.Warn("failed to parse digest cache timestamp", "path", c.FilePath, "image", key, "timestamp", entry.LookedUpAt)
			continue
		}
		lookedUpAt = lookedUpAt.UTC()
		if now.Sub(lookedUpAt) > imageTagCacheTTL {
			continue
		}
		digests[key] = digestTagCacheEntry{
			Digest:     entry.Digest,
			LookedUpAt: lookedUpAt,
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.images = images
	c.digests = digests
}

func (c *ImageTagCacheImpl) Save() {
	cacheFile := c.snapshot()
	data, err := yaml.Marshal(cacheFile)
	if err != nil {
		u.Logger.Warn("failed to encode image tag cache", "path", c.FilePath, "details", u.ExtractError(err))
		return
	}
	if err := c.OsWrapper.MkdirAll(filepath.Dir(c.FilePath), 0o755); err != nil {
		u.Logger.Warn("failed to prepare image tag cache directory", "path", c.FilePath, "details", u.ExtractError(err))
		return
	}
	if err := c.OsWrapper.WriteFile(c.FilePath, data, 0o644); err != nil {
		u.Logger.Warn("failed to save image tag cache", "path", c.FilePath, "details", u.ExtractError(err))
	}
}

func (c *ImageTagCacheImpl) GetTags(image string) ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.images[image]
	if !ok {
		return nil, false
	}
	return slices.Clone(entry.Tags), true
}

func (c *ImageTagCacheImpl) PutTags(image string, tags []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.images == nil {
		c.images = make(map[string]imageTagCacheEntry)
	}
	c.images[image] = imageTagCacheEntry{
		Tags:       slices.Clone(tags),
		LookedUpAt: c.nowUTC(),
	}
}

func (c *ImageTagCacheImpl) GetDigest(image, tag string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.digests[digestCacheKey(image, tag)]
	if !ok {
		return "", false
	}
	return entry.Digest, true
}

func (c *ImageTagCacheImpl) PutDigest(image, tag, digest string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.digests == nil {
		c.digests = make(map[string]digestTagCacheEntry)
	}
	c.digests[digestCacheKey(image, tag)] = digestTagCacheEntry{
		Digest:     digest,
		LookedUpAt: c.nowUTC(),
	}
}

func (c *ImageTagCacheImpl) decode(data []byte) (imageTagCacheFile, error) {
	var cacheFile imageTagCacheFile
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cacheFile); err != nil {
		if err == io.EOF {
			return imageTagCacheFile{Images: map[string]imageTagCacheFileEntry{}, Digests: map[string]digestTagCacheFileEntry{}}, nil
		}
		return imageTagCacheFile{}, u.Logger.NewError(err.Error())
	}
	if cacheFile.Images == nil {
		cacheFile.Images = map[string]imageTagCacheFileEntry{}
	}
	if cacheFile.Digests == nil {
		cacheFile.Digests = map[string]digestTagCacheFileEntry{}
	}
	return cacheFile, nil
}

func (c *ImageTagCacheImpl) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.images = make(map[string]imageTagCacheEntry)
	c.digests = make(map[string]digestTagCacheEntry)
}

func (c *ImageTagCacheImpl) snapshot() imageTagCacheFile {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := imageTagCacheFile{
		Images:  make(map[string]imageTagCacheFileEntry, len(c.images)),
		Digests: make(map[string]digestTagCacheFileEntry, len(c.digests)),
	}
	for image, entry := range c.images {
		out.Images[image] = imageTagCacheFileEntry{
			Tags:       slices.Clone(entry.Tags),
			LookedUpAt: entry.LookedUpAt.UTC().Format(time.RFC3339),
		}
	}
	for key, entry := range c.digests {
		out.Digests[key] = digestTagCacheFileEntry{
			Digest:     entry.Digest,
			LookedUpAt: entry.LookedUpAt.UTC().Format(time.RFC3339),
		}
	}
	return out
}

func (c *ImageTagCacheImpl) nowUTC() time.Time {
	return c.OsWrapper.Now().UTC()
}

func digestCacheKey(image, tag string) string {
	return image + ":" + tag
}
