package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/generator"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/state"
)

const manualUpdateMsg = "Requirements updated manually via editor."

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

// DashboardModel represents the TUI state
type DashboardModel struct {
	Session   *state.Session
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

	// Generation state
	isCompleted     bool
	isGenerating    bool
	genStatus       string
	genPhase        string
	genChan         chan string
	genFiles        []string
	genFileStatuses map[string]string
	genFileDetails  map[string]string

	// Compliance scorecard state
	standards        []config.Standard
	complianceScores map[string]int
	showScorecard    bool

	// Choice selection state
	selectedChoiceIdx int
	showTextInput     bool

	// External validator logs
	validatorLogs []string

	// Update requirement state
	showUpdatePrompt bool
	updateInput      textinput.Model
	isCLIUpdateMode  bool
	viewport         viewport.Model
	showViewer       bool
	selectedFileIdx  int
	isFullScreenViewer bool

	// Thought stream state
	thoughtChan     chan string
	streamingTokens string

	// Approval Gate state
	approvalChan      chan struct{}
	isWaitingApproval bool
	isEditingFile     bool
}


func NewDashboardModel(sess *state.Session, gw gateway.Gateway, outputDir string) DashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	ti := textinput.New()
	ti.Placeholder = "Type your answer here, or ':edit' to open in full editor..."
	ti.Prompt = "> "
	ti.PromptStyle = InputPrefixStyle
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 60

	// Check if already 100% completed
	completed := checkCompletion(sess.Scores)

	standards, _ := config.LoadStandards()

	// If already completed and output meta exists, try to load scores
	complianceScores := make(map[string]int)
	showScorecard := false
	if completed {
		dir := outputDir
		if dir == "" {
			dir = filepath.Join(state.GetSessionDir(sess.ProjectName), "output")
		}
		metaPath := filepath.Join(dir, ".synthspec-meta.json")
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

	showTextInput := len(sess.LastChoices) == 0

	templates, _ := config.LoadTemplates()
	var genFiles []string
	for _, t := range templates {
		genFiles = append(genFiles, t.FileName)
	}
	genFileStatuses := make(map[string]string)
	genFileDetails := make(map[string]string)
	for _, f := range genFiles {
		genFileStatuses[f] = "pending"
	}

	ui := textinput.New()
	ui.Placeholder = "Type new requirements or modifications here..."
	ui.Prompt = "> "
	ui.PromptStyle = InputPrefixStyle
	ui.CharLimit = 2000
	ui.Width = 60

	settings, _ := config.LoadSettings()
	if settings == nil {
		settings = &config.Settings{
			TimeoutSeconds:      config.DefaultTimeoutSeconds,
			MaxRetries:          config.DefaultMaxRetries,
			DefaultOutputFolder: config.DefaultOutputFolderValue,
			Debug:               false,
			VimMode:             false,
		}
	}

	return DashboardModel{
		Session:           sess,
		Gateway:           gw,
		OutputDir:         outputDir,
		Settings:          settings,
		textInput:         ti,
		updateInput:       ui,
		spinner:           s,
		isCompleted:       completed,
		standards:         standards,
		complianceScores:  complianceScores,
		showScorecard:     showScorecard,
		selectedChoiceIdx: 0,
		showTextInput:     showTextInput,
		genFiles:          genFiles,
		genFileStatuses:   genFileStatuses,
		genFileDetails:    genFileDetails,
		loading:           !completed && len(sess.History) == 0 && sess.LastQuestion == "",
		thoughtChan:       make(chan string, 100),
		streamingTokens:   "",
	}
}

func (m *DashboardModel) StartWithUpdatePrompt() {
	m.showUpdatePrompt = true
	m.isCLIUpdateMode = true
	m.updateInput.Focus()
}

func checkCompletion(scores gateway.ConfidenceScores) bool {
	return scores.Functional >= 100 &&
		scores.Structural >= 100 &&
		scores.Security >= 100 &&
		scores.Compliance >= 100
}

