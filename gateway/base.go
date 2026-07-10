package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/logger"
)

// ProviderAdapter defines the provider-specific implementation details
type ProviderAdapter interface {
	ProviderName() string
	ModelName() string

	BuildOracleRequest(facts domain.Facts, history []domain.Message, latestInput string) (*http.Request, error)
	ParseOracleResponse(body []byte) (*domain.OracleResponse, int, int, error)

	BuildGenerateSpecRequest(facts domain.Facts, fileName string, promptTemplate string) (*http.Request, error)
	ParseGenerateSpecResponse(body []byte) (string, int, int, error)

	BuildEvaluateComplianceRequest(fileName string, fileContent string, standards []domain.Standard) (*http.Request, error)
	ParseEvaluateComplianceResponse(body []byte) ([]domain.ComplianceResult, int, int, error)

	BuildRefineSpecRequest(fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (*http.Request, error)
	ParseRefineSpecResponse(body []byte) (string, int, int, error)

	BuildVerifyConsistencyRequest(files map[string]string) (*http.Request, error)
	ParseVerifyConsistencyResponse(body []byte) (*domain.ConsistencyReport, int, int, error)

	BuildSummarizeRequest(history []domain.Message) (*http.Request, error)
	ParseSummarizeResponse(body []byte) (string, int, int, error)

	BuildExtractStructuralEntitiesRequest(sourceDoc string) (*http.Request, error)
	BuildOptimizePromptRequest(files map[string]string) (*http.Request, error)
}

// BaseGateway implements the Gateway interface and handles common logic (retries, logging, etc.)
type BaseGateway struct {
	adapter      ProviderAdapter
	client       *http.Client
	maxRetries   int
	onTokenUsage func(prompt, completion int)
	budgetCheck  func() error
}

func NewBaseGateway(adapter ProviderAdapter, timeout time.Duration, maxRetries int) *BaseGateway {
	return &BaseGateway{
		adapter:    adapter,
		client:     &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
	}
}

func (b *BaseGateway) RegisterTokenCounter(fn func(prompt, completion int)) {
	b.onTokenUsage = fn
}

func (b *BaseGateway) RegisterBudgetCheck(fn func() error) {
	b.budgetCheck = fn
}

func (b *BaseGateway) executeRequest(ctx context.Context, req *http.Request) ([]byte, time.Duration, error) {
	if b.budgetCheck != nil {
		if err := b.budgetCheck(); err != nil {
			return nil, 0, err
		}
	}
	req = req.WithContext(ctx)
	startTime := time.Now()
	respBytes, err := SendWithRetry(ctx, b.client, req, b.maxRetries)
	duration := time.Since(startTime)
	return respBytes, duration, err
}

// buildJSONRequest is a helper for adapters to build standard POST JSON requests
func buildJSONRequest(url string, reqBody interface{}, headers map[string]string) (*http.Request, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

func (b *BaseGateway) QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string) (*OracleResponse, error) {
	req, err := b.adapter.BuildOracleRequest(facts, history, latestInput)
	if err != nil {
		return nil, err
	}

	respBytes, duration, err := b.executeRequest(ctx, req)
	if err != nil {
		logger.LogAPI(b.adapter.ProviderName(), b.adapter.ModelName(), duration, 0, 0, err)
		return nil, err
	}

	oracleResp, promptTokens, completionTokens, err := b.adapter.ParseOracleResponse(respBytes)
	if err != nil {
		logger.LogAPI(b.adapter.ProviderName(), b.adapter.ModelName(), duration, promptTokens, completionTokens, err)
		return nil, err
	}

	logger.LogAPI(b.adapter.ProviderName(), b.adapter.ModelName(), duration, promptTokens, completionTokens, nil)

	oracleResp.TokensPrompt = promptTokens
	oracleResp.NextQuestion = SanitizeNextQuestion(oracleResp.NextQuestion)
	oracleResp.TokensCompletion = completionTokens

	if b.onTokenUsage != nil {
		b.onTokenUsage(promptTokens, completionTokens)
	}

	return oracleResp, nil
}

func (b *BaseGateway) QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, tokenChan chan<- string) (*OracleResponse, error) {
	res, err := b.QueryOracle(ctx, facts, history, latestInput)
	if err != nil {
		close(tokenChan)
		return nil, err
	}
	domain.StreamOracleResponse(res, tokenChan)
	return res, nil
}

func (b *BaseGateway) GenerateSpecFile(ctx context.Context, facts Facts, fileName string, promptTemplate string) (string, error) {
	req, err := b.adapter.BuildGenerateSpecRequest(facts, fileName, promptTemplate)
	if err != nil {
		return "", err
	}

	respBytes, _, err := b.executeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	res, prompt, comp, err := b.adapter.ParseGenerateSpecResponse(respBytes)
	if err != nil {
		return "", err
	}
	if b.onTokenUsage != nil {
		b.onTokenUsage(prompt, comp)
	}
	return res, nil
}

