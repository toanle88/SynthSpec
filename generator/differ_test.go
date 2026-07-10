package generator

import (
	"strings"
	"testing"
)

func TestGenerateLineDiff(t *testing.T) {
	oldText := "line 1\nline 2\nline 3"
	newText := "line 1\nline 2 modified\nline 3\nline 4"

	diff := GenerateLineDiff(oldText, newText)
	lines := strings.Split(diff, "\n")

	hasMinus := false
	hasPlus := false
	for _, l := range lines {
		if strings.HasPrefix(l, "-line 2") {
			hasMinus = true
		}
		if strings.HasPrefix(l, "+line 2 modified") {
			hasPlus = true
		}
	}

	if !hasMinus || !hasPlus {
		t.Errorf("Diff did not capture changes correctly. Output:\n%s", diff)
	}
}
