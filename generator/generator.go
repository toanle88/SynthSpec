package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

// TelemetryMetadata represents the .synthspec-meta.json structure
type TelemetryMetadata struct {
	ProjectName         string            `json:"project_name"`
	GenerationTimestamp string            `json:"generation_timestamp"`
	EngineVersion       string            `json:"engine_version"`
	ProviderUsed        string            `json:"provider_used"`
	CompletionMetrics   CompletionMetrics `json:"completion_metrics"`
}

type CompletionMetrics struct {
	TotalTurns     int `json:"total_turns"`
	TokensConsumed int `json:"tokens_consumed"`
}

// Generate runs sequential spec generation for all files
func Generate(ctx context.Context, gw gateway.Gateway, sess *state.Session, outputDir string, progress chan<- string) error {
	defer close(progress)

	// Ensure output directory exists
	if outputDir == "" {
		outputDir = filepath.Join(state.GetSessionDir(sess.ProjectName), "output")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	files := []string{
		"01_prd_functional.md",
		"02_system_architecture.md",
		"03_security_threat_model.md",
		"04_openapi_contract.yaml",
		"05_engineering_backlog.json",
	}

	for _, fileName := range files {
		progress <- fmt.Sprintf("Synthesizing %s...", fileName)

		content, err := gw.GenerateSpecFile(ctx, sess.Facts, fileName)
		if err != nil {
			return fmt.Errorf("failed to generate %s: %w", fileName, err)
		}

		filePath := filepath.Join(outputDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s output file: %w", fileName, err)
		}
	}

	// Generate .synthspec-meta.json
	progress <- "Compiling solution metadata (.synthspec-meta.json)..."

	meta := TelemetryMetadata{
		ProjectName:         sess.ProjectName,
		GenerationTimestamp: time.Now().Format(time.RFC3339),
		EngineVersion:       "1.0.0",
		ProviderUsed:        sess.Provider,
		CompletionMetrics: CompletionMetrics{
			TotalTurns:     len(sess.History) / 2, // Pairs of user/assistant turns
			TokensConsumed: sess.TotalTokensUsed,
		},
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize telemetry metadata: %w", err)
	}

	metaPath := filepath.Join(outputDir, ".synthspec-meta.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("failed to write .synthspec-meta.json: %w", err)
	}

	progress <- fmt.Sprintf("All files generated in: %s", outputDir)
	return nil
}
