package export

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type excalElement struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	X               float64  `json:"x"`
	Y               float64  `json:"y"`
	Width           float64  `json:"width"`
	Height          float64  `json:"height"`
	StrokeColor     string   `json:"strokeColor"`
	BackgroundColor string   `json:"backgroundColor"`
	FillStyle       string   `json:"fillStyle"`
	StrokeWidth     float64  `json:"strokeWidth"`
	StrokeStyle     string   `json:"strokeStyle"`
	Roughness       float64  `json:"roughness"`
	Opacity         float64  `json:"opacity"`
	GroupIds        []string `json:"groupIds"`
	Roundness       *int     `json:"roundness"`
	Seed            int      `json:"seed"`
	Version         int      `json:"version"`
	VersionNonce    int      `json:"versionNonce"`
	IsDeleted       bool     `json:"isDeleted"`
	BoundElements   []string `json:"boundElements,omitempty"`
	Updated         int64    `json:"updated"`
	Link            string   `json:"link,omitempty"`
	Locked          bool     `json:"locked"`
	Text            string   `json:"text,omitempty"`
	FontSize        float64  `json:"fontSize,omitempty"`
	FontFamily      int      `json:"fontFamily,omitempty"`
	TextAlign       string   `json:"textAlign,omitempty"`
	VerticalAlign   string   `json:"verticalAlign,omitempty"`
	Baseline        float64  `json:"baseline,omitempty"`
	ContainerId     string   `json:"containerId,omitempty"`
}

type excalFile struct {
	Type     string         `json:"type"`
	Version  int            `json:"version"`
	Source   string         `json:"source"`
	Elements []excalElement `json:"elements"`
	AppState map[string]any `json:"appState"`
	Files    map[string]any `json:"files"`
}

type parsedEntity struct {
	Name       string   `json:"name"`
	Attributes []string `json:"attributes"`
}

type parsedWorkflow struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps"`
}

type parsedIntegration struct {
	Type    string `json:"type"`
	Details string `json:"details"`
}

type denseEntities struct {
	Entities     []parsedEntity      `json:"entities"`
	Workflows    []parsedWorkflow    `json:"workflows"`
	Integrations []parsedIntegration `json:"integrations"`
}

