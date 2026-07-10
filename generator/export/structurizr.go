package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExportToStructurizr converts dense entities to Structurizr DSL C4 format
func ExportToStructurizr(projectName string, outputDir string, distDir string) (string, error) {
	entitiesPath := filepath.Join(outputDir, ".synthspec-entities.json")
	var data denseEntities

	if _, err := os.Stat(entitiesPath); err == nil {
		content, readErr := os.ReadFile(entitiesPath)
		if readErr == nil {
			_ = json.Unmarshal(content, &data)
		}
	}

	var dsl strings.Builder
	dsl.WriteString(fmt.Sprintf("workspace {\n    model {\n        # People/Users\n        user = person \"User\" \"Active actor of the system\"\n\n        # Software System\n        system = softwareSystem \"%s\" \"Synthesized Architecture System\" {\n", projectName))

	// Write Integrations as Containers
	for _, integration := range data.Integrations {
		cleanName := strings.ReplaceAll(integration.Type, "\"", "'")
		cleanDetails := strings.ReplaceAll(integration.Details, "\"", "'")
		dsl.WriteString(fmt.Sprintf("            container%s = container \"%s\" \"%s\"\n", strings.ReplaceAll(cleanName, " ", ""), cleanName, cleanDetails))
	}

	// Write Entities as Components inside a placeholder container
	dsl.WriteString("            coreAPI = container \"Core API Application\" \"Handles core domain business logic\" {\n")
	for _, ent := range data.Entities {
		cleanName := strings.ReplaceAll(ent.Name, "\"", "'")
		attrs := strings.Join(ent.Attributes, ", ")
		dsl.WriteString(fmt.Sprintf("                component%s = component \"%s\" \"Attributes: %s\"\n", strings.ReplaceAll(cleanName, " ", ""), cleanName, attrs))
	}
	dsl.WriteString("            }\n        }\n\n        # Relations\n        user -> system \"Uses\"\n")

	// Relations between components and integrations
	for _, integration := range data.Integrations {
		cleanName := strings.ReplaceAll(integration.Type, "\"", "'")
		dsl.WriteString(fmt.Sprintf("        system.coreAPI -> system.container%s \"Communicates with\"\n", strings.ReplaceAll(cleanName, " ", "")))
	}

	dsl.WriteString("    }\n\n    views {\n        systemContext system \"SystemContext\" {\n            include *\n            autolayout lr\n        }\n        container system \"Containers\" {\n            include *\n            autolayout lr\n        }\n        component system.coreAPI \"Components\" {\n            include *\n            autolayout lr\n        }\n        theme default\n    }\n}\n")

	if err := os.MkdirAll(distDir, 0755); err != nil {
		return "", err
	}

	destPath := filepath.Join(distDir, fmt.Sprintf("%s.dsl", projectName))
	if err := os.WriteFile(destPath, []byte(dsl.String()), 0644); err != nil {
		return "", err
	}

	return destPath, nil
}