func (m DashboardModel) Init() tea.Cmd {
	logger.LogEvent("TUI", fmt.Sprintf("Dashboard initialized for project: %s", m.Session.ProjectName))
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)

	// Bootstrapping: If history is empty and last question is empty, query Oracle first
	if len(m.Session.History) == 0 && m.Session.LastQuestion == "" {
		cmds = append(cmds, m.queryOracleCmd(""), m.recvThoughtCmd())
	}

	return tea.Batch(cmds...)
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.showViewer {
		return m.handleViewerUpdate(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.loading || (m.isGenerating && !m.isWaitingApproval) {
			return m, nil
		}
		var keyCmd tea.Cmd
		var model tea.Model
		model, keyCmd = m.handleKeyMsg(msg)
		m = model.(DashboardModel)
		if keyCmd != nil {
			cmds = append(cmds, keyCmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case oracleResultMsg:
		return m.handleOracleResult(msg)

	case editorFinishedMsg:
		return m.handleEditorFinished(msg)

	case fileEditorFinishedMsg:
		m.isEditingFile = false
		if msg.err != nil {
			m.setError(fmt.Errorf("file editor failed: %w", msg.err))
		}
		if m.isWaitingApproval && m.showViewer {
			// Reload file contents in the viewport
			return m.openFileViewer()
		}
		return m, nil

	case genProgressMsg:
		return m.handleGenProgress(msg)

	case genFinishedMsg:
		return m.handleGenFinished(msg)

	case contextPruneResultMsg:
		return m.handleContextPruneResult(msg)

	case thoughtTokenMsg:
		m.streamingTokens += string(msg)
		return m, m.recvThoughtCmd()
	}

	if !m.isCompleted && !m.loading && m.showTextInput {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.showUpdatePrompt && !m.loading {
		m.updateInput, cmd = m.updateInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleViewerUpdate processes updates to the document viewer overlay, handling size changes and dismissal key events.
func (m DashboardModel) handleViewerUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
		m.updateViewportSize()
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc || keyMsg.String() == "q" {
			m.showViewer = false
			return m, nil
		}
		if keyMsg.String() == "f" || keyMsg.String() == "F" {
			m.isFullScreenViewer = !m.isFullScreenViewer
			m.updateViewportSize()
			return m, nil
		}
		if m.isWaitingApproval {
			switch strings.ToLower(keyMsg.String()) {
			case "a", "enter":
				m.showViewer = false
				if m.approvalChan != nil {
					close(m.approvalChan)
					m.approvalChan = nil
				}
				m.isWaitingApproval = false
				m.genStatus = "Domain Model approved! Commencing downstream parallel generation..."
				return m, nil
			case "e":
				return m.launchFileEditor("01_domain_model_use_cases.md")
			}
		}
		if m.Settings.VimMode {
			switch keyMsg.String() {
			case "j":
				m.viewport.LineDown(1)
				return m, nil
			case "k":
				m.viewport.LineUp(1)
				return m, nil
			case "d", "ctrl+d":
				m.viewport.HalfPageDown()
				return m, nil
			case "u", "ctrl+u":
				m.viewport.HalfPageUp()
				return m, nil
			}
		}
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateViewportSize computes the width and height of the viewer viewport depending on layout mode.
func (m *DashboardModel) updateViewportSize() {
	width := m.width - 4
	if !m.isFullScreenViewer {
		sidebarWidth := 34
		chatWidth := m.width - sidebarWidth - 8
		if chatWidth < 40 {
			chatWidth = 40
		}
		width = chatWidth - 4
	}
	m.viewport.Width = width
	m.viewport.Height = m.height - 6
}

// handleKeyMsg routes key presses to specific action handlers based on key type.
func (m DashboardModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.isWaitingApproval {
		switch strings.ToLower(msg.String()) {
		case "a", "enter":
			if m.approvalChan != nil {
				close(m.approvalChan)
				m.approvalChan = nil
			}
			m.isWaitingApproval = false
			m.genStatus = "Domain Model approved! Commencing downstream parallel generation..."
			return m, nil
		case "v":
			m.selectedFileIdx = 0
			return m.openFileViewer()
		case "e":
			return m.launchFileEditor("01_domain_model_use_cases.md")
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlK:
		if !m.isCompleted && !m.loading && !m.isGenerating && !m.showUpdatePrompt {
			return m.startOracleQuery("I do not know the answer. Please recommend the best compliance/architectural choice based on industry standards.")
		}
	case tea.KeyEnter:
		return m.handleKeyEnter()
	case tea.KeyUp, tea.KeyPgUp:
		return m.handleKeyUp()
	case tea.KeyDown, tea.KeyPgDown:
		return m.handleKeyDown()
	case tea.KeyLeft:
		return m.handleKeyLeft()
	case tea.KeyRight:
		return m.handleKeyRight()
	case tea.KeyEsc:
		return m.handleKeyEsc()
	case tea.KeyTab:
		return m.handleKeyTab(false)
	case tea.KeyShiftTab:
		return m.handleKeyTab(true)
	case tea.KeyRunes:
		return m.handleKeyRunes(msg)
	}
	return m, nil
}

// handleKeyTab cycles focus or selection in incomplete and completed phases.
func (m DashboardModel) handleKeyTab(shift bool) (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		if shift {
			return m.navigateFilesUp()
		}
		return m.navigateFilesDown()
	}
	if !m.showTextInput {
		m.cycleChoice(shift)
	}
	return m, nil
}

func (m *DashboardModel) cycleChoice(shift bool) {
	choices := m.getChoicesList()
	if len(choices) == 0 {
		return
	}
	if shift {
		m.selectedChoiceIdx--
		if m.selectedChoiceIdx < 0 {
			m.selectedChoiceIdx = len(choices) - 1
		}
	} else {
		m.selectedChoiceIdx++
		if m.selectedChoiceIdx >= len(choices) {
			m.selectedChoiceIdx = 0
		}
	}
}

// handleKeyEnter processes Enter key presses, submitting inputs, selecting options, or launching full editors.
func (m DashboardModel) handleKeyEnter() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m.handleKeyEnterUpdatePrompt()
	}
	if m.isCompleted {
		return m.handleKeyEnterCompleted()
	}
	if m.showTextInput {
		return m.handleKeyEnterTextInput()
	}
	return m.handleKeyEnterChoiceSelection()
}

// handleKeyEnterUpdatePrompt processes Enter key presses inside the manual requirements update prompt.
func (m DashboardModel) handleKeyEnterUpdatePrompt() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.updateInput.Value())
	if val == "" {
		return m, nil
	}
	m.updateInput.SetValue("")
	m.showUpdatePrompt = false
	m.isCompleted = false
	return m.startOracleQuery("I have a new requirement/change: " + val)
}

// handleKeyEnterCompleted processes Enter key presses on the completion screen, opening the document viewer.
func (m DashboardModel) handleKeyEnterCompleted() (tea.Model, tea.Cmd) {
	if m.showViewer {
		return m, nil
	}
	if len(m.genFiles) > 0 {
		return m.openFileViewer()
	}
	return m, nil
}

// handleKeyEnterTextInput processes Enter key presses when typing answers directly into the console.
func (m DashboardModel) handleKeyEnterTextInput() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.textInput.Value())
	if val == "" {
		return m, nil
	}

	m.textInput.SetValue("")

	if val == ":edit" {
		editorCmd, tempPath, err := state.GetEditorCommand(m.Session.ProjectName, m.Session.Facts)
		if err != nil {
			m.setError(err)
			return m, nil
		}
		m.editorTempPath = tempPath
		return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})
	}

	return m.startOracleQuery(val)
}

