package local

import (
	"strings"

	u "github.com/quollix/common/utils"
)

type composeImageReference struct {
	Image  string
	Tag    string
	Digest string
}

type ImageReferenceParser interface {
	ParseComposeImageReference(imageReference string) composeImageReference
	ParseRegistryImageReference(image string) (imageReference, error)
}

type ImageReferenceParserImpl struct{}

func (p *ImageReferenceParserImpl) ParseComposeImageReference(imageReference string) composeImageReference {
	nameAndTag, digest, _ := strings.Cut(imageReference, "@")
	lastSlash := strings.LastIndex(nameAndTag, "/")
	lastColon := strings.LastIndex(nameAndTag, ":")
	if lastColon == -1 || lastColon < lastSlash {
		return composeImageReference{Image: nameAndTag, Digest: digest}
	}
	return composeImageReference{
		Image:  nameAndTag[:lastColon],
		Tag:    nameAndTag[lastColon+1:],
		Digest: digest,
	}
}

func (p *ImageReferenceParserImpl) ParseRegistryImageReference(image string) (imageReference, error) {
	if strings.TrimSpace(image) == "" {
		return imageReference{}, u.Logger.NewError("image must not be empty")
	}
	parts := strings.Split(image, "/")
	if isRegistryHost(parts[0]) {
		if len(parts) < 2 || strings.Join(parts[1:], "/") == "" {
			return imageReference{}, u.Logger.NewError("registry image must include repository path")
		}
		return imageReference{
			provider: providerOCIRegistry,
			host:     parts[0],
			repoPath: strings.Join(parts[1:], "/"),
		}, nil
	}
	repoPath := image
	if !strings.Contains(image, "/") {
		repoPath = "library/" + image
	}
	return imageReference{provider: providerDockerHub, repoPath: repoPath}, nil
}

func (r composeImageReference) String() string {
	imageReference := r.Image
	if r.Tag != "" {
		imageReference += ":" + r.Tag
	}
	if r.Digest != "" {
		imageReference += "@" + r.Digest
	}
	return imageReference
}
