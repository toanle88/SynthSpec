package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Aesthetic Color Palette (Dark Theme with Vibrant Accents)
var (
	ColorBg      = lipgloss.Color("#1a1a24") // Deep navy slate
	ColorBorder  = lipgloss.Color("#3e3e57") // Steel gray border
	ColorAccent  = lipgloss.Color("#7d56f4") // Vibrant Violet
	ColorSuccess = lipgloss.Color("#04d98b") // Emerald Green
	ColorWarning = lipgloss.Color("#f29c38") // Amber Orange
	ColorInfo    = lipgloss.Color("#00bbf9") // Cyan Blue
	ColorText    = lipgloss.Color("#e2e2e9") // Light off-white
	ColorMuted   = lipgloss.Color("#6c6c8c") // Muted gray
)

// UI Container Layout Styles
var (
	DocStyle = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText).
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1).
			Bold(true).
			Foreground(ColorAccent).
			Align(lipgloss.Center)

	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			Width(30)

	MainPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	ErrorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#ef4444")).
			Padding(0, 1)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	ThoughtBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Foreground(ColorMuted).
			Padding(0, 1).
			Italic(true)

	ThoughtTitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Bold(true)
)

// Specific UI Text Styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText)

	MetricLabelStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Bold(true)

	MetricScoreStyle = lipgloss.NewStyle().
				Bold(true)

	QuestionStyle = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true).
			Italic(true)

	InputPrefixStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)
)

// RenderProgressBar draws a colorful progress bar based on percentage
func RenderProgressBar(width int, percentage int) string {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	filledLength := (width * percentage) / 100
	emptyLength := width - filledLength

	filledChar := "█"
	emptyChar := "░"

	filledStr := strings.Repeat(filledChar, filledLength)
	emptyStr := strings.Repeat(emptyChar, emptyLength)

	// Pick color based on progress tier
	var color lipgloss.Color
	switch {
	case percentage == 100:
		color = ColorSuccess
	case percentage >= 50:
		color = ColorAccent
	case percentage >= 25:
		color = ColorInfo
	default:
		color = ColorWarning
	}

	barStyle := lipgloss.NewStyle().Foreground(color)
	scoreStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

	return fmt.Sprintf("%s%s %s",
		barStyle.Render(filledStr),
		lipgloss.NewStyle().Foreground(ColorBorder).Render(emptyStr),
		scoreStyle.Render(fmt.Sprintf("%3d%%", percentage)),
	)
}
