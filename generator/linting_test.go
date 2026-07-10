package generator

import (
	"testing"
)

func TestLintMarkdown(t *testing.T) {
	backticks := "```"
	validMD := "\n# Header\nSome text.\n\n" + backticks + "json\n{\"key\": \"value\"}\n" + backticks + "\n\n| Col 1 | Col 2 |\n|-------|-------|\n| Val 1 | Val 2 |\n"
	if err := LintMarkdown(validMD); err != nil {
		t.Errorf("expected valid markdown to pass, got: %v", err)
	}

	invalidCodeBlock := backticks + "go\nfmt.Println(1)"
	if err := LintMarkdown(invalidCodeBlock); err == nil {
		t.Error("expected failure for unbalanced code block")
	}

	invalidTable := "\n| Col 1 | Col 2 |\n|-------|-------|\n| Val 1 |\n"
	if err := LintMarkdown(invalidTable); err == nil {
		t.Error("expected failure for mismatched table columns")
	}
}
