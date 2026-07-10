package welcome

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/logger"
)

func (m *WelcomeModel) adjustSettingFocus(delta int) {
	if m.SelectedSettingIdx < len(m.settingInputs) {
		m.settingInputs[m.SelectedSettingIdx].Blur()
	}

	totalSettings := len(m.settingInputs) + 2
	m.SelectedSettingIdx = (m.SelectedSettingIdx + delta + totalSettings) % totalSettings

	if m.SelectedSettingIdx < len(m.settingInputs) {
		m.settingInputs[m.SelectedSettingIdx].Focus()
	}
}

func (m *WelcomeModel) saveSettingsFromInputs() {
	var tSec, mRet int
	var bCap float64
	_, _ = fmt.Sscanf(m.settingInputs[0].Value(), "%d", &tSec)
	_, _ = fmt.Sscanf(m.settingInputs[1].Value(), "%d", &mRet)
	outFolder := strings.TrimSpace(m.settingInputs[2].Value())
	_, _ = fmt.Sscanf(m.settingInputs[3].Value(), "%f", &bCap)

	if tSec > 0 {
		m.Settings.TimeoutSeconds = tSec
	}
	if mRet >= 0 {
		m.Settings.MaxRetries = mRet
	}
	if outFolder != "" {
		m.Settings.DefaultOutputFolder = outFolder
	}
	if bCap >= 0 {
		m.Settings.HardBudgetCap = bCap
	}

	_ = config.SaveSettings(m.Settings, true)
	_ = config.SaveSettings(m.Settings, false)

	logger.LogEvent("TUI", fmt.Sprintf("Saved settings: timeout_seconds=%d max_retries=%d default_output_folder=%s debug=%t vim_mode=%t hard_budget_cap=%.2f", m.Settings.TimeoutSeconds, m.Settings.MaxRetries, m.Settings.DefaultOutputFolder, m.Settings.Debug, m.Settings.VimMode, m.Settings.HardBudgetCap))
	_ = logger.Init(false, m.Settings.Debug)
	m.Phase = PhaseMenu
}

func (m WelcomeModel) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "up", keyShiftTab:
		m.adjustSettingFocus(-1)
	case "k":
		if m.Settings.VimMode {
			m.adjustSettingFocus(-1)
		}
	case "down", "tab":
		m.adjustSettingFocus(1)
	case "j":
		if m.Settings.VimMode {
			m.adjustSettingFocus(1)
		}
	case " ", "space":
		if m.SelectedSettingIdx == len(m.settingInputs) {
			m.Settings.Debug = !m.Settings.Debug
			logger.LogEvent("TUI", fmt.Sprintf("Debug logging toggled: %t", m.Settings.Debug))
		} else if m.SelectedSettingIdx == len(m.settingInputs)+1 {
			m.Settings.VimMode = !m.Settings.VimMode
			logger.LogEvent("TUI", fmt.Sprintf("Vim mode toggled: %t", m.Settings.VimMode))
		}
	case "enter":
		m.saveSettingsFromInputs()
	case "esc":
		m.Phase = PhaseMenu
	default:
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx], cmd = m.settingInputs[m.SelectedSettingIdx].Update(msg)
		}
	}
	return m, cmd
}
