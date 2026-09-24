package commands

import (
	"strings"

	"qsc/tools"

	u "github.com/quollix/common/utils"
)

const (
	TextDiffOldContentEmptyError = "old content is empty"
	TextDiffNewContentEmptyError = "new content is empty"
	TextDiffContentsEqualError   = "contents are equal"
	diffContextLineCount         = 3
)

type DiffLineKind int

const (
	DiffLineUnchanged DiffLineKind = iota
	DiffLineAdded
	DiffLineDeleted
)

type DiffLine struct {
	Kind DiffLineKind
	Text string
}

func SplitDiffLines(content string) []string {
	if content == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

func DiffLines(oldLines []string, newLines []string) ([]DiffLine, error) {
	if len(oldLines) == 0 {
		return nil, u.Logger.NewError(TextDiffOldContentEmptyError)
	}
	if len(newLines) == 0 {
		return nil, u.Logger.NewError(TextDiffNewContentEmptyError)
	}

	if areStringSlicesEqual(oldLines, newLines) {
		return nil, u.Logger.NewError(TextDiffContentsEqualError)
	}

	lcsLengths := buildLCSLengths(oldLines, newLines)
	return buildDiffLines(oldLines, newLines, lcsLengths), nil
}

func RenderDiff(lines []DiffLine) string {
	included := diffLineIndicesToRender(lines)
	var builder strings.Builder
	omittedLines := false
	for i, line := range lines {
		if !included[i] {
			omittedLines = true
			continue
		}
		if omittedLines {
			builder.WriteString("...\n")
			omittedLines = false
		}
		renderDiffLine(&builder, line)
	}
	if omittedLines && builder.Len() > 0 {
		builder.WriteString("...\n")
	}
	return builder.String()
}

func diffLineIndicesToRender(lines []DiffLine) []bool {
	included := make([]bool, len(lines))
	for i, line := range lines {
		if line.Kind == DiffLineUnchanged {
			continue
		}
		start := max(0, i-diffContextLineCount)
		end := min(len(lines), i+diffContextLineCount+1)
		for j := start; j < end; j++ {
			included[j] = true
		}
	}
	return included

}

func renderDiffLine(builder *strings.Builder, line DiffLine) {
	switch line.Kind {
	case DiffLineAdded:
		builder.WriteString(tools.AnsiGreen + "+" + line.Text + tools.AnsiReset + "\n")
	case DiffLineDeleted:
		builder.WriteString(tools.AnsiRed + "-" + line.Text + tools.AnsiReset + "\n")
	default:
		builder.WriteString(" " + line.Text + "\n")
	}
}

func areStringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func buildLCSLengths(oldLines []string, newLines []string) [][]int {
	lcsLengths := make([][]int, len(oldLines)+1)
	for i := range lcsLengths {
		lcsLengths[i] = make([]int, len(newLines)+1)
	}

	for i := 1; i <= len(oldLines); i++ {
		for j := 1; j <= len(newLines); j++ {
			if oldLines[i-1] == newLines[j-1] {
				lcsLengths[i][j] = lcsLengths[i-1][j-1] + 1
				continue
			}
			lcsLengths[i][j] = max(lcsLengths[i-1][j], lcsLengths[i][j-1])
		}
	}
	return lcsLengths
}

func buildDiffLines(oldLines []string, newLines []string, lcsLengths [][]int) []DiffLine {
	diffLines := make([]DiffLine, 0, len(oldLines)+len(newLines))
	i := len(oldLines)
	j := len(newLines)

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			diffLines = append(diffLines, DiffLine{Kind: DiffLineUnchanged, Text: oldLines[i-1]})
			i--
			j--
			continue
		}
		if j > 0 && (i == 0 || lcsLengths[i][j-1] >= lcsLengths[i-1][j]) {
			diffLines = append(diffLines, DiffLine{Kind: DiffLineAdded, Text: newLines[j-1]})
			j--
			continue
		}
		diffLines = append(diffLines, DiffLine{Kind: DiffLineDeleted, Text: oldLines[i-1]})
		i--
	}

	reverseDiffLines(diffLines)
	return diffLines
}

func reverseDiffLines(values []DiffLine) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}
