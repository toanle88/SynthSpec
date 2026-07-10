package generator

import (
	"strings"

	"github.com/toanle/synthspec/domain"
)

// GenerateLineDiff calculates a line-by-line LCS diff between two texts.
func GenerateLineDiff(oldText, newText string) string {
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	m := len(oldLines)
	n := len(newLines)

	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else {
				if lcs[i-1][j] > lcs[i][j-1] {
					lcs[i][j] = lcs[i-1][j]
				} else {
					lcs[i][j] = lcs[i][j-1]
				}
			}
		}
	}

	var diff []string
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			diff = append([]string{"  " + oldLines[i-1]}, diff...)
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			diff = append([]string{"+" + newLines[j-1]}, diff...)
			j--
		} else if i > 0 && (j == 0 || lcs[i][j-1] < lcs[i-1][j]) {
			diff = append([]string{"-" + oldLines[i-1]}, diff...)
			i--
		}
	}
	return strings.Join(diff, "\n")
}

// ComputeDiff calculates FileDiff between old and new contents.
func ComputeDiff(fileName, oldContent, newContent string) domain.FileDiff {
	return domain.FileDiff{
		FileName:   fileName,
		OldContent: oldContent,
		NewContent: newContent,
		DiffText:   GenerateLineDiff(oldContent, newContent),
	}
}
