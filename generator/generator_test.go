package generator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

func TestSanitizeJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No backticks",
			input:    `{"epics": []}`,
			expected: `{"epics": []}`,
		},
		{
			name:     "With json language backticks",
			input:    "```json\n{\"epics\": []}\n```",
			expected: `{"epics": []}`,
		},
		{
			name:     "With plain backticks",
			input:    "```\n{\"epics\": []}\n```",
			expected: `{"epics": []}`,
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeJSONOutput(tt.input)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestValidateBacklog(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errSub  string
	}{
		{
			name:    "Valid backlog",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{"epics": `,
			wantErr: true,
			errSub:  "invalid JSON syntax",
		},
		{
			name:    "Empty epics",
			input:   `{"epics": []}`,
			wantErr: true,
			errSub:  "backlog must contain at least one epic",
		},
		{
			name:    "Epic missing ID",
			input:   `{"epics": [{"title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "epic 0 is missing ID",
		},
		{
			name:    "Epic missing Title",
			input:   `{"epics": [{"id": "EP-1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "epic EP-1 is missing Title",
		},
		{
			name:    "Epic missing Description",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "epic EP-1 is missing Description",
		},
		{
			name:    "Epic missing tasks",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": []}]}`,
			wantErr: true,
			errSub:  "epic EP-1 must contain at least one task",
		},
		{
			name:    "Task missing ID",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "task 0 in epic EP-1 is missing ID",
		},
		{
			name:    "Task missing Summary",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "task TSK-1 in epic EP-1 is missing Summary",
		},
		{
			name:    "Task missing Details",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "task TSK-1 in epic EP-1 is missing Details",
		},
		{
			name:    "Task missing Acceptance Criteria",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": []}]}]}`,
			wantErr: true,
			errSub:  "task TSK-1 in epic EP-1 must contain at least one acceptance criterion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBacklog(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errSub) {
					t.Errorf("expected error containing %q, got %q", tt.errSub, err.Error())
				}
			}
		})
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
	// If a custom mock response for consistency is desired, we can add it, but default to consistent=true
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
	err = Generate(context.Background(), tg, sess, tempDir, progress)
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
			// Drain progress notifications
			continue
		}
	}()

	done := make(chan error, 1)
	go func() {
		done <- Generate(context.Background(), gateway, sess, tempDir, progress)
	}()

	seen := make(map[string]bool)
	deadline := time.After(5 * time.Second)
	for len(seen) < 2 {
		select {
		case fileName := <-started:
			seen[fileName] = true
		case <-deadline:
			t.Fatalf("expected two downstream files to start in parallel, saw: %v", seen)
		}
	}

	close(release)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("parallel generation failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("parallel generation did not finish")
	}
}

func TestGenerate_TransientAPIFailure(t *testing.T) {
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
				"ERROR:timeout",
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
	err = Generate(context.Background(), tg, sess, tempDir, progress)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	if tg.callCounts["05_coding_standards_guidelines.md"] != 2 {
		t.Errorf("expected 2 calls (1 retry), got %d", tg.callCounts["05_coding_standards_guidelines.md"])
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
	err = Generate(context.Background(), tg, sess, tempDir, progress)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	if tg.callCounts["05_coding_standards_guidelines.md"] != 2 {
		t.Errorf("expected 2 calls, got %d", tg.callCounts["05_coding_standards_guidelines.md"])
	}
}

func TestGenerate_PersistentFailure(t *testing.T) {
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
				`   `,
				`   `,
				`   `,
				`   `,
				`   `,
				`   `,
				`   `,
				`   `,
				`   `,
				`   `,
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
	err = Generate(context.Background(), tg, sess, tempDir, progress)
	if err == nil {
		t.Fatal("expected failure, got success")
	}

	if tg.callCounts["05_coding_standards_guidelines.md"] != 10 {
		t.Errorf("expected 10 calls, got %d", tg.callCounts["05_coding_standards_guidelines.md"])
	}
}

func TestGenerate_ResumableProgressSkipCompleted(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-gen-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-project",
		Provider:    "test-provider",
	}

	t.Run("SimulateFailure", func(t *testing.T) {
		runSimulateFailureSubtest(t, sess, tempDir)
	})

	t.Run("ResumeHealthy", func(t *testing.T) {
		runResumeHealthySubtest(t, sess, tempDir)
	})
}

func runSimulateFailureSubtest(t *testing.T, sess *state.Session, tempDir string) {
	tg1 := &TestGateway{
		responses: map[string][]string{
			"01_domain_model_use_cases.md": {"Domain content"},
			"02_prd_functional.md":         {"PRD content"},
			"03_system_architecture.md":    {"ERROR:mocked_api_failure"},
		},
		callCounts: make(map[string]int),
	}

	progress1 := make(chan string, 20)
	go func() {
		for range progress1 {
			continue
		}
	}()
	err1 := Generate(context.Background(), tg1, sess, tempDir, progress1)
	if err1 == nil {
		t.Fatal("expected failure on 03_system_architecture.md, got success")
	}

	// Verify the failed file was not cached, while the completed siblings were preserved.
	if len(sess.GeneratedFiles) != 6 {
		t.Errorf("expected 6 cached files after a downstream failure, got %d", len(sess.GeneratedFiles))
	}
	cachedFiles := make(map[string]bool)
	for _, gf := range sess.GeneratedFiles {
		cachedFiles[gf.FileName] = true
	}
	if !cachedFiles["01_domain_model_use_cases.md"] || !cachedFiles["02_prd_functional.md"] {
		t.Errorf("expected the source doc and PRD to remain cached, got: %+v", sess.GeneratedFiles)
	}
	if cachedFiles["03_system_architecture.md"] {
		t.Errorf("expected failed file 03_system_architecture.md to remain uncached, got: %+v", sess.GeneratedFiles)
	}
}

