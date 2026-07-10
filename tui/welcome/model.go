package welcome

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/logger"
)

type WelcomePhase int

const (
	PhaseMenu WelcomePhase = iota
	PhaseCreateInput
	PhaseBlueprintSelect
	PhaseResumeSelect
	PhaseStatusAlert
	PhaseSettings
	PhaseViewAssets
	PhaseAuditWorkspace
	PhaseExportSelect
	PhaseProjectMenu
	PhaseProjectViewFiles
	PhaseFileContentViewer
	PhaseDeleteConfirm
)

type WelcomeAction int

const (
	ActionNone WelcomeAction = iota
	ActionCreate
	ActionResume
	ActionExport
	ActionExit
)

const (
	keyCtrlP       = "ctrl+p"
	keyCtrlN       = "ctrl+n"
	keyShiftTab    = "shift+tab"
	cancelLiteral  = "[ Cancel ]"
	noFilesLiteral = "No Files"
)

type WelcomeModel struct {
	Phase          WelcomePhase
	Action         WelcomeAction
	ProjectName    string
	SelectedOption int
	Options        []string

	// Project Resume/Menu Selection
	Projects              []string
	FilteredProjects      []string
	SelectedProject       int
	filterInput           textinput.Model
	IsNewProject          bool
	ProjectOptions        []string
	SelectedProjectOption int

	// Project File Viewer
	ProjectFiles        []string
	SelectedProjectFile int
	ViewerLines         []string
	ViewerScrollOffset  int

	// TextInput for name
	textInput textinput.Model

	// Blueprint Selection
	Blueprints        []config.Blueprint
	SelectedBlueprint string
	SelectedBPIdx     int

	// Alerts
	alertTitle   string
	alertMessage string
	alertNext    WelcomePhase

	// Global Settings Phase Fields
	Settings           *config.Settings
	SelectedSettingIdx int
	settingInputs      []textinput.Model

	// Terminal dimensions
	width  int
	height int
}

func NewWelcomeModel() WelcomeModel {
	ti := textinput.New()
	ti.Placeholder = "Enter project name..."
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 30

	fi := textinput.New()
	fi.Placeholder = "Type to search..."
	fi.CharLimit = 64
	fi.Width = 30

	blueprints, err := config.LoadBlueprints()
	if err != nil {
		logger.Log("WARN: failed to load blueprints: %v", err)
	}
	loadSettings, err := config.LoadSettings()
	if err != nil {
		logger.Log("WARN: failed to load settings: %v", err)
	}
	settings := loadSettings
	if settings == nil {
		settings = &config.Settings{
			TimeoutSeconds:      config.DefaultTimeoutSeconds,
			MaxRetries:          config.DefaultMaxRetries,
			DefaultOutputFolder: config.DefaultOutputFolderValue,
		}
	}

	tInput := textinput.New()
	tInput.Placeholder = "60"
	tInput.CharLimit = 5
	tInput.Width = 10

	rInput := textinput.New()
	rInput.Placeholder = "3"
	rInput.CharLimit = 5
	rInput.Width = 10

	oInput := textinput.New()
	oInput.Placeholder = "./output"
	oInput.CharLimit = 256
	oInput.Width = 30

	cInput := textinput.New()
	cInput.Placeholder = "0.00"
	cInput.CharLimit = 10
	cInput.Width = 10

	return WelcomeModel{
		Phase:                 PhaseMenu,
		Action:                ActionNone,
		Options:               []string{"Create New Project", "Resume Existing Project", "Export to Static HTML", "View Assets", "Audit Workspace", "Settings", "Exit"},
		SelectedOption:        0,
		ProjectOptions:        []string{"Start/Resume Specification", "View Generated Files", "Export to Static HTML", "Delete Project", "Back to Main Menu"},
		SelectedProjectOption: 0,
		textInput:             ti,
		filterInput:           fi,
		Blueprints:            blueprints,
		Settings:              settings,
		settingInputs:         []textinput.Model{tInput, rInput, oInput, cInput},
	}
}

func (m WelcomeModel) Init() tea.Cmd {
	return textinput.Blink
}
