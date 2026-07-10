package generator

import (
	"errors"
	"fmt"
	"strings"
)

// LintMarkdown checks the structural integrity of a markdown string.
func LintMarkdown(content string) error {
	// 1. Check code blocks balance
	codeBlocksCount := strings.Count(content, "```")
	if codeBlocksCount%2 != 0 {
		return errors.New("malformed markdown: unbalanced code blocks (missing closing ```)")
	}

	// 2. Check markdown table formatting
	lines := strings.Split(content, "\n")
	inTable := false
	var expectedCols int

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Detect table start (has | and separator row)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			cols := strings.Count(trimmed, "|") - 1
			if !inTable {
				inTable = true
				expectedCols = cols
			} else {
				// Separator row like |---|---|
				if strings.Contains(trimmed, "-") && !strings.ContainsAny(trimmed, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
					continue
				}
				if cols != expectedCols {
					return fmt.Errorf("malformed table at line %d: column count mismatch (expected %d, got %d)", lineNum+1, expectedCols, cols)
				}
			}
		} else {
			inTable = false
		}
	}

	return nil
}
