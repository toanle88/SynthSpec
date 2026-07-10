package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/logger"
)



// Session represents a project session state
type Session struct {
	mu                    sync.Mutex                 `json:"-"`
	ProjectName           string                     `json:"project_name"`
	Provider              string                     `json:"provider"`
	Model                 string                     `json:"model"`
	CreatedAt             time.Time                  `json:"created_at"`
	UpdatedAt             time.Time                  `json:"updated_at"`
	History               []domain.Message           `json:"history"`
	Facts                 domain.Facts               `json:"facts"`
	Scores                domain.ConfidenceScores    `json:"scores"`
	Rationales            domain.DimensionRationales `json:"rationales"`
	LastQuestion          string                     `json:"last_question"`
	LastChoices           []string                   `json:"last_choices"`
	TotalTokensUsed       int                        `json:"total_tokens_used"`
	TotalPromptTokens     int                        `json:"total_prompt_tokens"`
	TotalCompletionTokens int                        `json:"total_completion_tokens"`
	GeneratedFiles        []domain.GeneratedFileState `json:"generated_files,omitempty"`
}

// GetSessionDir returns the project directory path
func GetSessionDir(projectName string) string {
	return filepath.Join(config.GetSynthspecRoot(), projectName)
}

// GetSessionPath returns the project session.json file path
func GetSessionPath(projectName string) string {
	return filepath.Join(GetSessionDir(projectName), "session.json")
}

// Save persists the session state to disk
func (s *Session) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Session) saveLocked() error {
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
	root := config.GetSynthspecRoot()
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
func (s *Session) SaveGeneratedFile(state domain.GeneratedFileState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for idx, gf := range s.GeneratedFiles {
		if gf.FileName == state.FileName {
			s.GeneratedFiles[idx] = state
			found = true
			break
		}
	}
	if !found {
		s.GeneratedFiles = append(s.GeneratedFiles, state)
	}

	return s.saveLocked()
}

// LoadGeneratedFile retrieves a generated file's state
func (s *Session) LoadGeneratedFile(fileName string) (domain.GeneratedFileState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, gf := range s.GeneratedFiles {
		if gf.FileName == fileName {
			return gf, true
		}
	}
	return domain.GeneratedFileState{}, false
}

// UpdateFacts updates the compiled facts
func (s *Session) UpdateFacts(facts domain.Facts) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Facts = facts
	return s.saveLocked()
}

// UpdateScores updates confidence scores
func (s *Session) UpdateScores(scores domain.ConfidenceScores, rationales domain.DimensionRationales) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Scores = scores
	s.Rationales = rationales
	return s.saveLocked()
}

// UpdateHistory appends to conversation history
func (s *Session) UpdateHistory(history []domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.History = history
	return s.saveLocked()
}

// UpdateTokens increments token usage
func (s *Session) UpdateTokens(prompt, completion int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalPromptTokens += prompt
	s.TotalCompletionTokens += completion
	s.TotalTokensUsed += prompt + completion
	return s.saveLocked()
}

// SaveSession persists the entire session
func (s *Session) SaveSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

// GetProjectName returns the project name
func (s *Session) GetProjectName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ProjectName
}

// GetProvider returns the provider name
func (s *Session) GetProvider() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Provider
}

// GetHistory returns the conversation history
func (s *Session) GetHistory() []domain.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.History
}

// GetTotalTokens returns the total tokens used
func (s *Session) GetTotalTokens() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.TotalTokensUsed
}

// GetFacts returns the current facts
func (s *Session) GetFacts() domain.Facts {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Facts
}

// AddTurn appends a conversation turn and updates the total tokens
func (s *Session) AddTurn(userMsg, assistantMsg string, tokensPrompt, tokensCompletion int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.History = append(s.History, domain.Message{Role: "user", Content: userMsg})
	s.History = append(s.History, domain.Message{Role: "assistant", Content: assistantMsg})
	s.TotalPromptTokens += tokensPrompt
	s.TotalCompletionTokens += tokensCompletion
	s.TotalTokensUsed += tokensPrompt + tokensCompletion
}

