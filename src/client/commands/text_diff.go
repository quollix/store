package commands

import (
	"strings"

	u "github.com/quollix/common/utils"
)

const (
	TextDiffOldContentEmptyError = "old content is empty"
	TextDiffNewContentEmptyError = "new content is empty"
	TextDiffContentsEqualError   = "contents are equal"
)

func DiffText(oldContent string, newContent string) (string, error) {
	if oldContent == "" {
		return "", u.Logger.NewError(TextDiffOldContentEmptyError)
	}
	if newContent == "" {
		return "", u.Logger.NewError(TextDiffNewContentEmptyError)
	}

	oldLines := splitDiffLines(oldContent)
	newLines := splitDiffLines(newContent)
	if areStringSlicesEqual(oldLines, newLines) {
		return "", u.Logger.NewError(TextDiffContentsEqualError)
	}

	lcsLengths := buildLCSLengths(oldLines, newLines)
	diffLines := buildDiffLines(oldLines, newLines, lcsLengths)
	return strings.Join(diffLines, "\n") + "\n", nil
}

func splitDiffLines(content string) []string {
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
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

func buildDiffLines(oldLines []string, newLines []string, lcsLengths [][]int) []string {
	diffLines := make([]string, 0, len(oldLines)+len(newLines))
	i := len(oldLines)
	j := len(newLines)

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			diffLines = append(diffLines, " "+oldLines[i-1])
			i--
			j--
			continue
		}
		if j > 0 && (i == 0 || lcsLengths[i][j-1] >= lcsLengths[i-1][j]) {
			diffLines = append(diffLines, "+"+newLines[j-1])
			j--
			continue
		}
		diffLines = append(diffLines, "-"+oldLines[i-1])
		i--
	}

	reverseStrings(diffLines)
	return diffLines
}

func reverseStrings(values []string) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}
