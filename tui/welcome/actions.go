package welcome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/generator/export"
	"github.com/toanle/synthspec/state"
)

func (m WelcomeModel) handleProjectMenuSelection() (tea.Model, tea.Cmd) {
	switch m.SelectedProjectOption {
	case 0:
		if m.IsNewProject {
			m.Action = ActionCreate
		} else {
			m.Action = ActionResume
		}
		return m, tea.Quit

	case 1:
		return m.handleProjectMenuSelectionViewFiles()

	case 2:
		return m.handleProjectMenuSelectionExport()

	case 3:
		m.Phase = PhaseDeleteConfirm
		return m, nil

	case 4:
		m.Phase = PhaseMenu
		return m, nil
	}
	return m, nil
}

func (m WelcomeModel) handleProjectMenuSelectionViewFiles() (tea.Model, tea.Cmd) {
	if m.IsNewProject {
		m.alertTitle = noFilesLiteral
		m.alertMessage = "This is a new project. No specifications have been generated yet."
		m.alertNext = PhaseProjectMenu
		m.Phase = PhaseStatusAlert
		return m, nil
	}
	outDir := filepath.Join(state.GetSessionDir(m.ProjectName), "output")
	files, err := os.ReadDir(outDir)
	if err != nil || len(files) == 0 {
		m.alertTitle = noFilesLiteral
		m.alertMessage = "No generated markdown specifications were found for this project."
		m.alertNext = PhaseProjectMenu
		m.Phase = PhaseStatusAlert
		return m, nil
	}

	var mdFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".md") {
			mdFiles = append(mdFiles, f.Name())
		}
	}
	if len(mdFiles) == 0 {
		m.alertTitle = noFilesLiteral
		m.alertMessage = "No generated markdown specifications were found for this project."
		m.alertNext = PhaseProjectMenu
		m.Phase = PhaseStatusAlert
		return m, nil
	}
	m.ProjectFiles = mdFiles
	m.SelectedProjectFile = 0
	m.Phase = PhaseProjectViewFiles
	return m, nil
}

func (m WelcomeModel) handleProjectMenuSelectionExport() (tea.Model, tea.Cmd) {
	if m.IsNewProject {
		m.alertTitle = "No Specifications"
		m.alertMessage = "This is a new project. Start specification generation before exporting."
		m.alertNext = PhaseProjectMenu
		m.Phase = PhaseStatusAlert
		return m, nil
	}
	projDir := state.GetSessionDir(m.ProjectName)
	outputDir := filepath.Join(projDir, "output")
	distDir := filepath.Join(projDir, "dist")

	indexPath, err := export.ExportToHTML(m.ProjectName, outputDir, distDir)
	if err != nil {
		m.alertTitle = "Export Failed"
		m.alertMessage = fmt.Sprintf("Failed to export: %v", err)
	} else {
		m.alertTitle = "Export Successful"
		m.alertMessage = fmt.Sprintf("HTML exported successfully to:\n%s", indexPath)
	}
	m.alertNext = PhaseProjectMenu
	m.Phase = PhaseStatusAlert
	return m, nil
}

func (m *WelcomeModel) handleMenuSelection() tea.Cmd {
	switch m.SelectedOption {
	case 0:
		m.textInput.SetValue("")
		m.Phase = PhaseCreateInput
		return m.textInput.Focus()
	case 1:
		projects, err := state.ListProjects()
		if err != nil {
			m.alertTitle = "Error Scanning Projects"
			m.alertMessage = fmt.Sprintf("Failed to list existing projects: %v", err)
			m.Phase = PhaseStatusAlert
			return nil
		}
		if len(projects) == 0 {
			m.alertTitle = "No Saved Projects"
			m.alertMessage = "No active SynthSpec projects were found in this directory.\nChoose 'Create New Project' to get started."
			m.Phase = PhaseStatusAlert
			return nil
		}
		m.Projects = projects
		m.FilteredProjects = projects
		m.SelectedProject = 0
		m.filterInput.SetValue("")
		m.Phase = PhaseResumeSelect
		return m.filterInput.Focus()
	case 2:
		projects, err := state.ListProjects()
		if err != nil {
			m.alertTitle = "Error Scanning Projects"
			m.alertMessage = fmt.Sprintf("Failed to list existing projects: %v", err)
			m.Phase = PhaseStatusAlert
			return nil
		}
		if len(projects) == 0 {
			m.alertTitle = "No Saved Projects"
			m.alertMessage = "No active SynthSpec projects were found to export.\nChoose 'Create New Project' to get started."
			m.Phase = PhaseStatusAlert
			return nil
		}
		m.Projects = projects
		m.FilteredProjects = projects
		m.SelectedProject = 0
		m.filterInput.SetValue("")
		m.Phase = PhaseExportSelect
		return m.filterInput.Focus()
	case 3:
		m.Phase = PhaseViewAssets
	case 4:
		m.Phase = PhaseAuditWorkspace
	case 5:
		m.settingInputs[0].SetValue(fmt.Sprintf("%d", m.Settings.TimeoutSeconds))
		m.settingInputs[1].SetValue(fmt.Sprintf("%d", m.Settings.MaxRetries))
		m.settingInputs[2].SetValue(m.Settings.DefaultOutputFolder)
		m.settingInputs[3].SetValue(fmt.Sprintf("%.2f", m.Settings.HardBudgetCap))
		m.SelectedSettingIdx = 0
		cmd := m.settingInputs[0].Focus()
		m.settingInputs[1].Blur()
		m.settingInputs[2].Blur()
		m.Phase = PhaseSettings
		return cmd
	case 6:
		m.Action = ActionExit
		m.Phase = PhaseStatusAlert
	}
	return nil
}

func PhaseThemeToggle() WelcomePhase {
	return PhaseAuditWorkspace
}
