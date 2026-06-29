package generator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/shared"
	"github.com/toanle/synthspec/state"
)

func TestSendProgress(t *testing.T) {
	ch := make(chan string, 1)
	sendProgress(ch, ProgressEvent{File: "test.md", Status: "done", Message: "Done"})
	msg := <-ch
	if !strings.Contains(msg, "test.md") || !strings.Contains(msg, "done") {
		t.Errorf("expected progress event JSON containing test.md and done, got: %s", msg)
	}
}

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
	shared.StreamOracleResponse(res, tokenChan)
	return res, nil
}

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
	if strings.HasPrefix(resp, "ERROR:") {
		return "", errors.New(strings.TrimPrefix(resp, "ERROR:"))
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
	if strings.HasPrefix(resp, "ERROR:") {
		return "", errors.New(strings.TrimPrefix(resp, "ERROR:"))
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

func TestGenerate_AllSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-gen-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-project",
		Provider:    "test-provider",
	}

	tg := &TestGateway{
		responses: map[string][]string{
			"05_coding_standards_guidelines.md": {
				`# Coding Guidelines`,
			},
		},
		callCounts: make(map[string]int),
	}

	progress := make(chan string, 20)
	go func() {
		for range progress {
			continue
		}
	}()
	err = Generate(context.Background(), tg, sess, tempDir, progress, nil)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	// Verify files exist
	files := []string{
		"01_domain_model_use_cases.md",
		"02_prd_functional.md",
		"03_system_architecture.md",
		"04_api_architecture_integration.md",
		"05_coding_standards_guidelines.md",
		"06_security_threat_model.md",
		"07_engineering_roadmap.md",
		".synthspec-meta.json",
	}
	for _, f := range files {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	if tg.callCounts["05_coding_standards_guidelines.md"] != 1 {
		t.Errorf("expected 1 call, got %d", tg.callCounts["05_coding_standards_guidelines.md"])
	}
}

func TestGenerate_DownstreamFilesRunInParallel(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-parallel-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-project",
		Provider:    "test-provider",
	}

	release := make(chan struct{})
	started := make(chan string, 2)
	gateway := &blockingGateway{
		TestGateway: &TestGateway{
			responses: map[string][]string{
				"01_domain_model_use_cases.md":       {"Domain content"},
				"02_prd_functional.md":               {"PRD content"},
				"03_system_architecture.md":          {"Architecture content"},
				"04_api_architecture_integration.md": {"API content"},
				"05_coding_standards_guidelines.md":  {"Coding content"},
				"06_security_threat_model.md":        {"Security content"},
				"07_engineering_roadmap.md":          {"Roadmap content"},
			},
			callCounts: make(map[string]int),
		},
		started: started,
		release: release,
		blocked: map[string]bool{
			"02_prd_functional.md":      true,
			"03_system_architecture.md": true,
		},
	}

	progress := make(chan string, 50)
	go func() {
		for range progress {
			continue
		}
	}()

	var genErr error
	done := make(chan struct{})
	go func() {
		genErr = Generate(context.Background(), gateway, sess, tempDir, progress, nil)
		close(done)
	}()

	// Wait for two downstream files to start in parallel
	select {
	case file1 := <-started:
		select {
		case file2 := <-started:
			t.Logf("successfully saw %s and %s start in parallel", file1, file2)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for second file to start generating in parallel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first file to start generating")
	}

	// Release block and wait for completion
	close(release)
	<-done

	if genErr != nil {
		t.Fatalf("expected generation to succeed, got: %v", genErr)
	}
}

func TestGenerate_TransientAPIFailure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-transient-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-project",
		Provider:    "test-provider",
	}

	tg := &TestGateway{
		responses: map[string][]string{
			"01_domain_model_use_cases.md": {
				"ERROR:Transient API Error",
				"ERROR:Transient API Error 2",
				"Domain Model Content Successful",
			},
		},
		callCounts: make(map[string]int),
	}

	progress := make(chan string, 50)
	go func() {
		for range progress {
			continue
		}
	}()
	err = Generate(context.Background(), tg, sess, tempDir, progress, nil)
	if err != nil {
		t.Fatalf("expected success on transient retry, got: %v", err)
	}

	if tg.callCounts["01_domain_model_use_cases.md"] != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", tg.callCounts["01_domain_model_use_cases.md"])
	}
}

