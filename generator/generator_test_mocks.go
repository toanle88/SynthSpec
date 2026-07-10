package generator

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/gateway"
)

// TestGateway implements gateway.Gateway for unit tests
type TestGateway struct {
	responses   map[string][]string // filename -> slice of responses (for mocking retries)
	callCounts  map[string]int
	queryCount  int
	queryErr    error
	queryResult *gateway.OracleResponse
	mu          sync.Mutex
}

func (tg *TestGateway) QueryOracle(ctx context.Context, facts gateway.Facts, history []gateway.Message, latestInput string) (*gateway.OracleResponse, error) {
	tg.mu.Lock()
	tg.queryCount++
	tg.mu.Unlock()
	return tg.queryResult, tg.queryErr
}

func (tg *TestGateway) QueryOracleStream(ctx context.Context, facts gateway.Facts, history []gateway.Message, latestInput string, tokenChan chan<- string) (*gateway.OracleResponse, error) {
	res, err := tg.QueryOracle(ctx, facts, history, latestInput)
	if err != nil {
		close(tokenChan)
		return nil, err
	}
	domain.StreamOracleResponse(res, tokenChan)
	return res, nil
}

const mockErrPrefix = "ERROR:"

func (tg *TestGateway) GenerateSpecFile(ctx context.Context, facts gateway.Facts, fileName string, promptTemplate string) (string, error) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.callCounts[fileName]++
	resps, ok := tg.responses[fileName]
	if !ok || len(resps) == 0 {
		if fileName == "04_openapi_contract.yaml" {
			return "openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}", nil
		}
		return "Mock generic content", nil
	}

	count := tg.callCounts[fileName]
	var resp string
	if count > len(resps) {
		resp = resps[len(resps)-1]
	} else {
		resp = resps[count-1]
	}
	if strings.HasPrefix(resp, mockErrPrefix) {
		return "", errors.New(strings.TrimPrefix(resp, mockErrPrefix))
	}
	return resp, nil
}

func (tg *TestGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []config.Standard) ([]gateway.ComplianceResult, error) {
	var results []gateway.ComplianceResult
	for _, std := range standards {
		hasTarget := false
		for _, tf := range std.TargetFiles {
			if tf == fileName {
				hasTarget = true
				break
			}
		}
		if !hasTarget {
			continue
		}
		results = append(results, gateway.ComplianceResult{
			StandardID: std.ID,
			Score:      100,
			Compliant:  true,
			Feedback:   "Mock passing standard",
		})
	}
	return results, nil
}

func (tg *TestGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []config.Standard, referenceDoc string) (string, error) {
	tg.mu.Lock()
	tg.callCounts[fileName]++
	resps, ok := tg.responses[fileName]
	tg.mu.Unlock()
	if !ok || len(resps) == 0 {
		return fileContent, nil
	}

	count := tg.callCounts[fileName]
	var resp string
	if count > len(resps) {
		resp = resps[len(resps)-1]
	} else {
		resp = resps[count-1]
	}
	if strings.HasPrefix(resp, mockErrPrefix) {
		return "", errors.New(strings.TrimPrefix(resp, mockErrPrefix))
	}
	return resp, nil
}

func (tg *TestGateway) VerifyConsistency(ctx context.Context, files map[string]string) (*gateway.ConsistencyReport, error) {
	for fileName, content := range files {
		if strings.Contains(content, "TRIGGER_INCONSISTENCY") {
			return &gateway.ConsistencyReport{
				Consistent: false,
				Feedback: map[string]string{
					fileName: "Consistency check failed feedback.",
				},
			}, nil
		}
	}
	return &gateway.ConsistencyReport{
		Consistent: true,
		Feedback:   make(map[string]string),
	}, nil
}

// Summarize generates a mock summary for testing
func (tg *TestGateway) Summarize(ctx context.Context, history []gateway.Message) (string, error) {
	return "Mock summary of conversation history for testing", nil
}

func (tg *TestGateway) ExtractStructuralEntities(ctx context.Context, sourceDoc string) (string, error) {
	return `{"entities":[{"name":"MockEntity"}]}`, nil
}

func (tg *TestGateway) OptimizePrompt(ctx context.Context, files map[string]string) (string, error) {
	return "Mock optimized prompt", nil
}

func (tg *TestGateway) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	res := make([][]float32, len(texts))
	for i := range texts {
		res[i] = make([]float32, 128)
		res[i][0] = 1.0 // Simple dummy normalized vector
	}
	return res, nil
}

func (tg *TestGateway) RegisterTokenCounter(fn func(prompt, completion int)) {
	// Not implemented for mock
}

func (tg *TestGateway) RegisterBudgetCheck(fn func() error) {
	// Not implemented for mock
}


// MockPersistence implements generator.SessionPersistence for testing
type MockPersistence struct {
	projectName string
	provider    string
	history     []domain.Message
	facts       domain.Facts
	totalTokens int
	files       map[string]domain.GeneratedFileState
	mu          sync.Mutex
}

func NewMockPersistence() *MockPersistence {
	return &MockPersistence{
		projectName: "test-project",
		provider:    "test-provider",
		history:     []domain.Message{},
		facts:       domain.Facts{},
		totalTokens: 0,
		files:       make(map[string]domain.GeneratedFileState),
	}
}

func (mp *MockPersistence) SaveGeneratedFile(state domain.GeneratedFileState) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.files[state.FileName] = state
	return nil
}

func (mp *MockPersistence) LoadGeneratedFile(fileName string) (domain.GeneratedFileState, bool) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	state, ok := mp.files[fileName]
	return state, ok
}

func (mp *MockPersistence) UpdateFacts(facts domain.Facts) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.facts = facts
	return nil
}

func (mp *MockPersistence) UpdateScores(scores domain.ConfidenceScores, rationales domain.DimensionRationales) error {
	return nil
}

func (mp *MockPersistence) UpdateHistory(history []domain.Message) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.history = history
	return nil
}

func (mp *MockPersistence) UpdateTokens(prompt, completion int) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.totalTokens += prompt + completion
	return nil
}

func (mp *MockPersistence) SaveSession() error {
	return nil
}

func (mp *MockPersistence) GetProjectName() string {
	return mp.projectName
}

func (mp *MockPersistence) GetProvider() string {
	return mp.provider
}

func (mp *MockPersistence) GetHistory() []domain.Message {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.history
}

func (mp *MockPersistence) GetTotalTokens() int {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.totalTokens
}

func (mp *MockPersistence) GetFacts() domain.Facts {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.facts
}

type blockingGateway struct {
	*TestGateway
	started chan string
	release <-chan struct{}
	blocked map[string]bool
}

func (bg *blockingGateway) GenerateSpecFile(ctx context.Context, facts gateway.Facts, fileName string, promptTemplate string) (string, error) {
	if bg.blocked[fileName] {
		bg.started <- fileName
		<-bg.release
	}
	return bg.TestGateway.GenerateSpecFile(ctx, facts, fileName, promptTemplate)
}