func (b *BaseGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []domain.Standard) ([]ComplianceResult, error) {
	applicableStandards := config.FilterApplicableStandards(standards, fileName)
	if len(applicableStandards) == 0 {
		return nil, nil
	}

	req, err := b.adapter.BuildEvaluateComplianceRequest(fileName, fileContent, applicableStandards)
	if err != nil {
		return nil, err
	}

	respBytes, _, err := b.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	res, prompt, comp, err := b.adapter.ParseEvaluateComplianceResponse(respBytes)
	if err != nil {
		return nil, err
	}
	if b.onTokenUsage != nil {
		b.onTokenUsage(prompt, comp)
	}
	return res, nil
}

func (b *BaseGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (string, error) {
	req, err := b.adapter.BuildRefineSpecRequest(fileName, fileContent, feedback, failedStandards, referenceDoc)
	if err != nil {
		return "", err
	}

	respBytes, _, err := b.executeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	res, prompt, comp, err := b.adapter.ParseRefineSpecResponse(respBytes)
	if err != nil {
		return "", err
	}
	if b.onTokenUsage != nil {
		b.onTokenUsage(prompt, comp)
	}
	return res, nil
}

func (b *BaseGateway) VerifyConsistency(ctx context.Context, files map[string]string) (*ConsistencyReport, error) {
	req, err := b.adapter.BuildVerifyConsistencyRequest(files)
	if err != nil {
		return nil, err
	}

	respBytes, _, err := b.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	res, prompt, comp, err := b.adapter.ParseVerifyConsistencyResponse(respBytes)
	if err != nil {
		return nil, err
	}
	if b.onTokenUsage != nil {
		b.onTokenUsage(prompt, comp)
	}
	return res, nil
}

func (b *BaseGateway) Summarize(ctx context.Context, history []Message) (string, error) {
	req, err := b.adapter.BuildSummarizeRequest(history)
	if err != nil {
		return "", err
	}

	respBytes, _, err := b.executeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	res, prompt, comp, err := b.adapter.ParseSummarizeResponse(respBytes)
	if err != nil {
		return "", err
	}
	if b.onTokenUsage != nil {
		b.onTokenUsage(prompt, comp)
	}
	return res, nil
}

func (b *BaseGateway) ExtractStructuralEntities(ctx context.Context, sourceDoc string) (string, error) {
	req, err := b.adapter.BuildExtractStructuralEntitiesRequest(sourceDoc)
	if err != nil {
		return "", err
	}

	respBytes, _, err := b.executeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	res, prompt, comp, err := b.adapter.ParseGenerateSpecResponse(respBytes)
	if err != nil {
		return "", err
	}
	if b.onTokenUsage != nil {
		b.onTokenUsage(prompt, comp)
	}
	return res, nil
}

func (b *BaseGateway) OptimizePrompt(ctx context.Context, files map[string]string) (string, error) {
	req, err := b.adapter.BuildOptimizePromptRequest(files)
	if err != nil {
		return "", err
	}

	respBytes, _, err := b.executeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	res, prompt, comp, err := b.adapter.ParseGenerateSpecResponse(respBytes)
	if err != nil {
		return "", err
	}
	if b.onTokenUsage != nil {
		b.onTokenUsage(prompt, comp)
	}
	return res, nil
}

// GenerateEmbeddings calculates numeric vector embeddings for a slice of texts.
// It falls back to a deterministic local hashing-based embedding vector if the provider adapter doesn't support it.
func (b *BaseGateway) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	type EmbeddingAdapter interface {
		BuildEmbeddingsRequest(texts []string) (*http.Request, error)
		ParseEmbeddingsResponse(body []byte) ([][]float32, error)
	}

	if ea, ok := b.adapter.(EmbeddingAdapter); ok {
		req, err := ea.BuildEmbeddingsRequest(texts)
		if err == nil {
			respBytes, _, err := b.executeRequest(ctx, req)
			if err == nil {
				if embedRes, err := ea.ParseEmbeddingsResponse(respBytes); err == nil {
					return embedRes, nil
				}
			}
		}
	}

	// Local fallback: deterministic 128-dimensional pseudo-embeddings
	res := make([][]float32, len(texts))
	for i, txt := range texts {
		res[i] = pseudoEmbed(txt)
	}
	return res, nil
}

func pseudoEmbed(text string) []float32 {
	vec := make([]float32, 128)
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return vec
	}
	for i := 0; i < 128; i++ {
		var val float32
		for _, word := range words {
			// FNV-1a-like simple hash function
			h := uint32(2166136261)
			for j := 0; j < len(word); j++ {
				h = (h ^ uint32(word[j])) * 16777619
			}
			h = (h ^ uint32(i)) * 16777619
			val += float32(h) / float32(0xFFFFFFFF)
		}
		vec[i] = val / float32(len(words))
	}
	
	// Normalize
	var norm float64
	for _, v := range vec {
		norm += float64(v * v)
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}
	return vec
}