func TestGenerate_TransientValidationFailure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-gen-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-project",
		Provider:    "test-provider",
	}

	tg := &TestGateway{
		responses: map[string][]string{
			"05_coding_standards_guidelines.md": {
				`   `, // fails validation (empty/whitespace)
				`# Coding Guidelines`,
			},
		},
		callCounts: make(map[string]int),
	}

	progress := make(chan string, 20)
	go func() {
		for range progress {
			continue
		}
	}()
	err = Generate(context.Background(), tg, sess, tempDir, progress, nil)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	if tg.callCounts["05_coding_standards_guidelines.md"] != 2 {
		t.Errorf("expected 2 calls, got %d", tg.callCounts["05_coding_standards_guidelines.md"])
	}
}

func TestGenerate_PersistentFailure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-persistent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-project",
		Provider:    "test-provider",
	}

	tg := &TestGateway{
		responses: map[string][]string{
			"01_domain_model_use_cases.md": {
				"ERROR:Persistent Error",
			},
		},
		callCounts: make(map[string]int),
	}

	progress := make(chan string, 100)
	go func() {
		for range progress {
			continue
		}
	}()
	err = Generate(context.Background(), tg, sess, tempDir, progress, nil)
	if err == nil {
		t.Fatal("expected persistent failure error, got nil")
	}

	if tg.callCounts["01_domain_model_use_cases.md"] != 10 {
		t.Errorf("expected 10 retries, got %d", tg.callCounts["01_domain_model_use_cases.md"])
	}
}

func TestGenerate_ResumableProgressSkipCompleted(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-resumable-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Pre-create output files that are completed
	completedFiles := []string{
		"01_domain_model_use_cases.md",
		"02_prd_functional.md",
		"03_system_architecture.md",
	}
	for _, f := range completedFiles {
		err := os.WriteFile(filepath.Join(tempDir, f), []byte("Pre-existing successful content"), 0644)
		if err != nil {
			t.Fatalf("failed to create dummy file: %v", err)
		}
	}

	templates, err := config.LoadTemplates()
	if err != nil {
		t.Fatalf("failed to load templates: %v", err)
	}

	// Compute hashes to match pre-existing conditions
	currentPromptHash := ""
	for _, t := range templates {
		if t.FileName == "01_domain_model_use_cases.md" {
			currentPromptHash = computeSha256(t.Prompt)
		}
	}

	sess := &state.Session{
		ProjectName: "test-resumable-project",
		Provider:    "test-provider",
		GeneratedFiles: []state.GeneratedFileState{
			{FileName: "01_domain_model_use_cases.md", HasError: false, PromptHash: currentPromptHash},
			{FileName: "02_prd_functional.md", HasError: false, PromptHash: computeSha256(templates[1].Prompt)},
			{FileName: "03_system_architecture.md", HasError: false, PromptHash: computeSha256(templates[2].Prompt)},
		},
	}

	// Set facts hashes to match current hash
	factsBytes, _ := json.Marshal(sess.Facts)
	currentFactsHash := computeSha256(string(factsBytes))
	for idx := range sess.GeneratedFiles {
		if sess.GeneratedFiles[idx].FileName != "01_domain_model_use_cases.md" {
			sourcePath := filepath.Join(tempDir, "01_domain_model_use_cases.md")
			sourceBytes, _ := os.ReadFile(sourcePath)
			sess.GeneratedFiles[idx].FactsHash = computeSha256(currentFactsHash + computeSha256(string(sourceBytes)))
		} else {
			sess.GeneratedFiles[idx].FactsHash = currentFactsHash
		}
	}

	tg := &TestGateway{
		callCounts: make(map[string]int),
	}

	progress := make(chan string, 100)
	go func() {
		for range progress {
			continue
		}
	}()
	err = Generate(context.Background(), tg, sess, tempDir, progress, nil)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	// Gateway should NOT be called for the first three files
	for _, f := range completedFiles {
		if tg.callCounts[f] > 0 {
			t.Errorf("expected 0 gateway calls for completed file %s, got %d", f, tg.callCounts[f])
		}
	}

	// Gateway SHOULD be called for the remaining files
	remainingFiles := []string{
		"04_api_architecture_integration.md",
		"05_coding_standards_guidelines.md",
		"06_security_threat_model.md",
		"07_engineering_roadmap.md",
	}
	for _, f := range remainingFiles {
		if tg.callCounts[f] == 0 {
			t.Errorf("expected gateway calls for remaining file %s, got 0", f)
		}
	}
}

