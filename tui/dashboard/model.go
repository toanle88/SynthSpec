package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui/shared"
)

const (
	manualUpdateMsg     = "Requirements updated manually via editor."
	domainModelFilename = "01_domain_model_use_cases.md"
)

// Msg Types
type oracleResultMsg struct {
	resp *gateway.OracleResponse
	err  error
}

type editorFinishedMsg struct {
	err error
}

type fileEditorFinishedMsg struct {
	err error
}

type genProgressMsg string
type genFinishedMsg struct {
	err error
}

type contextPruneResultMsg struct {
	pruned bool
	err    error
}

type thoughtTokenMsg string
type streamDoneMsg struct{}
type initQueryMsg struct{}
type typingTickMsg struct{}

// DashboardModel represents the TUI state
type DashboardModel struct {
	Session   state.SessionManager
	Gateway   gateway.Gateway
	OutputDir string
	Settings  *config.Settings

	textInput textinput.Model
	spinner   spinner.Model
	loading   bool
	err       error

	// Layout sizes
	width  int
	height int

	// Editor state
	editorTempPath string

	// Embedded sub-states
	GenerationState
	ThoughtStreamState
	ComplianceState
	ViewerState
	ApprovalGateState

	// Main Model fields
	selectedChoiceIdx int
	showTextInput     bool
	validatorLogs     []string

	showUpdatePrompt bool
	updateInput      textinput.Model
	isCLIUpdateMode  bool
	chatViewport     viewport.Model
}

type GenerationState struct {
	isCompleted     bool
	isGenerating    bool
	genStatus       string
	genPhase        string
	genChan         chan string
	genFiles        []string
	genFileStatuses map[string]string
	genFileDetails  map[string]string
	cancelGen       context.CancelFunc
	forceFinishChan chan struct{}
}

type ThoughtStreamState struct {
	thoughtChan     chan string
	streamingTokens string
	thoughtBuffer   string
	isStreaming     bool
	isTyping        bool
}

type ComplianceState struct {
	standards        []config.Standard
	complianceScores map[string]int
	showScorecard    bool
}

type ViewerState struct {
	viewport           viewport.Model
	showViewer         bool
	selectedFileIdx    int
	isFullScreenViewer bool
}

type ApprovalGateState struct {
	approvalChan          chan struct{}
	diffApprovalChan      chan struct{}
	isWaitingApproval     bool
	isWaitingDiffApproval bool
	isEditingFile         bool
	proposedDiffs         []domain.FileDiff
	selectedDiffIdx       int
	showDiffViewer        bool
}

func NewDashboardModel(sess state.SessionManager, gw gateway.Gateway, outputDir string) DashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = shared.SpinnerStyle

	ti := textinput.New()
	ti.Placeholder = "Type your answer here, or ':edit' to open in full editor..."
	ti.Prompt = "> "
	ti.PromptStyle = shared.InputPrefixStyle
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 60

	// Check if already 100% completed
	completed := checkCompletion(sess.GetScores())

	standards, err := config.LoadStandards()
	if err != nil {
		logger.Log("WARN: failed to load standards: %v", err)
	}

	resolvedOutputDir := outputDir
	if resolvedOutputDir == "" {
		resolvedOutputDir = filepath.Join(state.GetSessionDir(sess.GetProjectName()), "output")
	}

	complianceScores, showScorecard := initializeComplianceScores(sess, completed, resolvedOutputDir)
	showTextInput := len(sess.GetLastChoices()) == 0

	templates, err := config.LoadTemplates()
	if err != nil {
		logger.Log("WARN: failed to load templates: %v", err)
	}
	var genFiles []string
	for _, t := range templates {
		genFiles = append(genFiles, t.FileName)
	}
	genFileStatuses := initializeFileStatuses(genFiles, resolvedOutputDir)
	genFileDetails := make(map[string]string)

	ui := textinput.New()
	ui.Placeholder = "Type new requirements or modifications here..."
	ui.Prompt = "> "
	ui.PromptStyle = shared.InputPrefixStyle
	ui.CharLimit = 2000
	ui.Width = 60

	settings := initializeSettings()

	return DashboardModel{
		Session:     sess,
		Gateway:     gw,
		OutputDir:   outputDir,
		Settings:    settings,
		textInput:   ti,
		updateInput: ui,
		spinner:     s,
		GenerationState: GenerationState{
			isCompleted:     completed,
			genFiles:        genFiles,
			genFileStatuses: genFileStatuses,
			genFileDetails:  genFileDetails,
		},
		ThoughtStreamState: ThoughtStreamState{
			thoughtChan:     make(chan string, 100),
			streamingTokens: "",
			thoughtBuffer:   "",
			isStreaming:     false,
			isTyping:        false,
		},
		ComplianceState: ComplianceState{
			standards:        standards,
			complianceScores: complianceScores,
			showScorecard:    showScorecard,
		},
		ViewerState: ViewerState{
			viewport:           viewport.Model{},
			showViewer:         false,
			selectedFileIdx:    0,
			isFullScreenViewer: false,
		},
		ApprovalGateState: ApprovalGateState{
			approvalChan:          nil,
			diffApprovalChan:      nil,
			isWaitingApproval:     false,
			isWaitingDiffApproval: false,
			isEditingFile:         false,
			showDiffViewer:        false,
		},
		selectedChoiceIdx: 0,
		showTextInput:     showTextInput,
		loading:           !completed && len(sess.GetHistory()) == 0 && sess.GetLastQuestion() == "",
		chatViewport:      viewport.New(0, 0),
	}
}