func runResumeHealthySubtest(t *testing.T, sess *state.Session, tempDir string) {
	tg2 := &TestGateway{
		responses: map[string][]string{
			"03_system_architecture.md":          {"Arch content"},
			"04_api_architecture_integration.md": {"# API Integration Guide"},
			"05_coding_standards_guidelines.md":  {"# Coding Guidelines"},
			"06_security_threat_model.md":        {"Threat model content"},
			"07_engineering_roadmap.md":          {"Roadmap content"},
		},
		callCounts: make(map[string]int),
	}

	progress2 := make(chan string, 20)
	go func() {
		for range progress2 {
			continue
		}
	}()
	err2 := Generate(context.Background(), tg2, sess, tempDir, progress2)
	if err2 != nil {
		t.Fatalf("expected resumption success, got err: %v", err2)
	}

	// Verify skipping occurred for the completed files.
	skippedFiles := []string{
		"01_domain_model_use_cases.md",
		"02_prd_functional.md",
		"04_api_architecture_integration.md",
		"05_coding_standards_guidelines.md",
		"06_security_threat_model.md",
		"07_engineering_roadmap.md",
	}
	for _, file := range skippedFiles {
		if tg2.callCounts[file] != 0 {
			t.Errorf("expected 0 calls for %s on resume, got %d", file, tg2.callCounts[file])
		}
	}
	// The failed file must be regenerated on resume.
	if tg2.callCounts["03_system_architecture.md"] != 1 {
		t.Errorf("expected 1 call for 03_system_architecture.md, got %d", tg2.callCounts["03_system_architecture.md"])
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
	err = Generate(context.Background(), tg, sess, tempDir, progress)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	// Verify that GenerateSpecFile was NOT called for 01_domain_model_use_cases.md because we resumed (it goes straight to refinement/validation)
	if tg.callCounts["01_domain_model_use_cases.md"] != 0 {
		t.Errorf("expected 0 calls to GenerateSpecFile/RefineSpecFile for resumed file, got %d", tg.callCounts["01_domain_model_use_cases.md"])
	}
}

func TestRunExternalValidator(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-val-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	ctx := context.Background()

	// Test success command
	successCmd := "echo success"

	out, err := runExternalValidator(ctx, successCmd, tempFile.Name())
	if err != nil {
		t.Errorf("expected no error, got: %v, output: %q", err, out)
	}
	if !strings.Contains(out, "success") {
		t.Errorf("expected output to contain 'success', got %q", out)
	}

	// Test failing command
	var failCmd string
	if runtime.GOOS == "windows" {
		failCmd = "type non_existent_file_12345.txt"
	} else {
		failCmd = "cat non_existent_file_12345.txt"
	}

	_, err = runExternalValidator(ctx, failCmd, tempFile.Name())
	if err == nil {
		t.Error("expected error for failing command, got nil")
	}
}

func TestBuildGenerationPromptIncludesReferenceDocument(t *testing.T) {
	facts := gateway.Facts{
		Functional: "Functional facts",
		Structural: "Structural facts",
	}

	prompt, err := buildGenerationPrompt("Write the file.\n\nUse these facts:", facts, "Domain model reference")
	if err != nil {
		t.Fatalf("failed to build generation prompt: %v", err)
	}

	if !strings.Contains(prompt, "\"functional\": \"Functional facts\"") {
		t.Fatalf("expected prompt to include serialized facts, got: %s", prompt)
	}

	if !strings.Contains(prompt, "Reference source document:") {
		t.Fatalf("expected prompt to include reference document marker, got: %s", prompt)
	}

	if !strings.Contains(prompt, "Domain model reference") {
		t.Fatalf("expected prompt to include reference document content, got: %s", prompt)
	}

	if strings.Index(prompt, "Reference source document:") < strings.Index(prompt, "\"functional\": \"Functional facts\"") {
		t.Fatalf("expected reference document to appear after facts, got: %s", prompt)
	}
}

func TestGenerate_ConsistencyCheckAndSelfCorrection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-gen-consistency-test")
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

	err = Generate(context.Background(), tg, sess, tempDir, progress)
	if err != nil {
		t.Fatalf("expected generation success after consistency refinement, got: %v", err)
	}

	// RefineSpecFile should have been called for 02_prd_functional.md to fix the TRIGGER_INCONSISTENCY.
	// Since tg.callCounts tracks calls in RefineSpecFile and GenerateSpecFile:
	// 1 GenerateSpecFile call + 1 RefineSpecFile call = 2 calls.
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
	err = Generate(context.Background(), tg, sess, tempDir, progress)
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
	err = Generate(context.Background(), tg, sess, tempDir, progress2)
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
	err = Generate(context.Background(), tg, sess, tempDir, progress3)
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

