package commands

import (
	"strings"
	"testing"

	"github.com/quollix/common/assert"
	u "github.com/quollix/common/utils"
)

func TestDiffText(t *testing.T) {
	tests := []struct {
		name                 string
		oldContent           string
		newContent           string
		expectedErrorMessage string
		expectedDiff         string
	}{
		{
			name:                 "old content is empty",
			oldContent:           "",
			newContent:           "a\n",
			expectedErrorMessage: TextDiffOldContentEmptyError,
		},
		{
			name:                 "new content is empty",
			oldContent:           "a\n",
			newContent:           "",
			expectedErrorMessage: TextDiffNewContentEmptyError,
		},
		{
			name:                 "contents are equal",
			oldContent:           "a\nb\n",
			newContent:           "a\nb\n",
			expectedErrorMessage: TextDiffContentsEqualError,
		},
		{
			name: "addition",
			oldContent: `
a
c
`,
			newContent: `
a
b
c
`,
			expectedDiff: `
 a
+b
 c
`,
		},
		{
			name: "deletion",
			oldContent: `
a
b
c
`,
			newContent: `
a
c
`,
			expectedDiff: `
 a
-b
 c
`,
		},
		{
			name: "modification",
			oldContent: `
a
b
c
`,
			newContent: `
a
x
c
`,
			expectedDiff: `
 a
-b
+x
 c
`,
		},
		{
			name: "multiple separated changes",
			oldContent: `
a
b
c
d
e
f
g
`,
			newContent: `
a
x
c
e
y
z
g
`,
			expectedDiff: `
 a
-b
+x
 c
-d
 e
-f
+y
+z
 g
`,
		},
		{
			name: "repeated lines",
			oldContent: `
a
x
a
b
`,
			newContent: `
a
x
a
c
`,
			expectedDiff: `
 a
 x
 a
-b
+c
`,
		},
		{
			name: "changed repeated line",
			oldContent: `
a
b
a
c
`,
			newContent: `
a
b
x
c
`,
			expectedDiff: `
 a
 b
-a
+x
 c
`,
		},
		{
			name:                 "contents are equal without final trailing newline",
			oldContent:           "a\nb\n",
			newContent:           "a\nb",
			expectedErrorMessage: TextDiffContentsEqualError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualDiff, err := DiffText(normalizeTestTextBlock(test.oldContent), normalizeTestTextBlock(test.newContent))

			if test.expectedErrorMessage != "" {
				assert.Equal(t, test.expectedErrorMessage, u.ExtractError(err))
				assert.Equal(t, "", actualDiff)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, strings.TrimPrefix(test.expectedDiff, "\n"), actualDiff)
		})
	}
}

func normalizeTestTextBlock(text string) string {
	return strings.TrimPrefix(text, "\n")
}
