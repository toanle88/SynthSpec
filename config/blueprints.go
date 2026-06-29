package config

import (
	_ "embed"
)

//go:embed blueprints.yaml
var defaultBlueprintsYAML []byte

type BlueprintFacts struct {
	Functional string `yaml:"functional" json:"functional"`
	Structural string `yaml:"structural" json:"structural"`
	Security   string `yaml:"security" json:"security"`
	Compliance string `yaml:"compliance" json:"compliance"`
}

type Blueprint struct {
	ID          string         `yaml:"id" json:"id"`
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description" json:"description"`
	Facts       BlueprintFacts `yaml:"facts" json:"facts"`
}

type BlueprintsConfig struct {
	Blueprints []Blueprint `yaml:"blueprints"`
}

// LoadBlueprints loads the blueprints from a local override file or falls back to the embedded defaults.
func LoadBlueprints() ([]Blueprint, error) {
	cfg, err := loadYAML[BlueprintsConfig](defaultBlueprintsYAML, []string{
		"blueprints.yaml",
		".synthspec/blueprints.yaml",
	})
	if err != nil {
		return nil, err
	}
	return cfg.Blueprints, nil
}
