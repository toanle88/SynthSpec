// Package config provides layered YAML configuration management.
//
// Configuration concerns are split across focused files:
//   - providers.go: Provider constants, model constants, LoadConfig
//   - standards.go: Standard type, LoadStandards, FilterApplicableStandards
//   - templates.go: Template type, LoadTemplates
//   - blueprints.go: Blueprint type, LoadBlueprints
//   - settings.go: Settings type, LoadSettings
//   - yaml.go: Generic loadYAML[T] helper
package config
