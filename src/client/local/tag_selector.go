package local

import (
	"strconv"
	"strings"

	"qsc/tools"

	u "github.com/quollix/common/utils"
)

type TagSelector interface {
	SelectLatestTag(originalTag string, tagList []string) (string, bool, error)
	Parse(tag string) ([]int, error)
}

type TagSelectorImpl struct{}

func (t *TagSelectorImpl) SelectLatestTag(originalTag string, tagList []string) (string, bool, error) {
	prefix, coreTagNumbers, suffix := tools.SplitTag(originalTag)
	originalTagNumbers, err := t.Parse(coreTagNumbers)
	if err != nil {
		return "", false, u.Logger.AddContext(err, tools.TagField, originalTag)
	}

	numericSuffix, hasNumericSuffix := parseNumericSuffix(suffix)
	candidates := []parsedTag{{
		raw:              originalTag,
		numbers:          originalTagNumbers,
		numericSuffix:    numericSuffix,
		hasNumericSuffix: hasNumericSuffix,
	}}
	candidates = append(candidates, t.filterCompatibleTags(tagList, prefix, suffix)...)

	highestTag := t.findMaxParsedTag(len(originalTagNumbers), candidates)
	if highestTag == nil || highestTag.raw == originalTag {
		return "", false, nil
	}
	return highestTag.raw, true, nil
}

type parsedTag struct {
	raw              string
	numbers          []int
	numericSuffix    int
	hasNumericSuffix bool
}

func (t *TagSelectorImpl) filterCompatibleTags(tagList []string, prefix, suffix string) []parsedTag {
	candidates := make([]parsedTag, 0, len(tagList))
	for _, tag := range tagList {
		candidatePrefix, coreTagNumbers, candidateSuffix := tools.SplitTag(tag)
		if candidatePrefix != prefix || !compatibleSuffix(suffix, candidateSuffix) {
			continue
		}
		numbers, err := t.Parse(coreTagNumbers)
		if err != nil {
			continue
		}
		numericSuffix, hasNumericSuffix := parseNumericSuffix(candidateSuffix)
		candidates = append(candidates, parsedTag{
			raw:              tag,
			numbers:          numbers,
			numericSuffix:    numericSuffix,
			hasNumericSuffix: hasNumericSuffix,
		})
	}
	return candidates
}

func compatibleSuffix(originalSuffix, candidateSuffix string) bool {
	if originalSuffix == candidateSuffix {
		return true
	}
	if isAlpineVersionSuffix(originalSuffix) && isAlpineVersionSuffix(candidateSuffix) {
		return true
	}
	_, originalOk := parseNumericSuffix(originalSuffix)
	_, candidateOk := parseNumericSuffix(candidateSuffix)
	return originalOk && candidateOk
}

func isAlpineVersionSuffix(suffix string) bool {
	const prefix = "-alpine"
	if !strings.HasPrefix(suffix, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(suffix, prefix), ".")
	if len(parts) != 2 {
		return false
	}
	_, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	_, err = strconv.Atoi(parts[1])
	return err == nil
}

func parseNumericSuffix(suffix string) (int, bool) {
	if !strings.HasPrefix(suffix, "-") {
		return 0, false
	}
	value := strings.TrimPrefix(suffix, "-")
	if value == "" || strings.Contains(value, ".") {
		return 0, false
	}
	number, err := strconv.Atoi(value)
	return number, err == nil
}

func (t *TagSelectorImpl) IntSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (t *TagSelectorImpl) Parse(tag string) ([]int, error) {
	parts := strings.Split(tag, ".")
	ints := make([]int, len(parts))
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, u.Logger.NewError("integer conversion failed", "details", err.Error())
		}
		ints[i] = n
	}
	return ints, nil
}

func (t *TagSelectorImpl) findMaxParsedTag(desiredLength int, tags []parsedTag) *parsedTag {
	if len(tags) == 0 {
		return nil
	}
	var maxTag *parsedTag
	for i := range tags {
		tag := &tags[i]
		if len(tag.numbers) != desiredLength {
			continue
		}
		if maxTag == nil {
			maxTag = tag
			continue
		}
		if isHigherParsedTag(tag, maxTag) {
			maxTag = tag
		}
	}
	return maxTag
}

func isHigherParsedTag(candidate, current *parsedTag) bool {
	for i := range candidate.numbers {
		if candidate.numbers[i] > current.numbers[i] {
			return true
		}
		if candidate.numbers[i] < current.numbers[i] {
			return false
		}
	}
	if candidate.hasNumericSuffix && current.hasNumericSuffix {
		return candidate.numericSuffix > current.numericSuffix
	}
	return false
}
