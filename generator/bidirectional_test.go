package generator

import (
	"context"
	"testing"
)

func TestProposeUpstreamUpdate(t *testing.T) {
	tg := &TestGateway{
		responses: map[string][]string{
			"01_domain_model_use_cases.md": {
				"Updated domain model: added Transaction",
			},
		},
		callCounts: make(map[string]int),
	}

	res, err := ProposeUpstreamUpdate(context.Background(), tg, "Old model", []string{"Transaction"})
	if err != nil {
		t.Fatalf("ProposeUpstreamUpdate failed: %v", err)
	}

	if res != "Updated domain model: added Transaction" {
		t.Errorf("expected updated content, got: %q", res)
	}
}
