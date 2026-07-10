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

// loadAndMergeYAML parses the embedded YAML config first, then deep-merges local overrides
// sequentially using a user-supplied merge function.
func loadAndMergeYAML[T any](embedded []byte, localPaths []string, mergeFn func(base T, override T) T) (T, error) {
	var zero T
	var base T
	if err := yaml.Unmarshal(embedded, &base); err != nil {
		return zero, fmt.Errorf("failed to parse embedded YAML configuration: %w", err)
	}

	for _, p := range localPaths {
		if _, err := os.Stat(p); err == nil {
			if fileData, readErr := os.ReadFile(p); readErr == nil {
				var override T
				if err := yaml.Unmarshal(fileData, &override); err == nil {
					base = mergeFn(base, override)
				} else {
					return zero, fmt.Errorf("failed to parse local override YAML (%s): %w", p, err)
				}
			}
		}
	}

	return base, nil
}

