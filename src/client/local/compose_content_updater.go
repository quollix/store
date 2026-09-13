package local

import (
	"strings"

	"qsc/tools"

	u "github.com/quollix/common/utils"
)

type ComposeContentUpdater interface {
	UpdateComposeTags(dockerComposeContent []byte, updates []tools.ServiceUpdate) ([]byte, error)
}

type ComposeContentUpdaterImpl struct{}

func (c *ComposeContentUpdaterImpl) UpdateComposeTags(dockerComposeContent []byte, updates []tools.ServiceUpdate) ([]byte, error) {
	updated := string(dockerComposeContent)
	for _, update := range updates {
		oldImage := composeImageReference{
			Image:  update.ImageName,
			Tag:    update.OldTag,
			Digest: update.OldDigest,
		}.String()
		newImage := composeImageReference{
			Image:  update.ImageName,
			Tag:    update.NewTag,
			Digest: update.NewDigest,
		}.String()
		nextUpdated, count := c.replaceComposeImageLine(updated, oldImage, newImage)
		if count != 1 {
			return nil, u.Logger.NewError(
				"compose image reference must exist exactly once",
				tools.ServiceNameField, update.ServiceName,
				"image", oldImage,
				"occurrences", count,
			)
		}
		updated = nextUpdated
	}
	return []byte(updated), nil
}

func (c *ComposeContentUpdaterImpl) replaceComposeImageLine(dockerComposeContent, oldImage, newImage string) (string, int) {
	lines := strings.SplitAfter(dockerComposeContent, "\n")
	occurrences := 0
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "image:") || !strings.Contains(line, oldImage) {
			continue
		}
		occurrences++
		lines[i] = strings.Replace(line, oldImage, newImage, 1)
	}
	return strings.Join(lines, ""), occurrences
}