// handleKeyEnterChoiceSelection processes Enter key presses when selecting choices in the list.
func (m DashboardModel) handleKeyEnterChoiceSelection() (tea.Model, tea.Cmd) {
	choices := m.getChoicesList()
	selected := choices[m.selectedChoiceIdx]

	if selected == "Custom user input..." {
		m.showTextInput = true
		m.textInput.Focus()
		m.textInput.SetValue("")
		return m, nil
	}

	var val string
	if selected == "Let AI decide" {
		val = "Let the AI decide based on current facts and context."
	} else {
		val = m.Session.LastChoices[m.selectedChoiceIdx]
	}

	return m.startOracleQuery(val)
}

// openFileViewer opens the full screen Markdown document viewer viewport overlay.
func (m DashboardModel) openFileViewer() (tea.Model, tea.Cmd) {
	selectedFile := m.genFiles[m.selectedFileIdx]
	dir := m.OutputDir
	if dir == "" {
		dir = filepath.Join(state.GetSessionDir(m.Session.ProjectName), "output")
	}
	filePath := filepath.Join(dir, selectedFile)
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		m.setError(fmt.Errorf("failed to read file %s: %w", selectedFile, err))
		return m, nil
	}
	content := HighlightMarkdown(string(contentBytes))

	m.viewport = viewport.New(0, 0)
	m.updateViewportSize()
	m.viewport.SetContent(content)
	m.showViewer = true
	return m, nil
}

