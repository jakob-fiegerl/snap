# Agent Guidelines for Snap

This document provides coding guidelines and conventions for AI agents working on the Snap codebase.

## Project Overview

Snap is a Git wrapper written in Go that provides intuitive, conversational commands for version control. It uses Charm's Bubble Tea for TUI interactions and Ollama's Phi-4 model for AI-powered commit messages.

**Stack:**
- Language: Go 1.24.1
- TUI Framework: Bubble Tea (charmbracelet/bubbletea)
- Styling: Lipgloss (charmbracelet/lipgloss)
- AI: Ollama API with Phi-4 model

## Build, Test, and Lint Commands

Whenever you build a new feature, make sure to add proper test, update the documentation and reinstall it.

### Building
```bash
# Build the binary
go build -o snap

# Build and install to ~/bin
./install.sh

# Tidy dependencies
go mod tidy
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -run TestFunctionName

# Run tests in a specific file/package
go test -v ./path/to/package

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting
```bash
# Format code (required before commit)
go fmt ./...

# Run go vet
go vet ./...

# Install and run golangci-lint (recommended)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run
```

### Running
```bash
# Run without installing
go run *.go save

# Run installed binary
snap save
snap changes
snap help
```

## Code Style Guidelines

### Imports
Follow standard Go import organization:
```go
import (
    // Standard library first
    "fmt"
    "strings"
    
    // Third-party packages second (alphabetically)
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)
```

### Formatting
- **ALWAYS** run `go fmt ./...` before committing
- Use tabs for indentation (Go standard)
- Maximum line length: ~100-120 characters (soft limit)
- Use `gofmt` style for all code

### Naming Conventions
- **Packages:** Single word, lowercase (e.g., `main`)
- **Files:** Lowercase with underscores if needed (e.g., `git.go`, `ollama.go`, `model.go`)
- **Types:** PascalCase (e.g., `OllamaRequest`, `model`, `state`)
- **Exported functions:** PascalCase (e.g., `GetGitDiff`, `CheckOllamaRunning`)
- **Unexported functions:** camelCase (e.g., `checkOllama`, `stageChanges`)
- **Constants:** camelCase for unexported, PascalCase for exported (e.g., `ollamaURL`, `stateChecking`)
- **Variables:** camelCase (e.g., `commitMessage`, `ollamaRunning`)

### Type Conventions
- Use explicit types for struct fields with JSON tags
- Use `iota` for enumerations (see `state` type in model.go)
- Prefer named return types for clarity when appropriate
- Use type aliases for better readability (e.g., `type state int`)

### Error Handling
- Always check and handle errors explicitly
- Return errors rather than panicking
- Use `fmt.Errorf` for error wrapping with context
- Pattern for error handling:
```go
result, err := SomeFunction()
if err != nil {
    return "", fmt.Errorf("failed to do something: %w", err)
}
```

### Comments
- Add package-level comments for exported functions
- Use `//` for single-line comments
- Document exported types and functions
- Example:
```go
// GetGitDiff returns the git diff of staged or unstaged changes
func GetGitDiff() (string, error) {
    // ...
}
```

## Project Structure

```
snap/
├── main.go          # CLI entry point, argument parsing, help text
├── model.go         # Bubble Tea TUI model, state management, view logic
├── git.go           # Git command wrappers (GetGitDiff, StageAllChanges, etc.)
├── ollama.go        # Ollama API integration for AI commit messages
├── go.mod           # Go module dependencies
├── install.sh       # Installation script
├── README.md        # User-facing documentation
└── AGENTS.md        # This file
```

## Command Implementation Pattern

When adding new commands to Snap:

1. **Add Git wrapper function** in `git.go`:
```go
func GitOperation() (string, error) {
    cmd := exec.Command("git", "operation", "flags")
    output, err := cmd.Output()
    return string(output), err
}
```

2. **Add command case** in `main.go`:
```go
case "commandname":
    result, err := GitOperation()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    fmt.Println(result)
    os.Exit(0)
```

3. **Update help text** in `printHelp()` function

4. **Update README.md** with new command documentation

## Bubble Tea Patterns

- State machine: Use `state` enum to track UI flow
- Messages: Create specific message types for async operations
- Commands: Return `tea.Cmd` for async operations
- Update: Handle all message types, return updated model and next command
- View: Render based on current state, use lipgloss for styling

## Ollama Integration

- Default endpoint: `http://localhost:11434`
- Default model: `phi4`
- Temperature: `0.3` for consistent commit messages
- Always check if Ollama is running before AI operations
- Clean up AI responses (remove prefixes, markdown artifacts)

## Testing Guidelines

- Write tests for all Git wrapper functions
- Test error conditions (Ollama not running, no changes, etc.)
- Mock external dependencies (Git commands, HTTP calls)
- Use table-driven tests for multiple scenarios

## Philosophy and Best Practices

1. **Simplicity:** Keep commands intuitive and conversational
2. **Safety:** Never destructively modify without confirmation
3. **Clarity:** Provide clear, helpful error messages
4. **Consistency:** Follow established patterns in the codebase
5. **User-friendly:** Default to the most common use case
6. **No staging area confusion:** Commands should work directly on working directory

## Common Pitfalls

- Forgetting to run `go fmt` before committing
- Not handling empty git diff (causes "no changes to commit" error)
- Not checking if Ollama is running before AI operations
- Using `panic` instead of returning errors
- Not updating help text when adding new commands
