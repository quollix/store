package commands

import (
	"testing"
	"time"

	"github.com/quollix/common/assert"
	"github.com/quollix/common/store"
)

var versionSelectorBaseTimestamp = time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)

func TestOrderVersionsNewestFirst_SortsNewestTimestampFirst(t *testing.T) {
	versions := []store.LeanVersionDto{
		{VersionId: 1, CreationTimestamp: versionSelectorBaseTimestamp},
		{VersionId: 3, CreationTimestamp: versionSelectorBaseTimestamp.Add(2 * time.Minute)},
		{VersionId: 2, CreationTimestamp: versionSelectorBaseTimestamp.Add(time.Minute)},
	}

	orderedVersions := orderVersionsNewestFirst(versions)

	assert.Equal(t, 3, orderedVersions[0].VersionId)
	assert.Equal(t, 2, orderedVersions[1].VersionId)
	assert.Equal(t, 1, orderedVersions[2].VersionId)
}

func TestOrderVersionsNewestFirst_SortsHigherVersionIDFirstWhenTimestampMatches(t *testing.T) {
	versions := []store.LeanVersionDto{
		{VersionId: 1, CreationTimestamp: versionSelectorBaseTimestamp},
		{VersionId: 3, CreationTimestamp: versionSelectorBaseTimestamp},
		{VersionId: 2, CreationTimestamp: versionSelectorBaseTimestamp},
	}

	orderedVersions := orderVersionsNewestFirst(versions)

	assert.Equal(t, 3, orderedVersions[0].VersionId)
	assert.Equal(t, 2, orderedVersions[1].VersionId)
	assert.Equal(t, 1, orderedVersions[2].VersionId)
}

func TestOrderVersionsOldestFirst_SortsOldestTimestampFirst(t *testing.T) {
	versions := []store.LeanVersionDto{
		{VersionId: 1, CreationTimestamp: versionSelectorBaseTimestamp},
		{VersionId: 3, CreationTimestamp: versionSelectorBaseTimestamp.Add(2 * time.Minute)},
		{VersionId: 2, CreationTimestamp: versionSelectorBaseTimestamp.Add(time.Minute)},
	}

	orderedVersions := orderVersionsOldestFirst(versions)

	assert.Equal(t, 1, orderedVersions[0].VersionId)
	assert.Equal(t, 2, orderedVersions[1].VersionId)
	assert.Equal(t, 3, orderedVersions[2].VersionId)
}

func TestOrderVersionsOldestFirst_SortsLowerVersionIDFirstWhenTimestampMatches(t *testing.T) {
	versions := []store.LeanVersionDto{
		{VersionId: 1, CreationTimestamp: versionSelectorBaseTimestamp},
		{VersionId: 3, CreationTimestamp: versionSelectorBaseTimestamp},
		{VersionId: 2, CreationTimestamp: versionSelectorBaseTimestamp},
	}

	orderedVersions := orderVersionsOldestFirst(versions)

	assert.Equal(t, 1, orderedVersions[0].VersionId)
	assert.Equal(t, 2, orderedVersions[1].VersionId)
	assert.Equal(t, 3, orderedVersions[2].VersionId)
}

func TestParseTwoVersionIndices_AllowsZeroAsOldestVersionIndex(t *testing.T) {
	leftIndex, rightIndex, err := parseTwoVersionIndices("0 1", 3)

	assert.Nil(t, err)
	assert.Equal(t, 0, leftIndex)
	assert.Equal(t, 1, rightIndex)
}

func TestFormatVersionTimestamp_DropsSubsecondsAndTimezone(t *testing.T) {
	timestamp := time.Date(2026, 9, 4, 10, 11, 12, 123456789, time.UTC)

	assert.Equal(t, "2026-09-04 10:11:12", formatVersionTimestamp(timestamp))
}