// handleKeyUp navigates upwards through choices in the interactive list.
func (m DashboardModel) handleKeyUp() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.navigateFilesUp()
	}
	if !m.showTextInput {
		choices := m.getChoicesList()
		m.selectedChoiceIdx--
		if m.selectedChoiceIdx < 0 {
			m.selectedChoiceIdx = len(choices) - 1
		}
	}
	return m, nil
}

// handleKeyDown navigates downwards through choices in the interactive list.
func (m DashboardModel) handleKeyDown() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.navigateFilesDown()
	}
	if !m.showTextInput {
		choices := m.getChoicesList()
		m.selectedChoiceIdx++
		if m.selectedChoiceIdx >= len(choices) {
			m.selectedChoiceIdx = 0
		}
	}
	return m, nil
}

// handleKeyLeft navigates leftwards in the file selection layout.
func (m DashboardModel) handleKeyLeft() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.navigateFilesLeft()
	}
	return m, nil
}

// handleKeyRight navigates rightwards in the file selection layout.
func (m DashboardModel) handleKeyRight() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.navigateFilesRight()
	}
	return m, nil
}

// handleKeyEsc dismisses active errors, update prompts, or custom text inputs.
func (m DashboardModel) handleKeyEsc() (tea.Model, tea.Cmd) {
	if m.err != nil {
		m.err = nil
		return m, nil
	}
	if m.showUpdatePrompt {
		m.showUpdatePrompt = false
		m.updateInput.Blur()
		if m.isCLIUpdateMode {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.showTextInput && len(m.Session.LastChoices) > 0 {
		m.showTextInput = false
		m.textInput.Blur()
	}
	return m, nil
}

// handleKeyRunes routes rune character keys based on session completion status.
func (m DashboardModel) handleKeyRunes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.handleKeyRunesCompleted(msg)
	}
	return m.handleKeyRunesIncomplete(msg)
}

