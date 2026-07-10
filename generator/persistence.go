package generator

import (
	"github.com/toanle/synthspec/domain"
)

// SessionPersistence abstracts session state persistence for the generator.
// This interface allows the generator to be tested without a real filesystem.
type SessionPersistence interface {
	// SaveGeneratedFile persists a generated file's state
	SaveGeneratedFile(state domain.GeneratedFileState) error

	// LoadGeneratedFile retrieves a generated file's state
	LoadGeneratedFile(fileName string) (domain.GeneratedFileState, bool)

	// UpdateFacts updates the compiled facts
	UpdateFacts(facts domain.Facts) error

	// UpdateScores updates confidence scores
	UpdateScores(scores domain.ConfidenceScores, rationales domain.DimensionRationales) error

	// UpdateHistory appends to conversation history
	UpdateHistory(history []domain.Message) error

	// UpdateTokens increments token usage
	UpdateTokens(prompt, completion int) error

	// SaveSession persists the entire session
	SaveSession() error

	// GetProjectName returns the project name
	GetProjectName() string

	// GetProvider returns the provider name
	GetProvider() string

	// GetHistory returns the conversation history
	GetHistory() []domain.Message

	// GetTotalTokens returns the total tokens used
	GetTotalTokens() int

	// GetFacts returns the current facts
	GetFacts() domain.Facts
}

