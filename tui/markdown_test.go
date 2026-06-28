package tui

import (
	"strings"
	"testing"
)

func TestHighlightMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check string // substring that should be present
	}{
		{
			name:  "h1 header",
			input: "# Title",
			check: "# Title",
		},
		{
			name:  "h2 header",
			input: "## Section",
			check: "## Section",
		},
		{
			name:  "h3 header",
			input: "### Subsection",
			check: "### Subsection",
		},
		{
			name:  "bullet list",
			input: "- item one\n- item two",
			check: "item one",
		},
		{
			name:  "numbered list",
			input: "1. first\n2. second",
			check: "first",
		},
		{
			name:  "bold text",
			input: "This is **bold** text",
			check: "bold",
		},
		{
			name:  "italic text",
			input: "This is *italic* text",
			check: "italic",
		},
		{
			name:  "inline code",
			input: "Use `code` here",
			check: "code",
		},
		{
			name:  "link",
			input: "[click me](https://example.com)",
			check: "click me",
		},
		{
			name:  "code block with Go",
			input: "```go\nfunc main() {}\n```",
			check: "func",
		},
		{
			name:  "code block with JSON",
			input: "```json\n{\"key\": \"value\"}\n```",
			check: "key",
		},
		{
			name:  "code block with YAML",
			input: "```yaml\nkey: value\n```",
			check: "key",
		},
		{
			name:  "mixed content",
			input: "# Title\n\nSome **bold** and *italic* text\n- item\n`code`",
			check: "Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HighlightMarkdown(tt.input)
			if !strings.Contains(result, tt.check) {
				t.Errorf("expected output to contain %q, got %q", tt.check, result)
			}
		})
	}
}

func TestHighlightMarkdown_Empty(t *testing.T) {
	result := HighlightMarkdown("")
	if result != "" {
		t.Errorf("expected empty output for empty input, got %q", result)
	}
}

func TestHighlightCodeLine_Go(t *testing.T) {
	input := "func main() string {"
	result := highlightCodeLine(input, "go")
	if !strings.Contains(result, "func") {
		t.Errorf("expected 'func' to be present in highlighted Go code")
	}
}

func TestHighlightCodeLine_JSON(t *testing.T) {
	input := `"key": "value"`
	result := highlightCodeLine(input, "json")
	if !strings.Contains(result, "key") {
		t.Errorf("expected 'key' to be present in highlighted JSON")
	}
}

func TestHighlightCodeLine_Default(t *testing.T) {
	input := "# comment"
	result := highlightCodeLine(input, "")
	if !strings.Contains(result, "comment") {
		t.Errorf("expected 'comment' to be present in highlighted code")
	}
}

func TestHighlightInline(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"bold only", "**bold**"},
		{"italic only", "*italic*"},
		{"code only", "`code`"},
		{"link only", "[text](url)"},
		{"mixed", "**bold** and *italic* and `code`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlightInline(tt.input)
			// Should not drop content
			if tt.input != "" && result == "" {
				t.Errorf("expected non-empty result for %q", tt.input)
			}
		})
	}
}

func TestReplaceWord(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		oldWord string
		newWord string
		want    string
	}{
		{"simple replace", "func main", "func", "FUNC", "FUNC main"},
		{"not word boundary", "gofunc", "func", "FUNC", "gofunc"},
		{"no match", "hello world", "func", "FUNC", "hello world"},
		{"multiple matches", "func main func test", "func", "FUNC", "FUNC main FUNC test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceWord(tt.text, tt.oldWord, tt.newWord)
			if got != tt.want {
				t.Errorf("replaceWord(%q, %q, %q) = %q, want %q", tt.text, tt.oldWord, tt.newWord, got, tt.want)
			}
		})
	}
}
