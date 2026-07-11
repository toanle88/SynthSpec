package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

// EngineVersion is the version of the generation engine, set at build time via ldflags.
var EngineVersion = "dev"

const maxRetries = 10

// GenerationMetadata represents the .synthspec-meta.json structure
type GenerationMetadata struct {
	ProjectName         string            `json:"project_name"`
	GenerationTimestamp string            `json:"generation_timestamp"`
	EngineVersion       string            `json:"engine_version"`
	ProviderUsed        string            `json:"provider_used"`
	CompletionMetrics   CompletionMetrics `json:"completion_metrics"`
	ComplianceSummary   map[string]int    `json:"compliance_summary,omitempty"`
}

type CompletionMetrics struct {
	TotalTurns     int    `json:"total_turns"`
	TokensConsumed int    `json:"tokens_consumed"`
	TotalDuration  string `json:"total_duration"`
}

// ProgressEvent represents a structured progress update sent to the TUI
type ProgressEvent struct {
	File    string `json:"file,omitempty"`
	Status  string `json:"status,omitempty"` // pending, skipped, synthesizing, correcting, auditing, refining, done, failed, started, completed, compiling_report, compiling_metadata
	Phase   string `json:"phase,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
	ValLogs string `json:"val_logs,omitempty"`
}

func sendProgress(progress chan<- string, event ProgressEvent) {
	if data, err := json.Marshal(event); err == nil {
		progress <- string(data)
	} else {
		progress <- event.Message
	}
}

// fileGenerator orchestrates the file generation pipeline
type fileGenerator struct {
	ctx              context.Context
	gw               gateway.Gateway
	persistence      SessionPersistence
	outputDir        string
	progress         chan<- string
	approvalChan     chan struct{}
	diffApprovalChan chan struct{}
	sourceFileName   string
	templates        []config.Template
	sessionMu        sync.Mutex
	proposedContents map[string]string
	proposedMu       sync.Mutex
	startTime        time.Time
	forceFinishChan  chan struct{}
}

// Generate runs sequential spec generation for all files
func Generate(ctx context.Context, gw gateway.Gateway, persistence SessionPersistence, outputDir string, progress chan<- string, approvalChan chan struct{}, diffApprovalChan chan struct{}, forceFinishChan chan struct{}) (genErr error) {
	startTime := time.Now()
	var fg *fileGenerator
	defer func() {
		elapsed := int64(time.Since(startTime).Seconds())
		_ = persistence.AddDuration(elapsed)
		if fg != nil && genErr != nil {
			_ = fg.writeProposedToDisk()
		}
	}()
	defer close(progress)

	// Load templates
	templates, err := config.LoadTemplates()
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Load quality standards configuration
	standards, err := config.LoadStandards()
	if err != nil {
		return fmt.Errorf("failed to load quality standards: %w", err)
	}

	// Ensure output directory exists
	if outputDir == "" {
		// We need to get project name from persistence - for now use a default
		// The caller should provide a proper outputDir
		outputDir = "output"
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var files []string
	for _, t := range templates {
		files = append(files, t.FileName)
	}

	sendProgress(progress, ProgressEvent{Status: "started", Phase: "source", Details: strings.Join(files, ","), Message: "Starting spec generation..."})

	fg = &fileGenerator{
		ctx:              ctx,
		gw:               gw,
		persistence:      persistence,
		outputDir:        outputDir,
		progress:         progress,
		approvalChan:     approvalChan,
		diffApprovalChan: diffApprovalChan,
		templates:        templates,
		proposedContents: make(map[string]string),
		startTime:        startTime,
		forceFinishChan:  forceFinishChan,
	}

	fileCompliances := make([]FileCompliance, len(templates))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	fg.ctx = runCtx

	sourceIdx, err := findSourceTemplate(templates)
	if err != nil {
		return err
	}
	fg.sourceFileName = templates[sourceIdx].FileName

	sourceCompliance, sourceDoc, _, err := fg.generateSourceDocument(templates[sourceIdx], standards)
	if err != nil {
		return err
	}
	fileCompliances[sourceIdx] = sourceCompliance

	sendProgress(progress, ProgressEvent{
		Status:  "started",
		Phase:   "parallel",
		Details: strings.Join(files, ","),
		Message: fmt.Sprintf("Source document locked. Starting parallel generation for %d downstream documents...", len(templates)-1),
	})

	if err := fg.generateDownstreamParallel(templates, sourceDoc, standards, fileCompliances); err != nil {
		return err
	}

	return fg.runConsistencyVerification(templates, standards, fileCompliances)
}