// EstimateHistoryTokens returns the estimated token count of the conversation history.
func (s *Session) EstimateHistoryTokens() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	totalChars := 0
	for _, msg := range s.History {
		totalChars += len(msg.Content)
	}
	// Conservative estimate: 1 token ≈ 3.5 characters
	return int(float64(totalChars) / 3.5)
}

func (s *Session) GetTotalPromptTokens() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.TotalPromptTokens
}

func (s *Session) GetTotalCompletionTokens() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.TotalCompletionTokens
}

func (s *Session) GetEstimatedCost() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return config.CalculateCost(s.Model, s.TotalPromptTokens, s.TotalCompletionTokens)
}

func (s *Session) CheckBudget(cap float64) error {
	if cap > 0.0 {
		cost := s.GetEstimatedCost()
		if cost >= cap {
			return domain.ErrBudgetExceeded
		}
	}
	return nil
}

func (s *Session) GetModel() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Model
}

func (s *Session) GetScores() domain.ConfidenceScores {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Scores
}

func (s *Session) GetRationales() domain.DimensionRationales {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Rationales
}

func (s *Session) GetLastQuestion() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.LastQuestion
}

func (s *Session) GetLastChoices() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.LastChoices
}

func (s *Session) GetGeneratedFiles() []domain.GeneratedFileState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.GeneratedFiles
}

func (s *Session) SetInterrogationState(lastQuestion string, lastChoices []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastQuestion = lastQuestion
	s.LastChoices = lastChoices
	return s.saveLocked()
}

func (s *Session) AddTokens(tokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalPromptTokens += tokens
	s.TotalTokensUsed += tokens
	return s.saveLocked()
}

func (s *Session) ClearGeneratedFiles() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GeneratedFiles = nil
	return s.saveLocked()
}

func (s *Session) SetGeneratedFiles(files []domain.GeneratedFileState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GeneratedFiles = files
	return s.saveLocked()
}

// SaveHistoryState saves a snapshot of the current session to the history folder.
func (s *Session) SaveHistoryState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(GetSessionDir(s.ProjectName), "history")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize session for history: %w", err)
	}

	ts := time.Now().UnixNano()
	filename := filepath.Join(dir, fmt.Sprintf("state_%d.json", ts))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	// Prune history to keep at most 10 states
	entries, err := os.ReadDir(dir)
	if err == nil {
		var historyFiles []string
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
				historyFiles = append(historyFiles, entry.Name())
			}
		}
		// Sort entries (they are named state_<unixnano>.json, alphabetical sort aligns with chronological)
		if len(historyFiles) > 10 {
			// Sort is natural for UnixNano string representation of same length (roughly) or simple sort
			// Let's delete the oldest ones (first in alphabetical order)
			for i := 0; i < len(historyFiles)-10; i++ {
				_ = os.Remove(filepath.Join(dir, historyFiles[i]))
			}
		}
	}

	return nil
}

// Undo restores the most recent history state.
func (s *Session) Undo() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(GetSessionDir(s.ProjectName), "history")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("no history available: %w", err)
	}

	var historyFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			historyFiles = append(historyFiles, entry.Name())
		}
	}

	if len(historyFiles) == 0 {
		return fmt.Errorf("no history states found to undo")
	}

	// The last file in alphabetical order is the newest
	newestFile := historyFiles[len(historyFiles)-1]
	path := filepath.Join(dir, newestFile)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read history snapshot: %w", err)
	}

	// Unmarshal into a temp session first to avoid partial corruption
	var temp Session
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to parse history snapshot: %w", err)
	}

	// Restore fields
	s.Provider = temp.Provider
	s.Model = temp.Model
	s.History = temp.History
	s.Facts = temp.Facts
	s.Scores = temp.Scores
	s.Rationales = temp.Rationales
	s.LastQuestion = temp.LastQuestion
	s.LastChoices = temp.LastChoices
	s.TotalTokensUsed = temp.TotalTokensUsed
	s.TotalPromptTokens = temp.TotalPromptTokens
	s.TotalCompletionTokens = temp.TotalCompletionTokens
	s.GeneratedFiles = temp.GeneratedFiles

	// Save to session.json
	if err := s.saveLocked(); err != nil {
		return fmt.Errorf("failed to save restored session: %w", err)
	}

	// Remove the undone snapshot
	_ = os.Remove(path)

	return nil
}

