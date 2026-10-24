package tools

import (
	"testing"

	"github.com/quollix/common/assert"
)

func TestSplitTag(t *testing.T) {
	tests := []struct {
		name        string
		tag         string
		wantPrefix  string
		wantVersion string
		wantSuffix  string
	}{
		{"plain version", "1.2.3", "", "1.2.3", ""},
		{"v prefix", "v1.2.3", "v", "1.2.3", ""},
		{"stable prefix", stableTagPrefix + "1.2.3", stableTagPrefix, "1.2.3", ""},
		{"latest prefix", latestTagPrefix + "20261020", latestTagPrefix, "20261020", ""},
		{"version with suffix", "1.2.3-alpine", "", "1.2.3", "-alpine"},
		{"prefix and suffix", latestTagPrefix + "20261020-alpine", latestTagPrefix, "20261020", "-alpine"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, version, suffix := SplitTag(tt.tag)

			assert.Equal(t, tt.wantPrefix, prefix)
			assert.Equal(t, tt.wantVersion, version)
			assert.Equal(t, tt.wantSuffix, suffix)
		})
	}
}
