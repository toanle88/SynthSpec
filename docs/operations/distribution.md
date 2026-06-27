# CLI Distribution Standards

As SynthSpec is an open-source, client-side CLI utility, it must build cleanly and cross-compile easily to target major systems: macOS, Windows, and Linux.

## Target Build Matrix

At compilation time, the project uses Go's build-in cross-compilation toolchain:

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o bin/synthspec.exe main.go

# macOS (Apple Silicon & Intel)
GOOS=darwin GOARCH=arm64 go build -o bin/synthspec-darwin-arm64 main.go
GOOS=darwin GOARCH=amd64 go build -o bin/synthspec-darwin-amd64 main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o bin/synthspec-linux-amd64 main.go
```

## Binary Distribution Practices

1. **Self-Contained Executable**: Binaries must compile statically, with all assets (prompts, themes) embedded directly in the binary using Go's `embed` package. The user should only need to download a single file.
2. **Release Integrity Checksums**: For every tag release, a `SHA256SUMS` file must be generated listing SHA-256 sums for all platform binaries to prevent tampering.
3. **Volatile Key Isolation**: Under no circumstances should test credentials, mock keys, or developers' keys be hardcoded or compiled into release binaries.