// ExportToExcalidraw reads entities metadata and writes a .excalidraw JSON file
func ExportToExcalidraw(projectName string, outputDir string, distDir string) (string, error) {
	entitiesPath := filepath.Join(outputDir, ".synthspec-entities.json")
	var data denseEntities

	if _, err := os.Stat(entitiesPath); err == nil {
		content, readErr := os.ReadFile(entitiesPath)
		if readErr == nil {
			_ = json.Unmarshal(content, &data)
		}
	}

	elements := []excalElement{}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	// Create elements layout
	// Entities in Column 1 (x: 100)
	// Workflows in Column 2 (x: 450)
	// Integrations in Column 3 (x: 800)
	
	yOffset := 100.0
	xBase := 100.0

	// Draw Entities
	for _, ent := range data.Entities {
		idRect := fmt.Sprintf("rect_%d", rnd.Int())
		idText := fmt.Sprintf("text_%d", rnd.Int())

		textLines := []string{ent.Name, "---"}
		for _, attr := range ent.Attributes {
			textLines = append(textLines, fmt.Sprintf("- %s", attr))
		}
		textVal := strings.Join(textLines, "\n")
		height := float64(40 + len(ent.Attributes)*20)

		rect := excalElement{
			ID:              idRect,
			Type:            "rectangle",
			X:               xBase,
			Y:               yOffset,
			Width:           250,
			Height:          height,
			StrokeColor:     "#1e293b",
			BackgroundColor: "#eff6ff",
			FillStyle:       "solid",
			StrokeWidth:     1.5,
			StrokeStyle:     "solid",
			Roughness:       0,
			Opacity:         100,
			Seed:            rnd.Int(),
			Version:         1,
			Updated:         time.Now().Unix(),
		}

		text := excalElement{
			ID:            idText,
			Type:          "text",
			X:             xBase + 15,
			Y:             yOffset + 15,
			Width:         220,
			Height:        height - 30,
			StrokeColor:   "#0f172a",
			StrokeWidth:   1,
			StrokeStyle:   "solid",
			Roughness:     0,
			Opacity:       100,
			Seed:          rnd.Int(),
			Version:       1,
			Updated:       time.Now().Unix(),
			Text:          textVal,
			FontSize:      16,
			FontFamily:    1, // Helvetica
			TextAlign:     "left",
			VerticalAlign: "top",
		}

		elements = append(elements, rect, text)
		yOffset += height + 40
	}

	// Draw Workflows
	yOffset = 100.0
	xBase = 450.0
	for _, wf := range data.Workflows {
		idRect := fmt.Sprintf("rect_%d", rnd.Int())
		idText := fmt.Sprintf("text_%d", rnd.Int())

		textLines := []string{wf.Name, "---"}
		for idx, step := range wf.Steps {
			textLines = append(textLines, fmt.Sprintf("%d. %s", idx+1, step))
		}
		textVal := strings.Join(textLines, "\n")
		height := float64(40 + len(wf.Steps)*22)

		rect := excalElement{
			ID:              idRect,
			Type:            "rectangle",
			X:               xBase,
			Y:               yOffset,
			Width:           300,
			Height:          height,
			StrokeColor:     "#1e293b",
			BackgroundColor: "#f0fdf4",
			FillStyle:       "solid",
			StrokeWidth:     1.5,
			StrokeStyle:     "solid",
			Roughness:       0,
			Opacity:         100,
			Seed:            rnd.Int(),
			Version:         1,
			Updated:         time.Now().Unix(),
		}

		text := excalElement{
			ID:            idText,
			Type:          "text",
			X:             xBase + 15,
			Y:             yOffset + 15,
			Width:         270,
			Height:        height - 30,
			StrokeColor:   "#0f172a",
			StrokeWidth:   1,
			StrokeStyle:   "solid",
			Roughness:     0,
			Opacity:       100,
			Seed:          rnd.Int(),
			Version:       1,
			Updated:       time.Now().Unix(),
			Text:          textVal,
			FontSize:      16,
			FontFamily:    1,
			TextAlign:     "left",
			VerticalAlign: "top",
		}

		elements = append(elements, rect, text)
		yOffset += height + 40
	}

	// Draw Integrations
	yOffset = 100.0
	xBase = 850.0
	for _, integration := range data.Integrations {
		idRect := fmt.Sprintf("rect_%d", rnd.Int())
		idText := fmt.Sprintf("text_%d", rnd.Int())

		textLines := []string{integration.Type, "---", integration.Details}
		textVal := strings.Join(textLines, "\n")
		height := 100.0

		rect := excalElement{
			ID:              idRect,
			Type:            "rectangle",
			X:               xBase,
			Y:               yOffset,
			Width:           250,
			Height:          height,
			StrokeColor:     "#1e293b",
			BackgroundColor: "#fffbeb",
			FillStyle:       "solid",
			StrokeWidth:     1.5,
			StrokeStyle:     "solid",
			Roughness:       0,
			Opacity:         100,
			Seed:            rnd.Int(),
			Version:         1,
			Updated:         time.Now().Unix(),
		}

		text := excalElement{
			ID:            idText,
			Type:          "text",
			X:             xBase + 15,
			Y:             yOffset + 15,
			Width:         220,
			Height:        height - 30,
			StrokeColor:   "#0f172a",
			StrokeWidth:   1,
			StrokeStyle:   "solid",
			Roughness:     0,
			Opacity:       100,
			Seed:          rnd.Int(),
			Version:       1,
			Updated:       time.Now().Unix(),
			Text:          textVal,
			FontSize:      16,
			FontFamily:    1,
			TextAlign:     "left",
			VerticalAlign: "top",
		}

		elements = append(elements, rect, text)
		yOffset += height + 40
	}

	doc := excalFile{
		Type:     "excalidraw",
		Version:  2,
		Source:   "https://synthspec.dev",
		Elements: elements,
		AppState: map[string]any{"theme": "light", "viewBackgroundColor": "#ffffff"},
		Files:    make(map[string]any),
	}

	if err := os.MkdirAll(distDir, 0755); err != nil {
		return "", err
	}

	destPath := filepath.Join(distDir, fmt.Sprintf("%s.excalidraw", projectName))
	fileBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(destPath, fileBytes, 0644); err != nil {
		return "", err
	}

	return destPath, nil
}
