package shared

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleH1     = lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Underline(true)
	styleH2     = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	styleH3     = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	styleH4     = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	styleBullet = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	styleCode   = lipgloss.NewStyle().Foreground(ColorWarning).Background(lipgloss.Color("#2a2a37"))

	// Syntax highlighting styles for code blocks
	styleKeyword = lipgloss.NewStyle().Foreground(ColorInfo).Bold(true)
	styleString  = lipgloss.NewStyle().Foreground(ColorSuccess)
	styleComment = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)
	styleNumber  = lipgloss.NewStyle().Foreground(ColorWarning)
	styleType    = lipgloss.NewStyle().Foreground(ColorAccent)
)

var (
	reBold   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reItalic = regexp.MustCompile(`\*([^*]+)\*`)
	reCode   = regexp.MustCompile("`([^`]+)`")
	reLink   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

var goKeywords = map[string]bool{
	"package": true, "import": true, "func": true, "return": true, "type": true,
	"struct": true, "interface": true, "map": true, "chan": true, "go": true,
	"select": true, "switch": true, "case": true, "default": true, "if": true,
	"else": true, "for": true, "range": true, "var": true, "const": true,
}

var goTypes = map[string]bool{
	"string": true, "int": true, "bool": true, "error": true, "byte": true,
	"rune": true, "float64": true, "nil": true, "true": true, "false": true,
}

// HighlightMarkdown highlights Markdown elements with Lipgloss/ANSI styles
func HighlightMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inCodeBlock := false
	codeBlockLang := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Code Block Toggle
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				codeBlockLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			} else {
				codeBlockLang = ""
			}
			// Style code block fence
			result = append(result, lipgloss.NewStyle().Foreground(ColorMuted).Render(line))
			continue
		}

		if inCodeBlock {
			result = append(result, highlightCodeLine(line, codeBlockLang))
			continue
		}

		result = append(result, highlightNonCodeLine(line, trimmed))
	}

	return strings.Join(result, "\n")
}

func highlightNonCodeLine(line, trimmed string) string {
	// Header checks
	if strings.HasPrefix(line, "# ") {
		return styleH1.Render(line)
	}
	if strings.HasPrefix(line, "## ") {
		return styleH2.Render(line)
	}
	if strings.HasPrefix(line, "### ") {
		return styleH3.Render(line)
	}
	if strings.HasPrefix(line, "#### ") || strings.HasPrefix(line, "##### ") || strings.HasPrefix(line, "###### ") {
		return styleH4.Render(line)
	}

	// Bullet lists
	if strings.HasPrefix(trimmed, "- ") {
		return highlightBullet(line, "- ")
	}
	if strings.HasPrefix(trimmed, "* ") {
		return highlightBullet(line, "* ")
	}

	// Numbered lists (regex check for digits + dot)
	matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed)
	if matched {
		return highlightNumbered(line, trimmed)
	}

	return highlightInline(line)
}

func highlightBullet(line, marker string) string {
	bulletIdx := strings.Index(line, marker)
	prefix := line[:bulletIdx]
	content := line[bulletIdx+len(marker):]
	return prefix + styleBullet.Render("• ") + highlightInline(content)
}

func highlightNumbered(line, trimmed string) string {
	dotIdx := strings.Index(trimmed, ". ")
	number := trimmed[:dotIdx+1]
	content := trimmed[dotIdx+2:]
	indentIdx := strings.Index(line, trimmed)
	prefix := line[:indentIdx]
	return prefix + styleBullet.Render(number+" ") + highlightInline(content)
}

