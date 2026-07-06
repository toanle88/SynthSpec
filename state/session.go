package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/generator"
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

// SessionPersistence implementation for generator.SessionPersistence interface

// SaveGeneratedFile persists a generated file's state
func (s *Session) SaveGeneratedFile(state generator.GeneratedFileState) error {
	// Convert generator.GeneratedFileState to state.GeneratedFileState
	genState := GeneratedFileState{
		FileName:       state.FileName,
		Results:        state.Results,
		HasError:       state.HasError,
		ErrMsg:         state.ErrMsg,
		InProgressText: state.InProgressText,
		CurrentAttempt: state.CurrentAttempt,
		PromptHash:     state.PromptHash,
		FactsHash:      state.FactsHash,
	}

	found := false
	for idx, gf := range s.GeneratedFiles {
		if gf.FileName == state.FileName {
			s.GeneratedFiles[idx] = genState
			found = true
			break
		}
	}
	if !found {
		s.GeneratedFiles = append(s.GeneratedFiles, genState)
	}

	return s.Save()
}

// LoadGeneratedFile retrieves a generated file's state
func (s *Session) LoadGeneratedFile(fileName string) (generator.GeneratedFileState, bool) {
	for _, gf := range s.GeneratedFiles {
		if gf.FileName == fileName {
			return generator.GeneratedFileState{
				FileName:       gf.FileName,
				Results:        gf.Results,
				HasError:       gf.HasError,
				ErrMsg:         gf.ErrMsg,
				InProgressText: gf.InProgressText,
				CurrentAttempt: gf.CurrentAttempt,
				PromptHash:     gf.PromptHash,
				FactsHash:      gf.FactsHash,
			}, true
		}
	}
	return generator.GeneratedFileState{}, false
}

// UpdateFacts updates the compiled facts
func (s *Session) UpdateFacts(facts domain.Facts) error {
	s.Facts = facts
	return s.Save()
}

// UpdateScores updates confidence scores
func (s *Session) UpdateScores(scores domain.ConfidenceScores, rationales domain.DimensionRationales) error {
	s.Scores = scores
	s.Rationales = rationales
	return s.Save()
}

// UpdateHistory appends to conversation history
func (s *Session) UpdateHistory(history []domain.Message) error {
	s.History = history
	return s.Save()
}

// UpdateTokens increments token usage
func (s *Session) UpdateTokens(prompt, completion int) error {
	s.TotalTokensUsed += prompt + completion
	return s.Save()
}

// SaveSession persists the entire session
func (s *Session) SaveSession() error {
	return s.Save()
}

// GetProjectName returns the project name
func (s *Session) GetProjectName() string {
	return s.ProjectName
}

// GetProvider returns the provider name
func (s *Session) GetProvider() string {
	return s.Provider
}

// GetHistory returns the conversation history
func (s *Session) GetHistory() []domain.Message {
	return s.History
}

// GetTotalTokens returns the total tokens used
func (s *Session) GetTotalTokens() int {
	return s.TotalTokensUsed
}

// GetFacts returns the current facts
func (s *Session) GetFacts() domain.Facts {
	return s.Facts
}

// AddTurn appends a conversation turn and updates the total tokens
func (s *Session) AddTurn(userMsg, assistantMsg string, tokensPrompt, tokensCompletion int) {
	s.History = append(s.History, domain.Message{Role: "user", Content: userMsg})
	s.History = append(s.History, domain.Message{Role: "assistant", Content: assistantMsg})
	s.TotalTokensUsed += tokensPrompt + tokensCompletion
}
