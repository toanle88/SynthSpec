package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

const pendingStr = "⏳ Pending"

func (m DashboardModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing SynthSpec Dashboard..."
	}

	header := m.renderHeader()
	sidebar := m.renderSidebar()
	mainChat := m.renderMainChat()
	footer := m.renderFooter()

	// Compute widths
	sidebarWidth := 34
	chatWidth := m.width - sidebarWidth - 8
	if chatWidth < 40 {
		chatWidth = 40
	}

	// Height limits
	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 4
	if bodyHeight < 10 {
		bodyHeight = 10
	}

	var body string
	if m.showViewer && m.isFullScreenViewer {
		styledChat := MainPanelStyle.
			Width(m.width - 6).
			Height(bodyHeight).
			Render(mainChat)
		body = styledChat
	} else {
		// Apply styles and dimensions
		styledSidebar := SidebarStyle.
			Width(sidebarWidth).
			Height(bodyHeight).
			Render(sidebar)

		styledChat := MainPanelStyle.
			Width(chatWidth).
			Height(bodyHeight).
			Render(mainChat)

		// Combine body horizontally
		body = lipgloss.JoinHorizontal(lipgloss.Top, styledSidebar, styledChat)
	}

	// Combine everything vertically
	fullView := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	return DocStyle.Render(fullView)
}

func (m DashboardModel) renderHeader() string {
	title := " 🛠️  SynthSpec Solution Architect Dashboard "

	// Average Score Calculation
	avgScore := (m.Session.Scores.Functional +
		m.Session.Scores.Structural +
		m.Session.Scores.Security +
		m.Session.Scores.Compliance) / 4

	meta := fmt.Sprintf("Project: %s | Provider: %s | Model: %s",
		lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render(m.Session.ProjectName),
		strings.ToUpper(m.Session.Provider),
		m.Session.Model,
	)

	progBar := RenderProgressBar(40, avgScore)

	content := fmt.Sprintf("%s\n%s\nOverall Progress: %s", title, meta, progBar)

	return HeaderStyle.Width(m.width - 4).Render(content)
}

func (m DashboardModel) renderSidebar() string {
	var sections []string

	sections = append(sections, TitleStyle.Render("⚡ Requirement Dimensions"))

	dimensions := []struct {
		Name      string
		Score     int
		Rationale string
	}{
		{"Functional", m.Session.Scores.Functional, m.Session.Rationales.Functional},
		{"Structural", m.Session.Scores.Structural, m.Session.Rationales.Structural},
		{"Security", m.Session.Scores.Security, m.Session.Rationales.Security},
		{"Compliance", m.Session.Scores.Compliance, m.Session.Rationales.Compliance},
	}

	for _, d := range dimensions {
		prog := RenderProgressBar(18, d.Score)
		label := MetricLabelStyle.Render(d.Name)

		// Wrap rationale text
		rationale := d.Rationale
		if rationale == "" {
			rationale = "No feedback generated yet."
		}
		wrappedRationale := wrapText(rationale, 30)
		mutedRationale := lipgloss.NewStyle().Foreground(ColorMuted).Render(wrappedRationale)

		sections = append(sections, fmt.Sprintf("%s\n%s\n%s", label, prog, mutedRationale))
	}

	return strings.Join(sections, "\n\n")
}

func (m DashboardModel) renderGeneratingState() string {
	var content []string
	content = append(content, TitleStyle.Render("✨ Final Asset Synthesis in Progress"))
	switch m.genPhase {
	case "source":
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("Phase: Source document lock-in"))
	case "parallel":
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("Phase: Parallel downstream generation"))
		if len(m.genFiles) > 1 {
			content = append(content, lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("Fan-out active: %d downstream documents running in parallel.", len(m.genFiles)-1)))
		}
	}
	if m.isWaitingApproval {
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("⏸️  THE DOMAIN APPROVAL GATE ACTIVE"))
		content = append(content, "The source domain model has been generated and validated.")
		content = append(content, "Please review the file and approve it to proceed with downstream parallel generation:\n")
		content = append(content, lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("  Press [V] to View 01_domain_model_use_cases.md"))
		content = append(content, lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("  Press [E] to Edit 01_domain_model_use_cases.md"))
		content = append(content, lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("  Press [A] or [Enter] to Approve and Resume Downstream Synthesis"))
	} else {
		content = append(content, fmt.Sprintf("\n%s Running generative model downstream...", m.spinner.View()))
	}
	content = append(content, "\n"+TitleStyle.Render("📂 Document Synthesis Progress:"))
	content = append(content, m.renderFileProgressList())
	content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("Status: "+m.genStatus))
	content = append(content, "\n"+TitleStyle.Render("📋 Engineering Quality Standards Check:"))
	content = append(content, m.renderStandardsGrid())
	if len(m.validatorLogs) > 0 {
		content = append(content, "\n"+TitleStyle.Render("💻 Validator Live Console Logs:"))
		boxContent := strings.Join(m.validatorLogs, "\n")
		styledLogs := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a1a1aa")).
			Background(lipgloss.Color("#18181b")).
			Padding(1, 2).
			Width(m.width - 45).
			Render(boxContent)
		content = append(content, styledLogs)
	}
	return strings.Join(content, "\n")
}

