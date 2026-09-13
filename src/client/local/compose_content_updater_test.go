package local

import (
	"testing"

	"qsc/tools"

	"github.com/quollix/common/assert"
)

var composeContentUpdater = &ComposeContentUpdaterImpl{}

func TestComposeContentUpdaterImpl_UpdateComposeTags_UpdatesOnlyMatchingImage(t *testing.T) {
	composeContent := []byte(`
services:
  web:
    image: nginx:1.24
  database:
    image: postgres:16
`)
	expectedContent := `
services:
  web:
    image: nginx:1.25
  database:
    image: postgres:16
`
	actualContent, err := composeContentUpdater.UpdateComposeTags(composeContent, []tools.ServiceUpdate{{
		ServiceName: "web",
		ImageName:   "nginx",
		OldTag:      "1.24",
		NewTag:      "1.25",
	}})

	assert.Nil(t, err)
	assert.Equal(t, expectedContent, string(actualContent))
}

func TestComposeContentUpdaterImpl_UpdateComposeTags_PreservesFormatting(t *testing.T) {
	composeContent := []byte(`
services:
    web:
        image: nginx:1.24

    worker:
        image: redis:7
`)
	expectedContent := `
services:
    web:
        image: nginx:1.25

    worker:
        image: redis:7
`
	actualContent, err := composeContentUpdater.UpdateComposeTags(composeContent, []tools.ServiceUpdate{{ServiceName: "web", ImageName: "nginx", OldTag: "1.24", NewTag: "1.25"}})

	assert.Nil(t, err)
	assert.Equal(t, expectedContent, string(actualContent))
}

func TestComposeContentUpdaterImpl_UpdateComposeTags_IgnoresNonImageLines(t *testing.T) {
	composeContent := []byte(`
services:
  web:
    environment:
      SAMPLE_IMAGE: nginx:1.24
    image: nginx:1.24
`)
	expectedContent := `
services:
  web:
    environment:
      SAMPLE_IMAGE: nginx:1.24
    image: nginx:1.25
`
	actualContent, err := composeContentUpdater.UpdateComposeTags(composeContent, []tools.ServiceUpdate{{ServiceName: "web", ImageName: "nginx", OldTag: "1.24", NewTag: "1.25"}})

	assert.Nil(t, err)
	assert.Equal(t, expectedContent, string(actualContent))
}

func TestComposeContentUpdaterImpl_UpdateComposeTags_ReplacesDigest(t *testing.T) {
	composeContent := []byte(`
services:
  web:
    image: nginx:1.24@` + testDigestA + `
`)
	expectedContent := `
services:
  web:
    image: nginx:1.25@` + testDigestB + `
`
	actualContent, err := composeContentUpdater.UpdateComposeTags(composeContent, []tools.ServiceUpdate{{
		ServiceName: "web",
		ImageName:   "nginx",
		OldTag:      "1.24",
		NewTag:      "1.25",
		OldDigest:   testDigestA,
		NewDigest:   testDigestB,
	}})

	assert.Nil(t, err)
	assert.Equal(t, expectedContent, string(actualContent))
}

func TestComposeContentUpdaterImpl_UpdateComposeTags_ReturnsErrorWhenImageReferenceIsMissing(t *testing.T) {
	actualContent, err := composeContentUpdater.UpdateComposeTags([]byte("services: {}\n"), []tools.ServiceUpdate{{ServiceName: "web", ImageName: "nginx", OldTag: "1.24", NewTag: "1.25"}})

	assert.Nil(t, actualContent)
	assert.NotNil(t, err)
}

func TestComposeContentUpdaterImpl_UpdateComposeTags_ReturnsErrorWhenImageReferenceExistsMultipleTimes(t *testing.T) {
	composeContent := []byte(`
services:
  web:
    image: postgres:15
  database:
    image: postgres:15
`)
	actualContent, err := composeContentUpdater.UpdateComposeTags(composeContent, []tools.ServiceUpdate{{ServiceName: "database", ImageName: "postgres", OldTag: "15", NewTag: "16"}})

	assert.Nil(t, actualContent)
	assert.NotNil(t, err)
}