func TestResumableMidLoop(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-gen-resumable-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-resumable-project",
		Provider:    "test-provider",
		GeneratedFiles: []state.GeneratedFileState{
			{
				FileName:       "01_domain_model_use_cases.md",
				InProgressText: "In-progress draft of PRD",
				CurrentAttempt: 5,
				HasError:       true,
			},
		},
	}

	tg := &TestGateway{
		responses: map[string][]string{
			"01_domain_model_use_cases.md":       {"Domain content refined"},
			"02_prd_functional.md":               {"PRD content"},
			"03_system_architecture.md":          {"Arch content"},
			"04_api_architecture_integration.md": {"# API Integration Guide"},
			"05_coding_standards_guidelines.md":  {"# Coding Guidelines"},
			"06_security_threat_model.md":        {"Threat model content"},
			"07_engineering_roadmap.md":          {"Roadmap content"},
		},
		callCounts: make(map[string]int),
	}

	progress := make(chan string, 20)
	go func() {
		for range progress {
			continue
		}
	}()
	err = Generate(context.Background(), tg, sess, tempDir, progress, nil)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	// Verify that GenerateSpecFile was NOT called for 01_domain_model_use_cases.md because we resumed
	if tg.callCounts["01_domain_model_use_cases.md"] != 0 {
		t.Errorf("expected 0 calls to GenerateSpecFile/RefineSpecFile for resumed file, got %d", tg.callCounts["01_domain_model_use_cases.md"])
	}
}

func TestGenerate_ConsistencyCheckAndSelfCorrection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-consistency-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-consistency-project",
		Provider:    "test-provider",
	}

	tg := &TestGateway{
		responses: map[string][]string{
			"02_prd_functional.md": {
				"Functional Requirements - TRIGGER_INCONSISTENCY",
				"Functional Requirements - refined Fix: compliant",
			},
		},
		callCounts: make(map[string]int),
	}

	progress := make(chan string, 20)
	go func() {
		for range progress {
			continue
		}
	}()

	err = Generate(context.Background(), tg, sess, tempDir, progress, nil)
	if err != nil {
		t.Fatalf("expected generation success after consistency refinement, got: %v", err)
	}

	// RefineSpecFile should have been called for 02_prd_functional.md to fix the TRIGGER_INCONSISTENCY.
	if tg.callCounts["02_prd_functional.md"] != 2 {
		t.Errorf("expected 2 calls for 02_prd_functional.md, got %d", tg.callCounts["02_prd_functional.md"])
	}
}

func TestGenerate_DiffBasedCaching(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-cache-project",
		Provider:    "test-provider",
		Facts: gateway.Facts{
			Functional: "Functional Facts v1",
		},
	}

	tg := &TestGateway{
		responses: map[string][]string{
			"01_domain_model_use_cases.md":       {"Domain v1"},
			"02_prd_functional.md":               {"PRD v1"},
			"03_system_architecture.md":          {"System Arch v1"},
			"04_api_architecture_integration.md": {"API Integration v1"},
			"05_coding_standards_guidelines.md":  {"Coding Guidelines v1"},
			"06_security_threat_model.md":        {"Threat Model v1"},
			"07_engineering_roadmap.md":          {"Roadmap v1"},
		},
		callCounts: make(map[string]int),
	}

	progress := make(chan string, 100)
	go func() {
		for range progress {
			continue
		}
	}()

	// 1. Initial Generation
	err = Generate(context.Background(), tg, sess, tempDir, progress, nil)
	if err != nil {
		t.Fatalf("initial generation failed: %v", err)
	}

	// 2. Run again with no changes -> expect skipped status and no calls to gateway
	tg.callCounts = make(map[string]int)
	progress2 := make(chan string, 100)
	go func() {
		for range progress2 {
			continue
		}
	}()
	err = Generate(context.Background(), tg, sess, tempDir, progress2, nil)
	if err != nil {
		t.Fatalf("second generation failed: %v", err)
	}

	for fileName, count := range tg.callCounts {
		if count > 0 {
			t.Errorf("expected 0 calls for %s (cached), got %d", fileName, count)
		}
	}

	// 3. Modify facts -> expect file to be regenerated (call count > 0)
	sess.Facts.Functional = "Functional Facts v2 (Modified)"
	tg.callCounts = make(map[string]int)
	progress3 := make(chan string, 100)
	go func() {
		for range progress3 {
			continue
		}
	}()
	err = Generate(context.Background(), tg, sess, tempDir, progress3, nil)
	if err != nil {
		t.Fatalf("generation after facts modification failed: %v", err)
	}

	totalCalls := 0
	for _, count := range tg.callCounts {
		totalCalls += count
	}
	if totalCalls == 0 {
		t.Error("expected files to be regenerated after facts modification, but got 0 calls")
	}
}
