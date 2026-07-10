package generator

import (
	"testing"
)

func TestConsistencyAuditor(t *testing.T) {
	auditor := NewConsistencyAuditor()

	files := map[string]string{
		"01_domain_model_use_cases.md": "This is the domain model describing User and Account.",
		"02_api_spec.md":               "The API model Transaction is processed.",
	}

	report, err := auditor.Audit(files)
	if err != nil {
		t.Fatalf("Audit failed: %v", err)
	}

	if report.Consistent {
		t.Error("Expected report to be inconsistent due to missing Transaction entity")
	}

	feedback, ok := report.Feedback["02_api_spec.md"]
	if !ok || !testing.Short() && len(feedback) == 0 {
		t.Error("Expected feedback for 02_api_spec.md")
	}
}