func (m DashboardModel) renderCompletedState() string {
	var content []string
	content = append(content, TitleStyle.Render("🎉 Specification Complete!"))
	if m.genPhase == "parallel" {
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("Last phase: Parallel downstream generation"))
	}
	content = append(content, "\nAll requirement vectors have achieved 100% confidence and files have been generated.")
	content = append(content, "You can still edit raw facts to regenerate, or quit:\n")
	content = append(content, lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("  Press [G] to manually Regenerate files"))
	content = append(content, lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("  Press [U] to Add new requirements / Modify specifications"))
	content = append(content, lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("  Press [E] to launch Editor & make modifications"))
	content = append(content, lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("  Press [Q] to Save & Exit CLI"))
	content = append(content, "\n"+TitleStyle.Render("📂 Document Synthesis Status:"))
	content = append(content, m.renderFileProgressList())
	content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorMuted).Render(m.genStatus))
	if m.showScorecard {
		content = append(content, "\n"+TitleStyle.Render("📊 Final Architectural Quality Scorecard:"))
		content = append(content, m.renderStandardsGrid())
	}
	return strings.Join(content, "\n")
}

func (m DashboardModel) renderInterrogationState() string {
	var content []string
	content = append(content, TitleStyle.Render("💬 Conversation Timeline"))

	if m.Session.LastQuestion != "" && !m.loading && !m.isStreaming {
		content = append(content, "\n"+QuestionStyle.Render("Architect's Question:"))
		content = append(content, wrapText(m.Session.LastQuestion, m.width-45))
	}

	if m.loading || m.isStreaming {
		if m.loading {
			content = append(content, fmt.Sprintf("\n%s Architectural Reasoning in progress. Calling AI API...", m.spinner.View()))
		} else {
			content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("✓ Response received — streaming thought tokens..."))
		}
		content = append(content, "\n"+m.renderThoughtBox())
	} else if m.showTextInput {
		content = append(content, "\n"+m.textInput.View())
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorMuted).Render("(Press Esc to return to choices)"))
	} else {
		choices := m.getChoicesList()
		content = append(content, "\nSelect an option:")
		for i, choice := range choices {
			if i == m.selectedChoiceIdx {
				style := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
				content = append(content, style.Render(fmt.Sprintf("  ❯ %s", choice)))
			} else {
				content = append(content, fmt.Sprintf("    %s", choice))
			}
		}
	}
	return strings.Join(content, "\n")
}

func (m DashboardModel) renderThoughtBox() string {
	boxWidth := m.width - 45
	if boxWidth < 40 {
		boxWidth = 40
	}

	title := ThoughtTitleStyle.Render("💭 Streaming Thought Box (Reasoning Tokens)")

	tokens := m.streamingTokens
	if tokens == "" {
		tokens = "Awaiting first token chunk..."
	}

	wrappedTokens := wrapText(tokens, boxWidth-4)

	lines := strings.Split(wrappedTokens, "\n")
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	bodyContent := strings.Join(lines, "\n")

	return ThoughtBoxStyle.Width(boxWidth).Render(title + "\n\n" + bodyContent)
}

func (m DashboardModel) renderMainChat() string {
	if m.showViewer {
		return m.renderViewer()
	}
	if m.showUpdatePrompt {
		var content []string
		content = append(content, TitleStyle.Render("📝 Add New Requirement / Modify Specification"))
		content = append(content, "\nEnter your new requirements or modifications below:")
		content = append(content, "\n"+m.updateInput.View())
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorMuted).Render("(Press Enter to submit, Esc to cancel)"))
		return strings.Join(content, "\n")
	}
	if m.isGenerating {
		return m.renderGeneratingState()
	}
	if m.isCompleted {
		return m.renderCompletedState()
	}
	return m.renderInterrogationState()
}