// handleKeyRunesCompleted handles shortcut keys (generate, edit, update, quit) on the final completion dashboard screen.
func (m DashboardModel) handleKeyRunesCompleted(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := string(msg.Runes)
	switch strings.ToLower(key) {
	case "g":
		return m.triggerRegeneration()
	case "e":
		return m.launchExternalEditor()
	case "u":
		return m.activateUpdatePrompt()
	case "v":
		if len(m.genFiles) > 0 {
			return m.openFileViewer()
		}
	case "k":
		if m.Settings.VimMode {
			return m.navigateFilesUp()
		}
	case "j":
		if m.Settings.VimMode {
			return m.navigateFilesDown()
		}
	case "h":
		if m.Settings.VimMode {
			return m.navigateFilesLeft()
		}
	case "l":
		if m.Settings.VimMode {
			return m.navigateFilesRight()
		}
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// triggerRegeneration sets states and commands to begin a new specification file generation run.
func (m DashboardModel) triggerRegeneration() (tea.Model, tea.Cmd) {
	m.isGenerating = true
	m.genStatus = "Starting spec generation..."
	m.genPhase = "source"
	m.genChan = make(chan string, 10)
	m.genFileStatuses = make(map[string]string)
	m.genFileDetails = make(map[string]string)
	m.validatorLogs = nil
	for _, f := range m.genFiles {
		m.genFileStatuses[f] = "pending"
	}
	m.approvalChan = make(chan struct{})
	m.isWaitingApproval = false
	return m, tea.Batch(
		m.generateSpecsCmd(),
		m.recvGenProgressCmd(),
	)
}

// launchExternalEditor suspends Bubble Tea UI and runs the external system editor.
func (m DashboardModel) launchExternalEditor() (tea.Model, tea.Cmd) {
	editorCmd, tempPath, err := state.GetEditorCommand(m.Session.ProjectName, m.Session.Facts)
	if err != nil {
		m.setError(err)
		return m, nil
	}
	m.editorTempPath = tempPath
	return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// launchFileEditor suspends Bubble Tea UI and runs the external editor on an arbitrary generated file.
func (m DashboardModel) launchFileEditor(fileName string) (tea.Model, tea.Cmd) {
	dir := m.OutputDir
	if dir == "" {
		dir = filepath.Join(state.GetSessionDir(m.Session.ProjectName), "output")
	}
	filePath := filepath.Join(dir, fileName)
	editorCmd, err := state.GetFileEditorCommand(filePath)
	if err != nil {
		m.setError(err)
		return m, nil
	}
	m.isEditingFile = true
	return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		return fileEditorFinishedMsg{err: err}
	})
}


// activateUpdatePrompt focuses the text input to enter manually updated requirements.
func (m DashboardModel) activateUpdatePrompt() (tea.Model, tea.Cmd) {
	m.showUpdatePrompt = true
	m.updateInput.Focus()
	m.updateInput.SetValue("")
	return m, nil
}

// getFileGridPositions returns the 2D grid mapping of indices in m.genFiles.
func (m DashboardModel) getFileGridPositions() (int, [][]int) {
	sourceIdx := -1
	var downstream []int
	for idx, file := range m.genFiles {
		if file == "01_domain_model_use_cases.md" {
			sourceIdx = idx
		} else {
			downstream = append(downstream, idx)
		}
	}

	if len(downstream) == 0 {
		return sourceIdx, nil
	}

	half := (len(downstream) + 1) / 2
	var grid [][]int
	for i := 0; i < half; i++ {
		row := []int{downstream[i]}
		if half+i < len(downstream) {
			row = append(row, downstream[half+i])
		}
		grid = append(grid, row)
	}
	return sourceIdx, grid
}

// getGridPos determines whether the selected index is the source file or is in the downstream grid.
func (m DashboardModel) getGridPos(selected int, sourceIdx int, grid [][]int) (bool, int, int) {
	if selected == sourceIdx {
		return true, 0, 0
	}
	for r, rowFiles := range grid {
		for c, idx := range rowFiles {
			if idx == selected {
				return false, r, c
			}
		}
	}
	return true, 0, 0
}

// navigateFilesUp moves selectedFileIdx upwards in the completed file list.
func (m DashboardModel) navigateFilesUp() (tea.Model, tea.Cmd) {
	if len(m.genFiles) == 0 {
		return m, nil
	}
	sourceIdx, grid := m.getFileGridPositions()
	if len(grid) == 0 {
		m.selectedFileIdx = 0
		return m, nil
	}
	isSource, row, col := m.getGridPos(m.selectedFileIdx, sourceIdx, grid)
	if isSource {
		m.selectedFileIdx = grid[len(grid)-1][0]
	} else {
		if row > 0 {
			if col < len(grid[row-1]) {
				m.selectedFileIdx = grid[row-1][col]
			} else {
				m.selectedFileIdx = grid[row-1][0]
			}
		} else {
			if sourceIdx != -1 {
				m.selectedFileIdx = sourceIdx
			} else {
				m.selectedFileIdx = grid[len(grid)-1][0]
			}
		}
	}
	return m, nil
}

// navigateFilesDown moves selectedFileIdx downwards in the completed file list.
func (m DashboardModel) navigateFilesDown() (tea.Model, tea.Cmd) {
	if len(m.genFiles) == 0 {
		return m, nil
	}
	sourceIdx, grid := m.getFileGridPositions()
	if len(grid) == 0 {
		m.selectedFileIdx = 0
		return m, nil
	}
	isSource, row, col := m.getGridPos(m.selectedFileIdx, sourceIdx, grid)
	if isSource {
		m.selectedFileIdx = grid[0][0]
	} else {
		if row < len(grid)-1 {
			if col < len(grid[row+1]) {
				m.selectedFileIdx = grid[row+1][col]
			} else {
				m.selectedFileIdx = grid[row+1][0]
			}
		} else {
			if sourceIdx != -1 {
				m.selectedFileIdx = sourceIdx
			} else {
				m.selectedFileIdx = grid[0][col]
			}
		}
	}
	return m, nil
}

// navigateFilesLeft moves selectedFileIdx left in the grid.
func (m DashboardModel) navigateFilesLeft() (tea.Model, tea.Cmd) {
	if len(m.genFiles) == 0 {
		return m, nil
	}
	sourceIdx, grid := m.getFileGridPositions()
	if len(grid) == 0 {
		m.selectedFileIdx = 0
		return m, nil
	}
	isSource, row, col := m.getGridPos(m.selectedFileIdx, sourceIdx, grid)
	if isSource {
		return m, nil
	}
	if col > 0 {
		m.selectedFileIdx = grid[row][col-1]
	} else {
		m.selectedFileIdx = grid[row][len(grid[row])-1]
	}
	return m, nil
}

// navigateFilesRight moves selectedFileIdx right in the grid.
func (m DashboardModel) navigateFilesRight() (tea.Model, tea.Cmd) {
	if len(m.genFiles) == 0 {
		return m, nil
	}
	sourceIdx, grid := m.getFileGridPositions()
	if len(grid) == 0 {
		m.selectedFileIdx = 0
		return m, nil
	}
	isSource, row, col := m.getGridPos(m.selectedFileIdx, sourceIdx, grid)
	if isSource {
		return m, nil
	}
	if col < len(grid[row])-1 {
		m.selectedFileIdx = grid[row][col+1]
	} else {
		m.selectedFileIdx = grid[row][0]
	}
	return m, nil
}

// handleKeyRunesIncomplete processes vi-style navigation keys (j/k) when the requirements phase is still ongoing.
func (m DashboardModel) handleKeyRunesIncomplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showTextInput {
		return m, nil
	}
	key := string(msg.Runes)
	switch strings.ToLower(key) {
	case "g":
		return m.triggerRegeneration()
	case "e":
		return m.launchExternalEditor()
	case "u":
		return m.activateUpdatePrompt()
	case "q":
		return m, tea.Quit
	case "k":
		if m.Settings.VimMode {
			choices := m.getChoicesList()
			m.selectedChoiceIdx--
			if m.selectedChoiceIdx < 0 {
				m.selectedChoiceIdx = len(choices) - 1
			}
		}
	case "j":
		if m.Settings.VimMode {
			choices := m.getChoicesList()
			m.selectedChoiceIdx++
			if m.selectedChoiceIdx >= len(choices) {
				m.selectedChoiceIdx = 0
			}
		}
	}
	return m, nil
}

