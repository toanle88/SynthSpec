package gateway

import (
	"context"

	"github.com/toanle/synthspec/domain"
)

const (
	contentTypeHeader = "Content-Type"
	applicationJSON   = "application/json"
	authBearerPrefix  = "Bearer "
)

// Message represents a single turn in the conversation history
type Message = domain.Message

// Facts represents the compiled specification details
type Facts = domain.Facts

// ConfidenceScores represents confidence levels (0-100) across 4 dimensions
type ConfidenceScores = domain.ConfidenceScores

// DimensionRationales contains brief reasoning behind each dimension's score
type DimensionRationales = domain.DimensionRationales

// OracleResponse represents the structured JSON response from the LLM Oracle
type OracleResponse = domain.OracleResponse

// ComplianceResult represents the evaluation result of a single standard
type ComplianceResult = domain.ComplianceResult

// ConsistencyReport represents the evaluation of cross-document logical consistency
type ConsistencyReport = domain.ConsistencyReport

// Gateway defines the uniform interface for communicating with upstream LLMs
type Gateway interface {
	// QueryOracle sends the current facts, conversation history, and latest input
	// to get the next interrogation state from the Oracle.
	QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string, currentScores ConfidenceScores, currentRationales DimensionRationales) (*OracleResponse, error)

	// QueryOracleStream does the same as QueryOracle but streams the raw tokens/chunks back via tokenChan.
	QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, currentScores ConfidenceScores, currentRationales DimensionRationales, tokenChan chan<- string) (*OracleResponse, error)

	// GenerateSpecFile generates the contents of a specific output asset based on the compiled facts.
	GenerateSpecFile(ctx context.Context, facts Facts, fileName string, promptTemplate string) (string, error)

	// EvaluateCompliance evaluates a generated file's content against a set of standards
	EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []domain.Standard) ([]ComplianceResult, error)

	// RefineSpecFile attempts to fix a generated file to comply with standards based on feedback
	RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (string, error)

	// VerifyConsistency checks consistency across all generated documents
	VerifyConsistency(ctx context.Context, files map[string]string) (*ConsistencyReport, error)

	// Summarize generates a concise summary of the conversation history.
	Summarize(ctx context.Context, history []Message) (string, error)

	// ExtractStructuralEntities converts a markdown document into a dense JSON payload
	ExtractStructuralEntities(ctx context.Context, sourceDoc string) (string, error)

	// OptimizePrompt condenses generated markdown documents into dense, absolute, imperative directives.
	OptimizePrompt(ctx context.Context, files map[string]string) (string, error)

	// GenerateEmbeddings calculates numeric vector embeddings for a slice of texts.
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)

	// RegisterTokenCounter registers a callback to be triggered when tokens are consumed.
	RegisterTokenCounter(fn func(prompt, completion int))

	// RegisterBudgetCheck registers a callback to verify if the budget allows further requests.
	RegisterBudgetCheck(fn func() error)
}