// renderViewer draws the scrollable Markdown document viewer with a border, header, and footer.
func (m DashboardModel) renderViewer() string {
	if len(m.genFiles) == 0 {
		return "No files generated yet."
	}
	selectedFile := m.genFiles[m.selectedFileIdx]

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorMuted).
		Padding(0, 1)

	modeStr := "Split-Pane"
	if m.isFullScreenViewer {
		modeStr = "Full-Screen"
	}
	header := headerStyle.Render(fmt.Sprintf("📖 Viewing: %s (%s)", selectedFile, modeStr))

	footerStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(ColorMuted).
		Padding(0, 1)

	scrollPercent := fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100)
	if m.viewport.AtBottom() {
		scrollPercent = "End"
	} else if m.viewport.AtTop() {
		scrollPercent = "Top"
	}

	footerText := fmt.Sprintf("Progress: %s  |  [Esc / q] Back  |  [f] Toggle Layout  |  [j / k] Scroll", scrollPercent)
	if m.isWaitingApproval {
		footerText = fmt.Sprintf("Progress: %s  |  [A / Enter] Approve & Resume  |  [E] Edit  |  [Esc / q] Back  |  [f] Toggle Layout  |  [j / k] Scroll", scrollPercent)
	}
	footer := footerStyle.Render(footerText)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n"+m.viewport.View(),
		footer,
	)
}

func (m DashboardModel) renderFooter() string {
	var elements []string

	// Error Display
	if m.err != nil {
		var errTitle string
		var errBody string

		if apiErr, ok := m.err.(*gateway.APIError); ok {
			errTitle = fmt.Sprintf("API REQUEST FAILED (%d)", apiErr.StatusCode)
			errBody = apiErr.Message
			if apiErr.RetryAfter != "" {
				errBody += fmt.Sprintf("\nSuggested retry delay: %s", apiErr.RetryAfter)
			}
		} else {
			errTitle = "SYSTEM ERROR"
			errBody = m.err.Error()
		}

		// Wrap the error body to the dashboard width to prevent stretching the layout
		wrappedBody := wrapText(errBody, m.width-8)
		styledTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true).Render("⚠️  " + errTitle)
		boxContent := fmt.Sprintf("%s\n%s", styledTitle, wrappedBody)

		styledBox := ErrorBoxStyle.Width(m.width - 6).Render(boxContent)
		elements = append(elements, styledBox)
	}

	// Keybindings helper
	keys := []string{"Ctrl+C: Quit"}
	if m.err != nil {
		keys = append(keys, "Esc: Dismiss Error")
	} else if m.isCompleted {
		keys = append(keys, "v/Enter: View file", "g: Regenerate", "u: Modify", "e: Editor")
	} else if m.showTextInput {
		keys = append(keys, "Enter: Send", "Ctrl+K: I Don't Know", ":edit: Open full editor", "Esc: Cancel")
	} else {
		keys = append(keys, "j/k/Arrows: Navigate", "Enter: Select", "Ctrl+K: I Don't Know", "g: Regenerate", "u: Modify", "e: Editor")
	}
	elements = append(elements, strings.Join(keys, "  |  "))

	return FooterStyle.Render(strings.Join(elements, "\n"))
}

func wrapSingleLine(line string, width int) []string {
	if len(line) <= width {
		return []string{line}
	}

	indent := ""
	for _, c := range line {
		if c == ' ' || c == '\t' {
			indent += string(c)
		} else {
			break
		}
	}

	trimmed := strings.TrimSpace(line)
	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return []string{indent}
	}

	var result []string
	currentLine := indent + words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > width {
			result = append(result, currentLine)
			currentLine = indent + word
		} else {
			currentLine += " " + word
		}
	}
	result = append(result, currentLine)
	return result
}

// Simple text wrapping helper
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, wrapSingleLine(line, width)...)
	}
	return strings.Join(result, "\n")
}

var (
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleInfo    = lipgloss.NewStyle().Foreground(ColorInfo)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")) // Vibrant Red
)

func (m DashboardModel) getStandardScorecardStatus(std config.Standard) (string, lipgloss.Style) {
	score, found := m.complianceScores[std.ID]
	if !found {
		return "🔴 N/A", StyleError
	}
	if score >= std.MinScore {
		return fmt.Sprintf("🟢 %d%%", score), StyleSuccess
	} else if score > 0 {
		return fmt.Sprintf("🟡 %d%%", score), StyleWarning
	}
	return fmt.Sprintf("🔴 %d%%", score), StyleError
}

func isStandardFileInPast(std config.Standard, currentFileIdx int, files []string) bool {
	for _, tf := range std.TargetFiles {
		stdFileIdx := -1
		for i, f := range files {
			if f == tf {
				stdFileIdx = i
				break
			}
		}
		if stdFileIdx >= currentFileIdx {
			return false
		}
	}
	return true
}

func checkActiveStandardStatus(std config.Standard, status string) (string, lipgloss.Style, bool) {
	for _, tf := range std.TargetFiles {
		if strings.Contains(status, tf) {
			if strings.Contains(status, "Auditing") || strings.Contains(status, "Refining") || strings.Contains(status, "failed") {
				return "🔄 Auditing", StyleInfo, true
			}
			return "⏳ Building", StyleMuted, true
		}
	}
	return "", lipgloss.Style{}, false
}