// handleOracleResult processes updates returned by the Oracle LLM model, saving session state and checking for completeness.
func (m DashboardModel) handleOracleResult(msg oracleResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.setError(msg.err)
		return m, nil
	}

	m.updateSessionState(msg.resp)

	m.Session.Save()
	wasCompleted := m.isCompleted
	m.isCompleted = checkCompletion(m.Session.Scores)

	return m.checkAndTriggerPostOracle(wasCompleted)
}

// updateSessionState updates the session structure fields with response metadata.
func (m *DashboardModel) updateSessionState(resp *gateway.OracleResponse) {
	m.Session.Facts = resp.Facts
	m.Session.Scores = resp.ConfidenceScores
	m.Session.Rationales = resp.DimensionRationales
	m.Session.LastQuestion = resp.NextQuestion
	m.Session.LastChoices = resp.NextChoices
	m.Session.GeneratedFiles = nil
	m.selectedChoiceIdx = 0
	m.showTextInput = len(m.Session.LastChoices) == 0
}

// checkAndTriggerPostOracle performs checks to either initiate background document generation or queue context history compaction.
func (m DashboardModel) checkAndTriggerPostOracle(wasCompleted bool) (tea.Model, tea.Cmd) {
	var batchCmds []tea.Cmd
	if m.isCompleted && !wasCompleted {
		m.isGenerating = true
		m.genStatus = "Starting spec generation..."
		m.genPhase = "source"
		m.genChan = make(chan string, 10)
		m.genFileStatuses = make(map[string]string)
		m.genFileDetails = make(map[string]string)
		m.validatorLogs = nil
		for _, f := range m.genFiles {
			m.genFileStatuses[f] = "pending"
		}
		m.approvalChan = make(chan struct{})
		m.isWaitingApproval = false
		batchCmds = append(batchCmds, m.generateSpecsCmd(), m.recvGenProgressCmd())
	} else if !m.isCompleted {
		m.loading = true
		batchCmds = append(batchCmds, m.pruneContextCmd())
	}
	return m, tea.Batch(batchCmds...)
}

