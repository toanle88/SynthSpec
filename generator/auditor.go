package generator

import (
	"regexp"
	"strings"

	"github.com/toanle/synthspec/domain"
)

type ConsistencyAuditor struct{}

func NewConsistencyAuditor() *ConsistencyAuditor {
	return &ConsistencyAuditor{}
}

// Audit cross-references generated files to identify discrepancies.
func (a *ConsistencyAuditor) Audit(files map[string]string) (*domain.ConsistencyReport, error) {
	report := &domain.ConsistencyReport{
		Consistent: true,
		Feedback:   make(map[string]string),
	}

	domainModel, hasDomain := files["01_domain_model_use_cases.md"]
	if !hasDomain {
		return report, nil
	}

	// Find references like entity: User, struct User, model User, etc.
	entityRegex := regexp.MustCompile(`(?i:entity|struct|model|type)\s+([A-Z][a-zA-Z0-9_]+)`)

	for filename, content := range files {
		if filename == "01_domain_model_use_cases.md" {
			continue
		}

		matches := entityRegex.FindAllStringSubmatch(content, -1)
		var missing []string
		seen := make(map[string]bool)
		for _, m := range matches {
			if len(m) > 1 {
				entityName := m[1]
				if seen[entityName] {
					continue
				}
				seen[entityName] = true
				if !strings.Contains(domainModel, entityName) {
					missing = append(missing, entityName)
				}
			}
		}

		if len(missing) > 0 {
			report.Consistent = false
			report.Feedback[filename] = "Discrepancy: The following entities are referenced but missing from 01_domain_model_use_cases.md: " + strings.Join(missing, ", ")
		}
	}

	return report, nil
}
