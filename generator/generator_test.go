package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerate_AllSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-gen-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	persistence := NewMockPersistence()

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
	err = Generate(context.Background(), tg, persistence, tempDir, progress, nil, nil)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	files := []string{
		"01_domain_model_use_cases.md",
		"02_prd_functional.md",
		"03_system_architecture.md",
		"04_api_architecture_integration.md",
		"05_coding_standards_guidelines.md",
		"06_security_threat_model.md",
		"07_engineering_roadmap.md",
		".synthspec-meta.json",
		".synthspec-entities.json",
		"99_optimized_prompt.md",
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

	persistence := NewMockPersistence()

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
		genErr = Generate(context.Background(), gateway, persistence, tempDir, progress, nil, nil)
		close(done)
	}()

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

	persistence := NewMockPersistence()

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
	err = Generate(context.Background(), tg, persistence, tempDir, progress, nil, nil)
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

	persistence := NewMockPersistence()

	tg := &TestGateway{
		responses: map[string][]string{
			"05_coding_standards_guidelines.md": {
				`   `,
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
	err = Generate(context.Background(), tg, persistence, tempDir, progress, nil, nil)
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

	persistence := NewMockPersistence()

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
	err = Generate(context.Background(), tg, persistence, tempDir, progress, nil, nil)
	if err == nil {
		t.Fatal("expected persistent failure error, got nil")
	}

	if tg.callCounts["01_domain_model_use_cases.md"] != 10 {
		t.Errorf("expected 10 retries, got %d", tg.callCounts["01_domain_model_use_cases.md"])
	}
}