// handleEditorFinished reads back edited requirement documents when external editor processes terminate.
func (m DashboardModel) handleEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.setError(fmt.Errorf("editor failed: %w", msg.err))
		return m, nil
	}

	editedFacts, err := state.ReadBackEditedFacts(m.editorTempPath)
	if err != nil {
		m.setError(fmt.Errorf("failed to read back edited requirements: %w", err))
		return m, nil
	}

	m.Session.Facts = editedFacts
	m.Session.GeneratedFiles = nil
	m.Session.Save()

	return m.startOracleQuery(manualUpdateMsg)
}

// handleGenProgress processes incoming specification file generation progress notifications.
func (m DashboardModel) handleGenProgress(msg genProgressMsg) (tea.Model, tea.Cmd) {
	var ev generator.ProgressEvent
	if err := json.Unmarshal([]byte(msg), &ev); err != nil {
		m.genStatus = string(msg)
		return m, m.recvGenProgressCmd()
	}

	if ev.Status == "started" {
		m.handleGenProgressStart(ev)
	} else if ev.File != "" {
		m.handleGenProgressFile(ev)
	}

	if ev.ValLogs != "" {
		m.handleGenProgressLogs(ev)
	}

	if ev.Message != "" {
		m.genStatus = ev.Message
	}

	if ev.Status == "waiting_approval" {
		m.isWaitingApproval = true
		m.selectedFileIdx = 0
		model, cmd := m.openFileViewer()
		m = model.(DashboardModel)
		return m, tea.Batch(cmd, m.recvGenProgressCmd())
	}

	return m, m.recvGenProgressCmd()
}

// handleGenProgressStart initializes the TUI metadata maps when document synthesis first starts.
func (m *DashboardModel) handleGenProgressStart(ev generator.ProgressEvent) {
	if ev.Phase != "" {
		m.genPhase = ev.Phase
	}
	m.genFiles = strings.Split(ev.Details, ",")
	m.genFileStatuses = make(map[string]string)
	m.genFileDetails = make(map[string]string)
	m.validatorLogs = nil
	for _, f := range m.genFiles {
		m.genFileStatuses[f] = "pending"
	}
}

// handleGenProgressFile updates status information and evaluation details for individual target assets.
func (m *DashboardModel) handleGenProgressFile(ev generator.ProgressEvent) {
	if m.genFileStatuses == nil {
		m.genFileStatuses = make(map[string]string)
	}
	if m.genFileDetails == nil {
		m.genFileDetails = make(map[string]string)
	}
	m.genFileStatuses[ev.File] = ev.Status
	m.genFileDetails[ev.File] = ev.Details
}

// handleGenProgressLogs appends real-time validation logs and limits memory capacity to the most recent 10 lines.
func (m *DashboardModel) handleGenProgressLogs(ev generator.ProgressEvent) {
	lines := strings.Split(ev.ValLogs, "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			m.validatorLogs = append(m.validatorLogs, l)
		}
	}
	if len(m.validatorLogs) > 10 {
		m.validatorLogs = m.validatorLogs[len(m.validatorLogs)-10:]
	}
}

// handleGenFinished saves final status and parses scorecard performance statistics when the asset synthesis ends.
func (m DashboardModel) handleGenFinished(msg genFinishedMsg) (tea.Model, tea.Cmd) {
	m.isGenerating = false
	if msg.err != nil {
		m.setError(msg.err)
	} else {
		m.genStatus = "All specifications synthesized successfully!"
		m.Session.Save()

		dir := m.OutputDir
		if dir == "" {
			dir = filepath.Join(state.GetSessionDir(m.Session.ProjectName), "output")
		}
		metaPath := filepath.Join(dir, ".synthspec-meta.json")
		if metaBytes, readErr := os.ReadFile(metaPath); readErr == nil {
			var meta struct {
				ComplianceSummary map[string]int `json:"compliance_summary"`
			}
			if jsonErr := json.Unmarshal(metaBytes, &meta); jsonErr == nil {
				m.complianceScores = meta.ComplianceSummary
				m.showScorecard = true
				m.genStatus = "All specifications synthesized and audited successfully!"
			}
		}
	}
	return m, nil
}

