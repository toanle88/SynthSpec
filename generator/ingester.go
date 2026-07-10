package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/gateway"
)

// Ingester handles scanning local codebases and documents, chunking text, and saving embeddings
type Ingester struct {
	gw gateway.Gateway
}

func NewIngester(gw gateway.Gateway) *Ingester {
	return &Ingester{gw: gw}
}

// IngestDirectory walks the directory, chunks text files, generates embeddings, and saves them to targetKBPath
func (ing *Ingester) IngestDirectory(ctx context.Context, dirPath string, targetKBPath string) (int, error) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("source directory does not exist: %s", dirPath)
	}

	var textChunks []string
	var chunkMetadata []string // absolute filepaths or relative

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip hidden directories (like .git, .synthspec)
			if strings.HasPrefix(d.Name(), ".") && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		// Only process typical text/source files
		ext := strings.ToLower(filepath.Ext(path))
		isText := false
		textExts := []string{".go", ".py", ".js", ".ts", ".html", ".css", ".md", ".json", ".yaml", ".yml", ".txt", ".sql", ".sh"}
		for _, e := range textExts {
			if ext == e {
				isText = true
				break
			}
		}

		if !isText {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil // skip unreadable files
		}

		// Simple line-based/character chunking (approx 1000 characters per chunk with overlapping)
		fileText := string(content)
		chunks := chunkText(fileText, 1000, 200)
		for _, chunk := range chunks {
			if trimmed := strings.TrimSpace(chunk); len(trimmed) > 0 {
				textChunks = append(textChunks, trimmed)
				chunkMetadata = append(chunkMetadata, path)
			}
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed walking directory: %w", err)
	}

	if len(textChunks) == 0 {
		return 0, nil
	}

	// Generate embeddings in batches of 16
	var nodes []domain.VectorNode
	batchSize := 16
	for i := 0; i < len(textChunks); i += batchSize {
		end := i + batchSize
		if end > len(textChunks) {
			end = len(textChunks)
		}

		batchTexts := textChunks[i:end]
		vectors, err := ing.gw.GenerateEmbeddings(ctx, batchTexts)
		if err != nil {
			return 0, fmt.Errorf("failed generating embeddings: %w", err)
		}

		for j, vec := range vectors {
			nodes = append(nodes, domain.VectorNode{
				Text:     batchTexts[j],
				FilePath: chunkMetadata[i+j],
				Vector:   vec,
			})
		}
	}

	// Save to JSON
	dir := filepath.Dir(targetKBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create KB target directory: %w", err)
	}

	fileBytes, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed parsing KB nodes: %w", err)
	}

	if err := os.WriteFile(targetKBPath, fileBytes, 0644); err != nil {
		return 0, fmt.Errorf("failed writing KB to disk: %w", err)
	}

	return len(nodes), nil
}

func chunkText(text string, size int, overlap int) []string {
	if len(text) <= size {
		return []string{text}
	}

	var chunks []string
	start := 0
	for start < len(text) {
		end := start + size
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[start:end])
		start += (size - overlap)
		if start >= len(text) || size <= overlap {
			break
		}
	}
	return chunks
}
