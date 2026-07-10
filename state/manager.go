package state

import (
	"context"

	"github.com/toanle/synthspec/domain"
)

type SessionManager interface {
	GetProjectName() string
	GetProvider() string
	GetModel() string
	GetFacts() domain.Facts
	UpdateFacts(domain.Facts) error
	GetScores() domain.ConfidenceScores
	GetRationales() domain.DimensionRationales
	UpdateScores(domain.ConfidenceScores, domain.DimensionRationales) error
	GetLastQuestion() string
	GetLastChoices() []string
	SetInterrogationState(string, []string) error
	GetHistory() []domain.Message
	UpdateHistory([]domain.Message) error
	GetTotalTokens() int
	GetEstimatedCost() float64
	UpdateTokens(prompt, completion int) error
	AddTokens(int) error
	AddTurn(string, string, int, int)
	Save() error
	SaveSession() error
	ClearGeneratedFiles() error
	CheckAndPruneContext(context.Context, ContextSummarizer) (bool, error)
	GetGeneratedFiles() []domain.GeneratedFileState
	SetGeneratedFiles([]domain.GeneratedFileState) error
	LoadGeneratedFile(fileName string) (domain.GeneratedFileState, bool)
	SaveGeneratedFile(state domain.GeneratedFileState) error
	SaveHistoryState() error
	Undo() error
}
