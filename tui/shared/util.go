package shared

import (
	"regexp"
	"strings"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from a string
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// FuzzyMatch returns true if all characters of query appear in s in order
func FuzzyMatch(s, query string) bool {
	s = strings.ToLower(s)
	query = strings.ToLower(query)
	if query == "" {
		return true
	}
	sRunes := []rune(s)
	qRunes := []rune(query)
	sIdx := 0
	for _, qRune := range qRunes {
		found := false
		for sIdx < len(sRunes) {
			if sRunes[sIdx] == qRune {
				found = true
				sIdx++
				break
			}
			sIdx++
		}
		if !found {
			return false
		}
	}
	return true
}

// MinInt returns the smaller of two ints
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt returns the larger of two ints
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
