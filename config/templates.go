package config

import (
	_ "embed"
)

//go:embed templates.yaml
var defaultTemplatesYAML []byte

type Template struct {
	FileName         string `yaml:"file_name"`
	Name             string `yaml:"name"`
	IsSource         bool   `yaml:"is_source"`
	RequiresNonEmpty bool   `yaml:"requires_non_empty"`
	Prompt           string `yaml:"prompt"`
}

type TemplatesConfig struct {
	Templates []Template `yaml:"templates"`
}

// LoadTemplates loads the templates from a local override file or falls back to the embedded defaults.
func LoadTemplates() ([]Template, error) {
	cfg, err := loadAndMergeYAML[TemplatesConfig](defaultTemplatesYAML, []string{
		"templates.yaml",
		".synthspec/templates.yaml",
	}, func(base, override TemplatesConfig) TemplatesConfig {
		m := make(map[string]int)
		for i, tmpl := range base.Templates {
			m[tmpl.FileName] = i
		}

		for _, tmpl := range override.Templates {
			if idx, exists := m[tmpl.FileName]; exists {
				base.Templates[idx] = tmpl
			} else {
				base.Templates = append(base.Templates, tmpl)
			}
		}
		return base
	})
	if err != nil {
		return nil, err
	}
	return cfg.Templates, nil
}
