package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/toanle/synthspec/gateway"
)

// ProposeUpstreamUpdate retroactively updates the upstream domain model with new entities/edge cases.
func ProposeUpstreamUpdate(ctx context.Context, gw gateway.Gateway, domainModel string, missingEntities []string) (string, error) {
	if len(missingEntities) == 0 {
		return domainModel, nil
	}

	prompt := fmt.Sprintf(`You are an expert software architect.
We have an upstream Domain Model document:
"""
%s
"""

Downstream documents have identified and introduced the following missing structural entities/edge-cases:
%s

Intelligently update the upstream Domain Model document to incorporate these new entities/edge-cases.
Integrate them naturally into the existing markdown structure (e.g., in the entities list or under a new section) without changing or deleting any other existing manually-written content.
Output only the updated markdown document. Do not include extra explanation or markdown block quotes.`, domainModel, strings.Join(missingEntities, ", "))

	updated, err := gw.GenerateSpecFile(ctx, gateway.Facts{}, "01_domain_model_use_cases.md", prompt)
	if err != nil {
		return "", err
	}
	return updated, nil
}
