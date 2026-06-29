package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/logger"
)

// GeneratedFileState represents the status and compliance audit of a generated file
type GeneratedFileState struct {
	FileName       string                    `json:"file_name"`
	Results        []domain.ComplianceResult `json:"results"`
	HasError       bool                      `json:"has_error"`
	ErrMsg         string                    `json:"error_str,omitempty"`
	InProgressText string                    `json:"in_progress_text,omitempty"`
	CurrentAttempt int                       `json:"current_attempt,omitempty"`
	PromptHash     string                    `json:"prompt_hash,omitempty"`
	FactsHash      string                    `json:"facts_hash,omitempty"`
}

// Session represents a project session state
type Session struct {
	ProjectName     string                     `json:"project_name"`
	Provider        string                     `json:"provider"`
	Model           string                     `json:"model"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	History         []domain.Message           `json:"history"`
	Facts           domain.Facts               `json:"facts"`
	Scores          domain.ConfidenceScores    `json:"scores"`
	Rationales      domain.DimensionRationales `json:"rationales"`
	LastQuestion    string                     `json:"last_question"`
	LastChoices     []string                   `json:"last_choices"`
	TotalTokensUsed int                        `json:"total_tokens_used"`
	GeneratedFiles  []GeneratedFileState       `json:"generated_files,omitempty"`
}

// getSynthspecRoot returns the base directory for SynthSpec data.
// It prefers the user's config directory with a fallback to the current working directory.
func getSynthspecRoot() string {
	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, "synthspec")
	}
	return "synthspec"
}

// GetSessionDir returns the project directory path
func GetSessionDir(projectName string) string {
	return filepath.Join(getSynthspecRoot(), projectName)
}

// GetSessionPath returns the project session.json file path
func GetSessionPath(projectName string) string {
	return filepath.Join(GetSessionDir(projectName), "session.json")
}

// Save persists the session state to disk
func (s *Session) Save() error {
	logger.LogEvent("STATE", fmt.Sprintf("Saving session for project '%s'", s.ProjectName))
	s.UpdatedAt = time.Now()
	dir := GetSessionDir(s.ProjectName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.LogEvent("STATE", fmt.Sprintf("Error creating directory for project '%s': %v", s.ProjectName, err))
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		logger.LogEvent("STATE", fmt.Sprintf("Error marshaling session for project '%s': %v", s.ProjectName, err))
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	path := GetSessionPath(s.ProjectName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		logger.LogEvent("STATE", fmt.Sprintf("Error writing session file for project '%s': %v", s.ProjectName, err))
		return fmt.Errorf("failed to write session file: %w", err)
	}

	logger.LogEvent("STATE", fmt.Sprintf("Successfully saved session to '%s'", path))
	return nil
}

// LoadSession reads a session state from disk
func LoadSession(projectName string) (*Session, error) {
	path := GetSessionPath(projectName)
	logger.LogEvent("STATE", fmt.Sprintf("Loading session from '%s'", path))
	data, err := os.ReadFile(path)
	if err != nil {
		logger.LogEvent("STATE", fmt.Sprintf("Error reading session file '%s': %v", path, err))
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		logger.LogEvent("STATE", fmt.Sprintf("Error parsing session file '%s': %v", path, err))
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	logger.LogEvent("STATE", fmt.Sprintf("Successfully loaded session for project '%s'", s.ProjectName))
	return &s, nil
}

// ListProjects scans the local directory for active SynthSpec projects
func ListProjects() ([]string, error) {
	root := getSynthspecRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			sessionPath := filepath.Join(root, entry.Name(), "session.json")
			if _, err := os.Stat(sessionPath); err == nil {
				projects = append(projects, entry.Name())
			}
		}
	}
	return projects, nil
}

// AddTurn appends a conversation turn and updates the total tokens
func (s *Session) AddTurn(userMsg, assistantMsg string, tokensPrompt, tokensCompletion int) {
	s.History = append(s.History, domain.Message{Role: "user", Content: userMsg})
	s.History = append(s.History, domain.Message{Role: "assistant", Content: assistantMsg})
	s.TotalTokensUsed += tokensPrompt + tokensCompletion
}

const errorLogFile = "errors.log"

// LogError writes an error message with a timestamp to the project's error log file.
// If projectName is empty, it writes to the global error log file.
func LogError(projectName string, err error) {
	if err == nil {
		return
	}

	root := getSynthspecRoot()

	// Create root directory if it doesn't exist
	if errMk := os.MkdirAll(root, 0755); errMk != nil {
		return // Silently fail if we can't write to disk
	}

	var logPath string
	if projectName != "" {
		projDir := GetSessionDir(projectName)
		if errMk := os.MkdirAll(projDir, 0755); errMk == nil {
			logPath = filepath.Join(projDir, errorLogFile)
		} else {
			logPath = filepath.Join(root, errorLogFile)
		}
	} else {
		logPath = filepath.Join(root, errorLogFile)
	}

	f, errOpen := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if errOpen != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] ERROR: %v\n", timestamp, err)
	_, _ = f.WriteString(logEntry)
}
