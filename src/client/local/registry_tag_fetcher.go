package local

import (
	"strings"

	"qsc/tools"

	u "github.com/quollix/common/utils"
)

type RegistryTagFetcher interface {
	GetLatestTag(image, originalTag string) (string, bool, error)
	GetDigest(image, tag string) (string, error)
}

type registryProvider int

const (
	providerDockerHub registryProvider = iota
	providerOCIRegistry
)

type imageReference struct {
	provider registryProvider
	host     string
	repoPath string
}

type RegistryTagFetcherImpl struct {
	DockerHubRegistry    DockerHubRegistry
	OCIRegistry          OCIRegistry
	TagSelector          TagSelector
	ImageTagCache        ImageTagCache
	ImageReferenceParser ImageReferenceParser
}

func (r *RegistryTagFetcherImpl) GetLatestTag(image, originalTag string) (string, bool, error) {
	if tags, ok := r.ImageTagCache.GetTags(image); ok {
		u.Logger.Info("using cached tags for Docker image", tools.ImageField, image)
		return r.TagSelector.SelectLatestTag(originalTag, tags)
	}

	tagList, err := r.fetchImageTags(image)
	if err != nil {
		return "", false, err
	}
	r.ImageTagCache.PutTags(image, tagList)
	r.ImageTagCache.Save()
	return r.TagSelector.SelectLatestTag(originalTag, tagList)
}

func (r *RegistryTagFetcherImpl) GetDigest(image, tag string) (string, error) {
	if digest, ok := r.ImageTagCache.GetDigest(image, tag); ok {
		u.Logger.Info("using cached digest for Docker image", tools.ImageField, image, tools.TagField, tag)
		return digest, nil
	}

	ref, err := r.ImageReferenceParser.ParseRegistryImageReference(image)
	if err != nil {
		return "", u.Logger.AddContext(err, tools.ImageField, image)
	}

	var digest string
	switch ref.provider {
	case providerDockerHub:
		digest, err = r.DockerHubRegistry.FetchDigest(ref.repoPath, tag)
	case providerOCIRegistry:
		digest, err = r.OCIRegistry.FetchDigest(ref.host, ref.repoPath, tag)
	default:
		return "", u.Logger.NewError("unsupported registry provider", tools.ImageField, image)
	}
	if err != nil {
		return "", err
	}
	r.ImageTagCache.PutDigest(image, tag, digest)
	r.ImageTagCache.Save()
	return digest, nil
}

func (r *RegistryTagFetcherImpl) fetchImageTags(image string) ([]string, error) {
	u.Logger.Info("fetching tags for Docker image", tools.ImageField, image)
	ref, err := r.ImageReferenceParser.ParseRegistryImageReference(image)
	if err != nil {
		return nil, u.Logger.AddContext(err, tools.ImageField, image)
	}

	switch ref.provider {
	case providerDockerHub:
		return r.DockerHubRegistry.FetchTags(ref.repoPath)
	case providerOCIRegistry:
		return r.OCIRegistry.FetchTags(ref.host, ref.repoPath)
	default:
		return nil, u.Logger.NewError("unsupported registry provider", tools.ImageField, image)
	}
}

func isRegistryHost(firstPart string) bool {
	return strings.Contains(firstPart, ".") || strings.Contains(firstPart, ":") || firstPart == "localhost"
}
