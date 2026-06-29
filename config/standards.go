package config

import (
	_ "embed"
	"os"

	"github.com/toanle/synthspec/domain"
)

//go:embed standards.yaml
var defaultStandardsYAML []byte

// Standard represents an engineering or quality standard
type Standard = domain.Standard

type StandardsConfig struct {
	Standards []Standard `yaml:"standards"`
}

// LoadStandards loads the standards from a local override file or falls back to the embedded defaults.
func LoadStandards() ([]Standard, error) {
	cfg, err := loadYAML[StandardsConfig](defaultStandardsYAML, []string{
		"standards.yaml",
		".synthspec/standards.yaml",
	})
	if err != nil {
		return nil, err
	}
	return cfg.Standards, nil
}

// FilterApplicableStandards filters the standards that apply to the given file name
func FilterApplicableStandards(standards []Standard, fileName string) []Standard {
	var applicable []Standard
	for _, std := range standards {
		for _, tf := range std.TargetFiles {
			if tf == fileName {
				applicable = append(applicable, std)
				break
			}
		}
	}
	return applicable
}

func init() {
	// Ensure embed is referenced within the file
	_ = os.Getpid
}