func (m DashboardModel) getStandardStatus(std config.Standard) (string, lipgloss.Style) {
	if m.showScorecard {
		return m.getStandardScorecardStatus(std)
	}

	if !m.isGenerating {
		return pendingStr, StyleMuted
	}

	if statusText, style, active := checkActiveStandardStatus(std, m.genStatus); active {
		return statusText, style
	}

	files := []string{
		"01_domain_model_use_cases.md",
		"02_prd_functional.md",
		"03_system_architecture.md",
		"04_api_architecture_integration.md",
		"05_coding_standards_guidelines.md",
		"06_security_threat_model.md",
		"07_engineering_roadmap.md",
	}

	currentFileIdx := -1
	for i, f := range files {
		if strings.Contains(m.genStatus, f) {
			currentFileIdx = i
			break
		}
	}

	if currentFileIdx == -1 {
		if strings.Contains(m.genStatus, "successfully") || strings.Contains(m.genStatus, "Compiling") || strings.Contains(m.genStatus, "audited") {
			return "🟢 Verified", StyleSuccess
		}
		return pendingStr, StyleMuted
	}

	if isStandardFileInPast(std, currentFileIdx, files) {
		return "🟢 Verified", StyleSuccess
	}

	return pendingStr, StyleMuted
}

func (m DashboardModel) renderStandardsGrid() string {
	var leftCol []string
	var rightCol []string

	half := (len(m.standards) + 1) / 2
	for i, std := range m.standards {
		statusText, style := m.getStandardStatus(std)

		styledLabel := lipgloss.NewStyle().Foreground(ColorText).Render(std.Name)
		styledStatus := style.Bold(true).Render(statusText)

		padding := 28 - len(std.Name)
		if padding < 1 {
			padding = 1
		}
		item := fmt.Sprintf("  %s:%s%s", styledLabel, strings.Repeat(" ", padding), styledStatus)

		if i < half {
			leftCol = append(leftCol, item)
		} else {
			rightCol = append(rightCol, item)
		}
	}

	leftBlock := strings.Join(leftCol, "\n")
	rightBlock := strings.Join(rightCol, "\n")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftBlock,
		"       ", // spacer
		rightBlock,
	)
}

// getFileStatusIconAndStyle mapping maps dynamic status to its TUI icon and color style.
func (m DashboardModel) getFileStatusIconAndStyle(status string) (string, lipgloss.Style) {
	switch status {
	case "skipped", "done":
		return "🟢 Done", StyleSuccess
	case "waiting_approval":
		return "⏸️ Awaiting Approval", StyleWarning
	case "synthesizing":
		return "🔄 Synthesizing", StyleInfo
	case "correcting":
		return "⚠️ Correcting", StyleWarning
	case "auditing":
		return "🔍 Auditing", StyleInfo
	case "refining":
		return "🛠️ Refining", StyleWarning
	case "failed":
		return "🔴 Failed", StyleError
	default:
		return pendingStr, StyleMuted
	}
}

// renderFileProgressList draws the complete layout list of generated files with indicators.
func (m DashboardModel) renderFileProgressList() string {
	var sourceLines []string
	var downstreamLines []string

	for idx, file := range m.genFiles {
		status := m.genFileStatuses[file]
		details := m.genFileDetails[file]

		icon, style := m.getFileStatusIconAndStyle(status)
		styledIcon := style.Bold(true).Render(icon)

		var styledFile string
		prefix := "  "
		if m.isCompleted && idx == m.selectedFileIdx {
			prefix = "❯ "
			styledFile = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(file)
		} else {
			styledFile = lipgloss.NewStyle().Foreground(ColorText).Bold(true).Render(file)
		}

		var line string
		if details != "" && details != "completed successfully" && details != "already generated" {
			styledDetails := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("(%s)", details))
			line = fmt.Sprintf("%s%s %s %s", prefix, styledIcon, styledFile, styledDetails)
		} else {
			line = fmt.Sprintf("%s%s %s", prefix, styledIcon, styledFile)
		}

		if file == "01_domain_model_use_cases.md" {
			sourceLines = append(sourceLines, line)
		} else {
			downstreamLines = append(downstreamLines, line)
		}
	}

	var sections []string
	if len(sourceLines) > 0 {
		sections = append(sections, lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("Source"))
		sections = append(sections, sourceLines...)
	}
	if len(downstreamLines) > 0 {
		sections = append(sections, lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("Parallel downstream"))
		sections = append(sections, renderParallelProgressGrid(downstreamLines))
	}

	return strings.Join(sections, "\n")
}

func renderParallelProgressGrid(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	half := (len(lines) + 1) / 2
	leftLines := lines[:half]
	rightLines := lines[half:]

	leftBlock := strings.Join(leftLines, "\n")
	rightBlock := strings.Join(rightLines, "\n")

	if rightBlock == "" {
		return leftBlock
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftBlock,
		"    ",
		rightBlock,
	)
}
