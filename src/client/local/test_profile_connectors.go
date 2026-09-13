package local

import "qsc/tools"

import u "github.com/quollix/common/utils"

const (
	OldNginxTag    = "1.27.4-alpine"
	NewNginxTag    = "1.29.8-alpine"
	NewNginxDigest = "sha256:5616878291a2eed594aee8db4dade5878cf7edcb475e59193904b198d9b830de"
	OldCaddyTag    = "2.10.0-alpine"
	NewCaddyTag    = "2.10.2-alpine"
	OldCaddyDigest = "sha256:ae4458638da8e1a91aafffb231c5f8778e964bca650c8a8cb23a7e8ac557aa3c"
	NewCaddyDigest = "sha256:4c6e91c6ed0e2fa03efd5b44747b625fec79bc9cd06ac5235a779726618e530d"
)

type RegistryTagFetcherStubImpl struct{}

func (d *RegistryTagFetcherStubImpl) GetLatestTag(image, originalTag string) (string, bool, error) {
	switch image {
	case "nginx":
		if originalTag == NewNginxTag {
			return "", false, nil
		}
		return NewNginxTag, true, nil
	case "caddy":
		if originalTag == NewCaddyTag {
			return "", false, nil
		}
		return NewCaddyTag, true, nil
	default:
		return "", false, u.Logger.NewError("test registry tag fetcher only supports nginx and caddy", tools.ImageField, image)
	}
}

func (d *RegistryTagFetcherStubImpl) GetDigest(image, tag string) (string, error) {
	switch image {
	case "nginx":
		if tag != NewNginxTag {
			return "", u.Logger.NewError("test registry digest fetcher only supports the new nginx tag", tools.TagField, tag)
		}
		return NewNginxDigest, nil
	case "caddy":
		if tag != NewCaddyTag {
			return "", u.Logger.NewError("test registry digest fetcher only supports the new caddy tag", tools.TagField, tag)
		}
		return NewCaddyDigest, nil
	default:
		return "", u.Logger.NewError("test registry digest fetcher only supports nginx and caddy", tools.ImageField, image)
	}
}
