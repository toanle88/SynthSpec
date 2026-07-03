package export

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/toanle/synthspec/generator"
)

// DocumentData holds a single specification document's name, title, and content
type DocumentData struct {
	FileName string `json:"file_name"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

// ExportMetadata holds metadata context for the export
type ExportMetadata struct {
	ProjectName string         `json:"project_name"`
	ExportTime  string         `json:"export_time"`
	Version     string         `json:"version"`
	Metrics     interface{}    `json:"metrics,omitempty"`
	Scores      map[string]int `json:"scores,omitempty"`
}

// telemetryMeta is a minimal representation of .synthspec-meta.json for export use
type telemetryMeta struct {
	ProjectName       string         `json:"project_name"`
	ComplianceSummary map[string]int `json:"compliance_summary,omitempty"`
	CompletionMetrics interface{}    `json:"completion_metrics,omitempty"`
}

//go:embed assets/export_template.html
var exportTemplate string

// ExportToHTML scans the output directory for markdown files and compiles them into a standalone index.html
func ExportToHTML(projectName string, outputDir string, distDir string) (string, error) {
	// 1. Locate files in the output directory
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return "", fmt.Errorf("output directory does not exist: %s", outputDir)
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to read output directory: %w", err)
	}

	var documents []DocumentData
	var metaData []byte

	// 2. Read each document and the metadata json
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(outputDir, file.Name())

		if file.Name() == ".synthspec-meta.json" {
			data, err := os.ReadFile(filePath)
			if err == nil {
				metaData = data
			}
			continue
		}

		if strings.HasSuffix(file.Name(), ".md") {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to read file %s: %w", file.Name(), err)
			}

			// Extract title from the first header line (e.g. "# Title")
			title := file.Name()
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "# ") {
					title = strings.TrimPrefix(trimmed, "# ")
					break
				}
			}

			documents = append(documents, DocumentData{
				FileName: file.Name(),
				Title:    title,
				Content:  string(content),
			})
		}
	}

	if len(documents) == 0 {
		return "", fmt.Errorf("no markdown specification documents found in %s", outputDir)
	}

	// Sort documents by file name
	sort.Slice(documents, func(i, j int) bool {
		return documents[i].FileName < documents[j].FileName
	})

	// 3. Compile metadata
	var tMeta telemetryMeta
	if len(metaData) > 0 {
		_ = json.Unmarshal(metaData, &tMeta)
	}

	exportMeta := ExportMetadata{
		ProjectName: projectName,
		ExportTime:  time.Now().Format("2006-01-02 15:04:05"),
		Version:     generator.EngineVersion,
	}
	if tMeta.ProjectName != "" {
		exportMeta.Scores = tMeta.ComplianceSummary
		exportMeta.Metrics = tMeta.CompletionMetrics
	}

	docsJSON, err := json.Marshal(documents)
	if err != nil {
		return "", fmt.Errorf("failed to marshal documents to JSON: %w", err)
	}

	metaJSON, err := json.Marshal(exportMeta)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}

	// 4. Generate HTML template
	tmpl, err := template.New("export").Parse(exportTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		ProjectName string
		DocsJSON    string
		MetaJSON    string
	}{
		ProjectName: projectName,
		DocsJSON:    string(docsJSON),
		MetaJSON:    string(metaJSON),
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	// 5. Ensure dist directory exists and write index.html
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	destPath := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(destPath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write export HTML file: %w", err)
	}

	return destPath, nil
}
