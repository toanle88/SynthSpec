package welcome

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/tui/shared"
)

const selectionFormat = " %s %s"

func (m WelcomeModel) View() string {
	if m.Action == ActionExit && m.Phase == PhaseStatusAlert {
		return "Exiting SynthSpec. Goodbye!\n"
	}

	logo := `
   _____             __  __   _____                     
  / ____|           |  \/  | / ____|                    
 | (___   _   _  _  | \  / || (___   _ __    ___   ___  
  \___ \ | | | || |_| |\/| | \___ \ | '_ \  / _ \ / __| 
  ____) || |_| ||  _| |  | | ____) || |_) ||  __/| (__  
 |_____/  \__, ||_| |_|  |_||_____/ | .__/  \___| \___| 
           __/ |                    | |                 
          |___/                     |_|                 
`
	logoStyle := lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
	logoText := logoStyle.Render(logo)
	subTitle := lipgloss.NewStyle().Foreground(shared.ColorMuted).Italic(true).Render("  Open-Source BYOK AI Solution Architect CLI")

	var content string
	switch m.Phase {
	case PhaseMenu:
		content = m.viewMenu()
	case PhaseCreateInput:
		content = m.viewCreateInput()
	case PhaseBlueprintSelect:
		content = m.viewBlueprintSelect()
	case PhaseResumeSelect:
		content = m.viewResumeSelect()
	case PhaseExportSelect:
		content = m.viewExportSelect()
	case PhaseStatusAlert:
		content = m.viewStatusAlert()
	case PhaseSettings:
		content = m.viewSettings()
	case PhaseViewAssets:
		content = m.viewViewAssets()
	case PhaseAuditWorkspace:
		content = m.viewAuditWorkspace()
	case PhaseProjectMenu:
		content = m.viewProjectMenu()
	case PhaseProjectViewFiles:
		content = m.viewProjectViewFiles()
	case PhaseFileContentViewer:
		content = m.viewFileContentViewer()
	case PhaseDeleteConfirm:
		content = m.viewDeleteConfirm()
	}

	body := lipgloss.JoinVertical(lipgloss.Left, logoText, subTitle, content)
	h := 18
	w := 65

	centered := lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Top,
		body,
	)

	return shared.DocStyle.Render(centered)
}
