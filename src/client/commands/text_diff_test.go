package commands

import (
	"fmt"
	"testing"

	"qsc/tools"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestSplitDiffLines(t *testing.T) {
	assert.Equal(t, []string(nil), SplitDiffLines(""))
	assert.Equal(t, []string{"a", "b"}, SplitDiffLines("a\nb"))
	assert.Equal(t, []string{"a", "b"}, SplitDiffLines("a\nb\n"))
}

func TestDiffLines(t *testing.T) {
	tests := []struct {
		name                 string
		oldLines             []string
		newLines             []string
		expectedErrorMessage string
		expectedDiff         []DiffLine
	}{
		{
			name:                 "old content is empty",
			newLines:             []string{"a"},
			expectedErrorMessage: TextDiffOldContentEmptyError,
		},
		{
			name:                 "new content is empty",
			oldLines:             []string{"a"},
			expectedErrorMessage: TextDiffNewContentEmptyError,
		},
		{
			name:                 "contents are equal",
			oldLines:             []string{"a", "b"},
			newLines:             []string{"a", "b"},
			expectedErrorMessage: TextDiffContentsEqualError,
		},
		{
			name:         "addition",
			oldLines:     []string{"a", "c"},
			newLines:     []string{"a", "b", "c"},
			expectedDiff: diffLines("a", "+b", "c"),
		},
		{
			name:         "deletion",
			oldLines:     []string{"a", "b", "c"},
			newLines:     []string{"a", "c"},
			expectedDiff: diffLines("a", "-b", "c"),
		},
		{
			name:         "modification",
			oldLines:     []string{"a", "b", "c"},
			newLines:     []string{"a", "x", "c"},
			expectedDiff: diffLines("a", "-b", "+x", "c"),
		},
		{
			name:         "multiple separated changes",
			oldLines:     []string{"a", "b", "c", "d", "e", "f", "g"},
			newLines:     []string{"a", "x", "c", "e", "y", "z", "g"},
			expectedDiff: diffLines("a", "-b", "+x", "c", "-d", "e", "-f", "+y", "+z", "g"),
		},
		{
			name:         "repeated lines",
			oldLines:     []string{"a", "x", "a", "b"},
			newLines:     []string{"a", "x", "a", "c"},
			expectedDiff: diffLines("a", "x", "a", "-b", "+c"),
		},
		{
			name:         "changed repeated line",
			oldLines:     []string{"a", "b", "a", "c"},
			newLines:     []string{"a", "b", "x", "c"},
			expectedDiff: diffLines("a", "b", "-a", "+x", "c"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualDiff, err := DiffLines(test.oldLines, test.newLines)

			if test.expectedErrorMessage != "" {
				assert.Equal(t, test.expectedErrorMessage, u.ExtractError(err))
				assert.Equal(t, []DiffLine(nil), actualDiff)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, test.expectedDiff, actualDiff)
		})
	}
}

func TestRenderDiffColorsChangesAndCollapsesUnchangedLines(t *testing.T) {
	lines := diffLines(
		"before omitted 1",
		"before omitted 2",
		"before 1",
		"before 2",
		"before 3",
		"-old",
		"+new",
		"after 1",
		"after 2",
		"after 3",
		"after omitted 1",
		"after omitted 2",
	)

	assert.Equal(t, fmt.Sprintf(`...
 before 1
 before 2
 before 3
%s-old%s
%s+new%s
 after 1
 after 2
 after 3
...
`, tools.AnsiRed, tools.AnsiReset, tools.AnsiGreen, tools.AnsiReset), RenderDiff(lines))
}

func diffLines(lines ...string) []DiffLine {
	result := make([]DiffLine, 0, len(lines))
	for _, line := range lines {
		kind := DiffLineUnchanged
		text := line
		switch line[0] {
		case '+':
			kind = DiffLineAdded
			text = line[1:]
		case '-':
			kind = DiffLineDeleted
			text = line[1:]
		}
		result = append(result, DiffLine{Kind: kind, Text: text})
	}
	return result
}