func highlightInline(text string) string {
	if text == "" {
		return ""
	}

	// Format inline code backticks: use styleCode
	text = reCode.ReplaceAllStringFunc(text, func(m string) string {
		inner := m[1 : len(m)-1]
		return styleCode.Render(inner)
	})

	// Format bold: **text**
	text = reBold.ReplaceAllStringFunc(text, func(m string) string {
		inner := m[2 : len(m)-2]
		return lipgloss.NewStyle().Bold(true).Render(inner)
	})

	// Format italic: *text*
	text = reItalic.ReplaceAllStringFunc(text, func(m string) string {
		inner := m[1 : len(m)-1]
		return lipgloss.NewStyle().Italic(true).Render(inner)
	})

	// Format links: [text](url) -> cyan text + muted url
	text = reLink.ReplaceAllStringFunc(text, func(m string) string {
		matches := reLink.FindStringSubmatch(m)
		if len(matches) >= 3 {
			styledText := lipgloss.NewStyle().Foreground(ColorInfo).Underline(true).Render(matches[1])
			styledUrl := lipgloss.NewStyle().Foreground(ColorMuted).Render("(" + matches[2] + ")")
			return styledText + " " + styledUrl
		}
		return m
	})

	return text
}

func highlightCodeLine(line string, lang string) string {
	lang = strings.ToLower(lang)
	if line == "" {
		return ""
	}

	// Basic generic highlighter styles
	switch lang {
	case "go":
		return highlightGo(line)
	case "json", "yaml", "yml":
		return highlightJSONYAML(line)
	default:
		// Generic syntax highlighter
		return highlightGeneric(line)
	}
}

func highlightGo(line string) string {
	// Simple syntax highlight for Go keywords, strings, comments
	if strings.HasPrefix(strings.TrimSpace(line), "//") || strings.HasPrefix(strings.TrimSpace(line), "/*") {
		return styleComment.Render(line)
	}

	// Highlight comments at the end of line
	commentIdx := strings.Index(line, "//")
	codePart := line
	commentPart := ""
	if commentIdx >= 0 {
		codePart = line[:commentIdx]
		commentPart = styleComment.Render(line[commentIdx:])
	}

	words := strings.Fields(codePart)
	for _, w := range words {
		clean := strings.Trim(w, "(){}[],.;:")
		if goKeywords[clean] {
			codePart = replaceWord(codePart, clean, styleKeyword.Render(clean))
		} else if goTypes[clean] {
			codePart = replaceWord(codePart, clean, styleType.Render(clean))
		}
	}

	// Highlight string literals in Go
	reStr := regexp.MustCompile(`"([^"]*)"`)
	codePart = reStr.ReplaceAllStringFunc(codePart, func(m string) string {
		return styleString.Render(m)
	})

	return codePart + commentPart
}

func highlightJSONYAML(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
		return styleComment.Render(line)
	}

	// JSON / YAML key highlighting
	// Match "key": or key:
	reKey := regexp.MustCompile(`^(\s*)"?([a-zA-Z0-9_\-\.\/]+)"?\s*:\s*`)
	if reKey.MatchString(line) {
		matches := reKey.FindStringSubmatch(line)
		indent := matches[1]
		key := matches[2]
		rest := line[len(matches[0]):]

		styledKey := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(key)
		rest = highlightJSONYAMLValue(rest)
		return indent + styledKey + ": " + rest
	}

	// Fallback/Generic inside json/yaml
	return highlightGeneric(line)
}

func highlightJSONYAMLValue(rest string) string {
	if strings.HasPrefix(strings.TrimSpace(rest), `"`) {
		return styleString.Render(rest)
	}
	trimmedRest := strings.TrimSpace(rest)
	if trimmedRest == "true" || trimmedRest == "false" || trimmedRest == "null" {
		return styleKeyword.Render(rest)
	}
	if matched, _ := regexp.MatchString(`^\s*\d+(\.\d+)?`, rest); matched {
		return styleNumber.Render(rest)
	}
	return rest
}

func highlightGeneric(line string) string {
	// Highlight comments
	if strings.HasPrefix(strings.TrimSpace(line), "#") || strings.HasPrefix(strings.TrimSpace(line), "//") {
		return styleComment.Render(line)
	}

	// Highlight string literals
	reStr := regexp.MustCompile(`"([^"]*)"`)
	line = reStr.ReplaceAllStringFunc(line, func(m string) string {
		return styleString.Render(m)
	})

	return line
}

// replaceWord replaces a whole word in a string to avoid partial replacements (e.g. replacing "go" inside "go.mod")
func replaceWord(text, oldWord, newWord string) string {
	// Simple boundaries
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(oldWord) + `\b`)
	return re.ReplaceAllString(text, newWord)
}
