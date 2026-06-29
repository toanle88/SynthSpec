package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// loadYAML is a generic helper that loads a YAML file from embedded defaults,
// checks for local override files, and unmarshals into the target type.
// localPaths are checked in order — the first existing file wins.
func loadYAML[T any](embedded []byte, localPaths []string) (T, error) {
	var zero T
	data := embedded

	for _, p := range localPaths {
		if _, err := os.Stat(p); err == nil {
			if fileData, readErr := os.ReadFile(p); readErr == nil {
				data = fileData
				break
			}
		}
	}

	var cfg T
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return zero, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	return cfg, nil
}
