package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/tui/shared"
)

const (
	pendingStr  = "⏳ Pending"
	verifiedStr = "🟢 Verified"
)

var synthesisFiles = []string{
	domainModelFilename,
	"02_prd_functional.md",
	"03_system_architecture.md",
	"04_api_architecture_integration.md",
	"05_coding_standards_guidelines.md",
	"06_security_threat_model.md",
	"07_engineering_roadmap.md",
}

func (m DashboardModel) getStandardScorecardStatus(std config.Standard) (string, lipgloss.Style) {
	score, found := m.complianceScores[std.ID]
	if !found {
		return "🔴 N/A", shared.StyleError
	}
	if score >= std.MinScore {
		return fmt.Sprintf("🟢 %d%%", score), shared.StyleSuccess
	} else if score > 0 {
		return fmt.Sprintf("🟡 %d%%", score), shared.StyleWarning
	}
	return fmt.Sprintf("🔴 %d%%", score), shared.StyleError
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
				return "🔄 Auditing", shared.StyleInfo, true
			}
			return "⏳ Building", shared.StyleMuted, true
		}
	}
	return "", lipgloss.Style{}, false
}

func (m DashboardModel) areAllTargetFilesDone(std config.Standard) bool {
	if len(std.TargetFiles) == 0 {
		return false
	}
	for _, tf := range std.TargetFiles {
		status := m.genFileStatuses[tf]
		if status != "done" && status != "skipped" {
			return false
		}
	}
	return true
}

func (m DashboardModel) getActiveGenerationStandardStatus(std config.Standard) (string, lipgloss.Style) {
	if statusText, style, active := checkActiveStandardStatus(std, m.genStatus); active {
		return statusText, style
	}

	if m.areAllTargetFilesDone(std) {
		return verifiedStr, shared.StyleSuccess
	}

	currentFileIdx := -1
	for i, f := range synthesisFiles {
		if strings.Contains(m.genStatus, f) {
			currentFileIdx = i
			break
		}
	}

	if currentFileIdx == -1 {
		if strings.Contains(m.genStatus, "successfully") || strings.Contains(m.genStatus, "Compiling") || strings.Contains(m.genStatus, "audited") {
			return verifiedStr, shared.StyleSuccess
		}
		return pendingStr, shared.StyleMuted
	}

	if isStandardFileInPast(std, currentFileIdx, synthesisFiles) {
		return verifiedStr, shared.StyleSuccess
	}

	return pendingStr, shared.StyleMuted
}

func (m DashboardModel) getStandardStatus(std config.Standard) (string, lipgloss.Style) {
	if m.showScorecard {
		return m.getStandardScorecardStatus(std)
	}

	if !m.isGenerating {
		return pendingStr, shared.StyleMuted
	}

	return m.getActiveGenerationStandardStatus(std)
}

func (m DashboardModel) groupStandardsByFile() (map[string][]config.Standard, []config.Standard) {
	standardsByFile := make(map[string][]config.Standard)
	var unmapped []config.Standard
	for _, std := range m.standards {
		mapped := false
		for _, tf := range std.TargetFiles {
			for _, f := range synthesisFiles {
				if f == tf {
					standardsByFile[f] = append(standardsByFile[f], std)
					mapped = true
					break
				}
			}
			if mapped {
				break
			}
		}
		if !mapped {
			unmapped = append(unmapped, std)
		}
	}
	return standardsByFile, unmapped
}

func (m DashboardModel) getActiveFile() string {
	if m.isCompleted {
		return ""
	}
	for _, f := range synthesisFiles {
		status := m.genFileStatuses[f]
		if status != "done" && status != "skipped" {
			return f
		}
	}
	return ""
}

func (m DashboardModel) getGroupHeaderIcon(fileStatus string) string {
	switch fileStatus {
	case "done", "skipped":
		return "🟢"
	case "pending", "":
		return "⏳"
	default:
		return "🔄"
	}
}

func (m DashboardModel) countVerifiedStandards(standards []config.Standard) int {
	verifiedCount := 0
	for _, std := range standards {
		statusText, _ := m.getStandardStatus(std)
		if strings.Contains(statusText, "Verified") || (strings.Contains(statusText, "%") && !strings.Contains(statusText, "0%")) {
			verifiedCount++
		}
	}
	return verifiedCount
}

func (m DashboardModel) renderGroupDetails(fileStandards []config.Standard) string {
	var leftCol []string
	var rightCol []string
	half := (len(fileStandards) + 1) / 2

	for i, std := range fileStandards {
		statusText, style := m.getStandardStatus(std)
		styledLabel := lipgloss.NewStyle().Foreground(shared.ColorText).Render(std.Name)
		styledStatus := style.Bold(true).Render(statusText)

		padding := 32 - len(std.Name)
		if padding < 1 {
			padding = 1
		}
		item := fmt.Sprintf("    %s:%s%s", styledLabel, strings.Repeat(" ", padding), styledStatus)

		if i < half {
			leftCol = append(leftCol, item)
		} else {
			rightCol = append(rightCol, item)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		strings.Join(leftCol, "\n"),
		"       ", // spacer
		strings.Join(rightCol, "\n"),
	)
}

func (m DashboardModel) renderStandardGroup(fileHeader string, fileStandards []config.Standard, isExpanded bool, fileStatus string) string {
	headerIcon := m.getGroupHeaderIcon(fileStatus)
	verifiedCount := m.countVerifiedStandards(fileStandards)

	var fileLabel string
	if m.isCompleted {
		fileLabel = fmt.Sprintf("  %s %s", headerIcon, fileHeader)
	} else if isExpanded {
		fileLabel = fmt.Sprintf("  ▼ %s %s (Active)", headerIcon, fileHeader)
	} else if fileStatus == "done" || fileStatus == "skipped" {
		fileLabel = fmt.Sprintf("  ▶ %s %s (%d/%d Checked)", headerIcon, fileHeader, verifiedCount, len(fileStandards))
	} else {
		fileLabel = fmt.Sprintf("  ▶ %s %s (%d Pending)", headerIcon, fileHeader, len(fileStandards))
	}

	headerLine := lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render(fileLabel)
	if !isExpanded && !m.isCompleted {
		return headerLine
	}

	return headerLine + "\n" + m.renderGroupDetails(fileStandards)
}

func (m DashboardModel) renderStandardsGrid() string {
	standardsByFile, unmapped := m.groupStandardsByFile()
	activeFile := m.getActiveFile()

	var sections []string
	for _, f := range synthesisFiles {
		fileStandards := standardsByFile[f]
		if len(fileStandards) == 0 {
			continue
		}
		status := m.genFileStatuses[f]
		isExpanded := (f == activeFile)
		sections = append(sections, m.renderStandardGroup(f, fileStandards, isExpanded, status))
	}

	if len(unmapped) > 0 {
		sections = append(sections, m.renderStandardGroup("General / Unmapped Standards", unmapped, !m.isCompleted, "active"))
	}

	return strings.Join(sections, "\n\n")
}
