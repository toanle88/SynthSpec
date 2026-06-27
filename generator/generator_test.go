package generator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			name: "With json language backticks",
			input: "```json\n{\"epics\": []}\n```",
			expected: `{"epics": []}`,
		},
		{
			name: "With plain backticks",
			input: "```\n{\"epics\": []}\n```",
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
}

func (tg *TestGateway) QueryOracle(ctx context.Context, facts gateway.Facts, history []gateway.Message, latestInput string) (*gateway.OracleResponse, error) {
	tg.queryCount++
	return tg.queryResult, tg.queryErr
}

func (tg *TestGateway) GenerateSpecFile(ctx context.Context, facts gateway.Facts, fileName string) (string, error) {
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

func (tg *TestGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []config.Standard) (string, error) {
	tg.callCounts[fileName]++
	resps, ok := tg.responses[fileName]
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


func TestGenerateWithRetryAndValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-gen-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sess := &state.Session{
		ProjectName: "test-project",
		Provider:    "test-provider",
	}

	t.Run("All success first attempt", func(t *testing.T) {
		tg := &TestGateway{
			responses: map[string][]string{
				"05_engineering_backlog.json": {
					`{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "D1", "acceptance_criteria": ["AC"]}]}]}`,
				},
			},
			callCounts: make(map[string]int),
		}

		progress := make(chan string, 20)
		err := Generate(context.Background(), tg, sess, tempDir, progress)
		if err != nil {
			t.Fatalf("expected success, got err: %v", err)
		}

		// Drain progress
		for range progress {}

		// Verify files exist
		files := []string{
			"01_prd_functional.md",
			"02_system_architecture.md",
			"03_security_threat_model.md",
			"04_openapi_contract.yaml",
			"05_engineering_backlog.json",
			".synthspec-meta.json",
		}
		for _, f := range files {
			path := filepath.Join(tempDir, f)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected file %s to exist", f)
			}
		}

		if tg.callCounts["05_engineering_backlog.json"] != 1 {
			t.Errorf("expected 1 call, got %d", tg.callCounts["05_engineering_backlog.json"])
		}
	})

	t.Run("Transient API failure retry", func(t *testing.T) {
		sess.GeneratedFiles = nil
		os.RemoveAll(tempDir)
		os.MkdirAll(tempDir, 0755)
		tg := &TestGateway{
			responses: map[string][]string{
				"05_engineering_backlog.json": {
					"ERROR:timeout",
					`{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "D1", "acceptance_criteria": ["AC"]}]}]}`,
				},
			},
			callCounts: make(map[string]int),
		}

		progress := make(chan string, 20)
		err := Generate(context.Background(), tg, sess, tempDir, progress)
		if err != nil {
			t.Fatalf("expected success, got err: %v", err)
		}

		for range progress {}

		if tg.callCounts["05_engineering_backlog.json"] != 2 {
			t.Errorf("expected 2 calls (1 retry), got %d", tg.callCounts["05_engineering_backlog.json"])
		}
	})

	t.Run("Transient validation failure retry", func(t *testing.T) {
		sess.GeneratedFiles = nil
		os.RemoveAll(tempDir)
		os.MkdirAll(tempDir, 0755)
		tg := &TestGateway{
			responses: map[string][]string{
				"05_engineering_backlog.json": {
					`{"epics": []}`, // fails validation
					`{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "D1", "acceptance_criteria": ["AC"]}]}]}`,
				},
			},
			callCounts: make(map[string]int),
		}

		progress := make(chan string, 20)
		err := Generate(context.Background(), tg, sess, tempDir, progress)
		if err != nil {
			t.Fatalf("expected success, got err: %v", err)
		}

		for range progress {}

		if tg.callCounts["05_engineering_backlog.json"] != 2 {
			t.Errorf("expected 2 calls, got %d", tg.callCounts["05_engineering_backlog.json"])
		}
	})

	t.Run("Persistent failure", func(t *testing.T) {
		sess.GeneratedFiles = nil
		os.RemoveAll(tempDir)
		os.MkdirAll(tempDir, 0755)
		tg := &TestGateway{
			responses: map[string][]string{
				"05_engineering_backlog.json": {
					`{"epics": []}`,
					`{"epics": []}`,
					`{"epics": []}`,
				},
			},
			callCounts: make(map[string]int),
		}

		progress := make(chan string, 20)
		err := Generate(context.Background(), tg, sess, tempDir, progress)
		if err == nil {
			t.Fatal("expected failure, got success")
		}

		for range progress {}

		if tg.callCounts["05_engineering_backlog.json"] != 10 {
			t.Errorf("expected 10 calls, got %d", tg.callCounts["05_engineering_backlog.json"])
		}
	})

	t.Run("Resumable progress skip completed", func(t *testing.T) {
		sess.GeneratedFiles = nil
		os.RemoveAll(tempDir)
		os.MkdirAll(tempDir, 0755)

		// 1. Simulate a failure on the third file ("03_security_threat_model.md")
		tg1 := &TestGateway{
			responses: map[string][]string{
				"01_prd_functional.md":        {"PRD content"},
				"02_system_architecture.md":   {"Arch content"},
				"03_security_threat_model.md": {"ERROR:mocked_api_failure"},
			},
			callCounts: make(map[string]int),
		}

		progress1 := make(chan string, 20)
		err1 := Generate(context.Background(), tg1, sess, tempDir, progress1)
		if err1 == nil {
			t.Fatal("expected failure on 03_security_threat_model.md, got success")
		}
		for range progress1 {}

		// Verify first two files were generated and written, but not the third
		if len(sess.GeneratedFiles) != 2 {
			t.Errorf("expected 2 files in GeneratedFiles cache, got %d", len(sess.GeneratedFiles))
		}
		if sess.GeneratedFiles[0].FileName != "01_prd_functional.md" || sess.GeneratedFiles[1].FileName != "02_system_architecture.md" {
			t.Errorf("unexpected cached files list: %+v", sess.GeneratedFiles)
		}

		// 2. Resume with a healthy gateway
		tg2 := &TestGateway{
			responses: map[string][]string{
				"03_security_threat_model.md": {"Threat model content"},
				"04_openapi_contract.yaml":    {"openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}"},
				"05_engineering_backlog.json": {
					`{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "D1", "acceptance_criteria": ["AC"]}]}]}`,
				},
			},
			callCounts: make(map[string]int),
		}

		progress2 := make(chan string, 20)
		err2 := Generate(context.Background(), tg2, sess, tempDir, progress2)
		if err2 != nil {
			t.Fatalf("expected resumption success, got err: %v", err2)
		}
		for range progress2 {}

		// Verify skipping occurred: tg2 call count for first two files must be 0
		if tg2.callCounts["01_prd_functional.md"] != 0 {
			t.Errorf("expected 0 calls for 01_prd_functional.md on resume, got %d", tg2.callCounts["01_prd_functional.md"])
		}
		if tg2.callCounts["02_system_architecture.md"] != 0 {
			t.Errorf("expected 0 calls for 02_system_architecture.md on resume, got %d", tg2.callCounts["02_system_architecture.md"])
		}
		// Remaining files must have been generated
		if tg2.callCounts["03_security_threat_model.md"] != 1 {
			t.Errorf("expected 1 call for 03_security_threat_model.md, got %d", tg2.callCounts["03_security_threat_model.md"])
		}
	})
}
