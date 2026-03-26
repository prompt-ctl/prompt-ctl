# Contributing to promptctl

Thank you for your interest in contributing to promptctl! This guide will help you get started.

## Developer Setup

### Requirements

- **Go 1.22+** ([download](https://go.dev/dl/))
- **Git**

### Clone and Build

```bash
git clone https://github.com/prompt-ctl/prompt-ctl.git
cd prompt-ctl
go mod download
go mod verify
go build -o promptctl .
```

Verify the build:

```bash
./promptctl version
```

## Running the Project

| Task | Command |
|------|---------|
| Build | `go build -o promptctl .` |
| Run | `./promptctl --help` |
| Tests | `go test ./...` |
| Coverage | `go test -cover ./...` |
| Vet | `go vet ./...` |
| Format | `gofmt -w .` |

You can also use `make build` and `make test` for convenience.

## Code Style

### Formatting

- Run `gofmt -w .` before every commit. CI will reject unformatted code.
- Run `go vet ./...` to catch common mistakes. This must pass.

### Naming Conventions

- Exported identifiers use `CamelCase` (e.g., `Execute`, `RunPrompt`)
- Unexported identifiers use `camelCase` with lowercase first letter (e.g., `createPrompt`, `runFix`)
- Package names are short, lowercase, single-word (e.g., `cmd`, `llm`, `prompt`)

### Error Handling

- Always check and return errors; do not silently ignore them.
- Use `fmt.Errorf("context: %w", err)` to wrap errors with context.
- Avoid `panic()` in library code; return errors to the caller.

## Pull Request Process

1. **Fork** the repository and create a feature branch:
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Write your code** following the style guidelines above.

3. **Commit** using conventional commit messages:
   - `feat: add new template type for API testing`
   - `fix: correct variable substitution in nested templates`
   - `chore: update Go dependencies`
   - `docs: improve installation instructions`
   - `test: add coverage for score command`

4. **Push** your branch and open a Pull Request.

5. **PR description** should:
   - Reference the related issue (e.g., "Closes #42")
   - Describe what changed and why
   - Include any testing you did

6. **CI must pass** before merge. The CI pipeline runs:
   - `go vet ./...`
   - `go test ./...`
   - Build verification

## Testing Requirements

### Unit Tests

Every new function should have corresponding tests. Place test files alongside the code they test:

```
cmd/score.go        -> cmd/score_test.go
internal/safepath/  -> internal/safepath/*_test.go
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./cmd/...

# Run a specific test
go test ./cmd/ -run TestScoreCommand
```

### Writing a Test

Here is a minimal example for testing a command helper:

```go
package cmd

import (
    "testing"
)

func TestMyHelper(t *testing.T) {
    got := myHelper("input")
    want := "expected output"
    if got != want {
        t.Errorf("myHelper() = %q, want %q", got, want)
    }
}
```

For table-driven tests (preferred for multiple cases):

```go
func TestMyHelper(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  string
    }{
        {"basic", "input", "expected"},
        {"empty", "", ""},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := myHelper(tt.input)
            if got != tt.want {
                t.Errorf("myHelper(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

## Adding a New Command

promptctl uses a manual command dispatch in `cmd/root.go`. To add a new command:

1. **Create the file** `cmd/mycmd.go`:

   ```go
   package cmd

   import (
       "fmt"
   )

   func runMyCmd() error {
       // Your command logic here
       fmt.Println("Hello from mycmd!")
       return nil
   }
   ```

2. **Register the command** in the `switch` statement in `cmd/root.go`:

   ```go
   case "mycmd":
       return runMyCmd()
   ```

3. **Add tests** in `cmd/mycmd_test.go`.

4. **Update the help text** in the `printUsage()` function in `cmd/root.go`.

5. **Update README.md** if the command is user-facing.

## Reporting Issues

### Bug Reports

When reporting a bug, please include:

- **OS and architecture** (e.g., macOS ARM64, Ubuntu 22.04 x86_64)
- **Go version** (`go version`)
- **promptctl version** (`promptctl version`)
- **Steps to reproduce** the issue
- **Expected vs actual behavior**
- **Error output** (if any)

### Feature Requests

- Describe the problem you are trying to solve
- Explain your proposed solution
- Consider how it fits with existing commands and templates

### Where to Report

- **Bugs**: [GitHub Issues](https://github.com/prompt-ctl/prompt-ctl/issues)
- **Ideas & Discussion**: [GitHub Discussions](https://github.com/prompt-ctl/prompt-ctl/discussions)

## Project Structure

```
promptctl/
  cmd/          CLI commands (root.go dispatches all commands)
  config/       Configuration management (~/.promptctl/)
  internal/     Core internal packages
    analytics/  Usage analytics
    onboarding/ First-run setup
    safepath/   Path validation
    shell/      Shell integration
    ui/         Terminal UI helpers
  llm/          LLM provider integrations
  prompt/       Embedded prompt templates (go:embed)
  scripts/      Build and release scripts
```

## Questions?

If something is unclear, open an issue or start a discussion. We are happy to help you get started.
