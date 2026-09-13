package local

import (
	"testing"

	"github.com/quollix/common/assert"
)

func TestTagSelectorImpl_SelectLatestTag_ReturnsHighestCompatibleTag(t *testing.T) {
	tagSelector := &TagSelectorImpl{}

	tag, ok, err := tagSelector.SelectLatestTag("1.2.3-alpine", []string{"1.2.4-alpine", "1.3.0", "1.2.5-bookworm"})

	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, "1.2.4-alpine", tag)
}

func TestTagSelectorImpl_SelectLatestTag_AllowsAlpineVersionSuffixUpdates(t *testing.T) {
	tagSelector := &TagSelectorImpl{}

	tag, ok, err := tagSelector.SelectLatestTag("1.6.42-alpine3.23", []string{"1.6.45-alpine3.24"})

	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, "1.6.45-alpine3.24", tag)
}

func TestTagSelectorImpl_SelectLatestTag_UsesNumericSuffix(t *testing.T) {
	tagSelector := &TagSelectorImpl{}

	tag, ok, err := tagSelector.SelectLatestTag("12.1-0", []string{"12.1-1", "12.1-2", "12.2-beta"})

	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, "12.1-2", tag)
}

func TestTagSelectorImpl_SelectLatestTag_ReturnsNoUpdateWhenCurrentTagIsHighest(t *testing.T) {
	tagSelector := &TagSelectorImpl{}

	tag, ok, err := tagSelector.SelectLatestTag("1.2.3", []string{"1.2.2", "1.2.3"})

	assert.Nil(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", tag)
}

func TestTagSelectorImpl_SelectLatestTag_ReturnsErrorWhenOriginalTagIsInvalid(t *testing.T) {
	tagSelector := &TagSelectorImpl{}

	tag, ok, err := tagSelector.SelectLatestTag("latest", []string{"1.2.3"})

	assert.Equal(t, "", tag)
	assert.False(t, ok)
	assert.NotNil(t, err)
}
