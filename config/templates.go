package config

import (
	_ "embed"
)

//go:embed templates.yaml
var defaultTemplatesYAML []byte

type Template struct {
	FileName string `yaml:"file_name"`
	Name     string `yaml:"name"`
	IsSource bool   `yaml:"is_source"`
	Prompt   string `yaml:"prompt"`
}

type TemplatesConfig struct {
	Templates []Template `yaml:"templates"`
}

// LoadTemplates loads the templates from a local override file or falls back to the embedded defaults.
func LoadTemplates() ([]Template, error) {
	cfg, err := loadYAML[TemplatesConfig](defaultTemplatesYAML, []string{
		"templates.yaml",
		".synthspec/templates.yaml",
	})
	if err != nil {
		return nil, err
	}
	return cfg.Templates, nil
}
