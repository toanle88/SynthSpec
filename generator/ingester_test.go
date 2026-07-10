package generator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIngester(t *testing.T) {
	tempDir := t.TempDir()

	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	file1 := filepath.Join(srcDir, "test1.txt")
	if err := os.WriteFile(file1, []byte("This is some sample text for ingestion testing. Go RAG embedding pipeline is active!"), 0644); err != nil {
		t.Fatal(err)
	}

	file2 := filepath.Join(srcDir, "test2.md")
	if err := os.WriteFile(file2, []byte("# Markdown Header\nMore sample content for indexing codebase elements."), 0644); err != nil {
		t.Fatal(err)
	}

	kbFile := filepath.Join(tempDir, "kb.json")
	tg := &TestGateway{}
	ing := NewIngester(tg)

	count, err := ing.IngestDirectory(context.Background(), srcDir, kbFile)
	if err != nil {
		t.Fatalf("IngestDirectory failed: %v", err)
	}

	if count == 0 {
		t.Fatal("expected at least one chunk to be ingested")
	}

	// Verify KB file was written
	if _, err := os.Stat(kbFile); os.IsNotExist(err) {
		t.Fatalf("expected KB file to exist at %s", kbFile)
	}

	// Read KB file back
	data, err := os.ReadFile(kbFile)
	if err != nil {
		t.Fatal(err)
	}

	type VectorNode struct {
		Text     string    `json:"text"`
		FilePath string    `json:"file_path"`
		Vector   []float32 `json:"vector"`
	}

	var nodes []VectorNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		t.Fatalf("failed to parse KB JSON: %v", err)
	}

	if len(nodes) != count {
		t.Errorf("expected %d nodes in JSON, got %d", count, len(nodes))
	}

	for _, node := range nodes {
		if node.Text == "" {
			t.Error("expected node text to be populated")
		}
		if node.FilePath == "" {
			t.Error("expected node file path to be populated")
		}
		if len(node.Vector) != 128 {
			t.Errorf("expected 128-dimensional embedding vector, got %d", len(node.Vector))
		}
	}
}