// handleContextPruneResult processes the outcome of Oracle conversation history context compaction.
func (m DashboardModel) handleContextPruneResult(msg contextPruneResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.setError(fmt.Errorf("context pruning failed: %w", msg.err))
	} else if msg.pruned {
		m.setError(fmt.Errorf("conversation summarized to fit context limit"))
	}
	return m, nil
}

func (m DashboardModel) startOracleQuery(val string) (tea.Model, tea.Cmd) {
	m.loading = true
	m.err = nil
	m.streamingTokens = ""
	return m, tea.Batch(m.queryOracleCmd(val), m.recvThoughtCmd())
}

func (m DashboardModel) recvThoughtCmd() tea.Cmd {
	return func() tea.Msg {
		token, ok := <-m.thoughtChan
		if !ok {
			return nil
		}
		return thoughtTokenMsg(token)
	}
}

// Background commands
// queryOracleCmd submits requirement definitions asynchronously to the LLM Oracle model.
func (m DashboardModel) queryOracleCmd(latestInput string) tea.Cmd {
	logger.LogEvent("TUI", fmt.Sprintf("Querying Oracle with latestInput (length: %d)", len(latestInput)))
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// If user answer was provided, append it to history beforehand
		if latestInput != "" && latestInput != manualUpdateMsg {
			m.Session.AddTurn(latestInput, m.Session.LastQuestion, m.Session.TotalTokensUsed, m.Session.TotalTokensUsed)
		}

		resp, err := m.Gateway.QueryOracleStream(ctx, m.Session.Facts, m.Session.History, latestInput, m.thoughtChan)
		if err != nil {
			return oracleResultMsg{err: err}
		}

		// Update tokens in session (will be saved in Update msg handler)
		if latestInput != "" && latestInput != manualUpdateMsg {
			// Back-fill actual assistant response
			m.Session.History[len(m.Session.History)-1].Content = resp.NextQuestion
		}
		m.Session.TotalTokensUsed = resp.TokensPrompt + resp.TokensCompletion

		return oracleResultMsg{resp: resp}
	}
}

// Receives generator logs asynchronously
// recvGenProgressCmd reads progress logs asynchronously from the pipeline worker channel.
func (m DashboardModel) recvGenProgressCmd() tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-m.genChan
		if !ok {
			return nil
		}
		return genProgressMsg(progress)
	}
}

// Background command to run generation sequentially
// generateSpecsCmd synthesizes all targets in parallel inside background worker goroutines.
func (m DashboardModel) generateSpecsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if m.approvalChan != nil {
			ctx = context.WithValue(ctx, generator.ApprovalChanKey, m.approvalChan)
		}
		err := generator.Generate(ctx, m.Gateway, m.Session, m.OutputDir, m.genChan)
		return genFinishedMsg{err: err}
	}
}

// pruneContextCmd triggers context history summarization when tokens limit is exceeded.
func (m DashboardModel) pruneContextCmd() tea.Cmd {
	return func() tea.Msg {
		pruned, err := m.Session.CheckAndPruneContext(context.Background(), m.Gateway)
		return contextPruneResultMsg{pruned: pruned, err: err}
	}
}

// getChoicesList formats standard and custom options to display on the interactive console list.
func (m DashboardModel) getChoicesList() []string {
	var list []string
	for i, c := range m.Session.LastChoices {
		if i == 0 {
			list = append(list, "(Recommended) "+c)
		} else {
			list = append(list, c)
		}
	}
	list = append(list, "Let AI decide")
	list = append(list, "Custom user input...")
	return list
}

// setError assigns the current model error and formats runtime diagnostic messages to errors.log.
func (m *DashboardModel) setError(err error) {
	m.err = err
	if err != nil {
		var projectName string
		if m.Session != nil {
			projectName = m.Session.ProjectName
		}
		state.LogError(projectName, err)
	}
}

