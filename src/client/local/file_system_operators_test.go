package local

import (
	"testing"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestFileSystemOperatorImpl_ParseServicesFromCompose_IncludesDigest(t *testing.T) {
	imageReferenceParserMock := NewImageReferenceParserMock(t)
	fileSystemOperator := &FileSystemOperatorImpl{ImageReferenceParser: imageReferenceParserMock}
	imageReferenceParserMock.EXPECT().ParseComposeImageReference("registry.example.com:5000/sample/app:1.2.3@" + testDigestA).Return(composeImageReference{
		Image:  "registry.example.com:5000/sample/app",
		Tag:    "1.2.3",
		Digest: testDigestA,
	})

	services, err := fileSystemOperator.ParseServicesFromCompose([]byte(`
services:
  web:
    image: registry.example.com:5000/sample/app:1.2.3@` + testDigestA + `
`))

	assert.Nil(t, err)
	assert.Equal(t, 1, len(services))
	assert.Equal(t, "web", services[0].Name)
	assert.Equal(t, "registry.example.com:5000/sample/app", services[0].Image)
	assert.Equal(t, "1.2.3", services[0].Tag)
	assert.Equal(t, testDigestA, services[0].Digest)
}

func TestFileSystemOperatorImpl_ParseServicesFromCompose_ReturnsErrorWhenServicesFieldIsMissing(t *testing.T) {
	fileSystemOperator := &FileSystemOperatorImpl{}

	services, err := fileSystemOperator.ParseServicesFromCompose([]byte(`volumes: {}`))

	assert.Nil(t, services)
	assert.Equal(t, "'services' field missing in docker-compose.yml", u.ExtractError(err))
}