func (m DashboardModel) Init() tea.Cmd {
	logger.LogEvent("TUI", fmt.Sprintf("Dashboard initialized for project: %s", m.Session.GetProjectName()))
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)

	// Bootstrapping: fire initQueryMsg so that Update() handles it via startOracleQuery,
	// which correctly sets isStreaming and thoughtChan on the REAL Bubble Tea model.
	// We cannot mutate those fields here because Init() has a value receiver —
	// any changes made to m inside Init() are discarded by the runtime.
	if len(m.Session.GetHistory()) == 0 && m.Session.GetLastQuestion() == "" {
		cmds = append(cmds, func() tea.Msg { return initQueryMsg{} })
	}

	return tea.Batch(cmds...)
}

func checkCompletion(scores gateway.ConfidenceScores) bool {
	return scores.Functional >= 100 &&
		scores.Structural >= 100 &&
		scores.Security >= 100 &&
		scores.Compliance >= 100
}

// setError assigns the current model error and formats runtime diagnostic messages to errors.log.
func (m *DashboardModel) setError(err error) {
	m.err = err
	if err != nil {
		var projectName string
		if m.Session != nil {
			projectName = m.Session.GetProjectName()
		}
		logger.LogError(projectName, "tui", "setError", err)
	}
}

func (m *DashboardModel) StartWithUpdatePrompt() {
	m.showUpdatePrompt = true
	m.isCLIUpdateMode = true
	m.updateInput.Focus()
}

func initializeComplianceScores(sess state.SessionManager, completed bool, resolvedOutputDir string) (map[string]int, bool) {
	complianceScores := make(map[string]int)
	showScorecard := false
	if completed {
		metaPath := filepath.Join(resolvedOutputDir, ".synthspec-meta.json")
		if metaBytes, readErr := os.ReadFile(metaPath); readErr == nil {
			var meta struct {
				ComplianceSummary map[string]int `json:"compliance_summary"`
			}
			if jsonErr := json.Unmarshal(metaBytes, &meta); jsonErr == nil && len(meta.ComplianceSummary) > 0 {
				complianceScores = meta.ComplianceSummary
				showScorecard = true
			}
		}
	}
	return complianceScores, showScorecard
}

func initializeFileStatuses(genFiles []string, resolvedOutputDir string) map[string]string {
	genFileStatuses := make(map[string]string)
	for _, f := range genFiles {
		filePath := filepath.Join(resolvedOutputDir, f)
		if _, err := os.Stat(filePath); err == nil {
			genFileStatuses[f] = "done"
		} else {
			genFileStatuses[f] = "pending"
		}
	}
	return genFileStatuses
}

func initializeSettings() *config.Settings {
	settings, err := config.LoadSettings()
	if err != nil {
		logger.Log("WARN: failed to load settings: %v", err)
	}
	if settings == nil {
		settings = &config.Settings{
			TimeoutSeconds:      config.DefaultTimeoutSeconds,
			MaxRetries:          config.DefaultMaxRetries,
			DefaultOutputFolder: config.DefaultOutputFolderValue,
			Debug:               false,
			VimMode:             false,
		}
	}
	return settings
}





