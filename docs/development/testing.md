# General Testing Standards

All code changes in the SynthSpec repository must pass automated verification checks before merging.

## Test Suite Requirements
1. **No Real Network Calls**: Tests must never attempt to contact real LLM API endpoints. Use mock providers.
2. **Deterministic Outputs**: Ensure logic tests (like session state math and percentage scores) are deterministic and free from race conditions.
3. **Cross-Platform Verification**: Run tests on both Windows and Unix environments to check path compatibility (e.g. backslashes vs. forward slashes in local state directories).

## CI/CD Verification
When a pull request is created:
- Standard GitHub Actions workflows execute `go test ./...`.
- Code must achieve at least 80% coverage on core packages (`state/` and `generator/`).
