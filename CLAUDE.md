# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The Janus Framework is a Go library for chaining security tools together to create complex, reusable workflows that can run at scale. It provides a uniform interface for connecting disparate security tools and enables automation of multi-step security processes.

## Core Architecture

### Key Components

- **Links** (`pkg/chain/link.go`): Individual processing units that implement the `Link` interface and define a `Process` method. Each link transforms data and can send zero or more outputs to the next link in the chain.

- **Chains** (`pkg/chain/chain.go`): Collections of links that process data sequentially through Go channels, enabling concurrent processing while maintaining order.

- **Multi-Chains** (`pkg/chain/multichain.go`): Run multiple processing pipelines in parallel, useful for applying different analysis methods to the same data.

- **Outputters** (`pkg/chain/outputter.go`): Handle final results, supporting multiple formats (JSON, Markdown, custom writers). Located in `pkg/output/`.

- **Configuration System** (`pkg/chain/cfg/`): Robust configuration with CLI arguments, environment variables, and programmatic configuration with type safety and validation.

### Package Structure

- `pkg/chain/`: Core framework components (chains, links, configuration)
- `pkg/links/`: Built-in link implementations (Docker, NoseyParker, utility links)
- `pkg/output/`: Output formatters (JSON, Markdown, Writer)
- `pkg/types/`: Common data types used throughout the framework
- `pkg/util/`: Utility functions
- `pkg/testutils/`: Testing utilities and mocks

## Development Commands

### Building
```bash
go build ./...
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./pkg/chain/...

# Run a specific test
go test -run TestName ./pkg/chain/
```

### Code Quality
```bash
# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Run static analysis (if golangci-lint is available)
golangci-lint run
```

### Module Management
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Configuration System

The framework uses a parameter/argument distinction:
- **Parameters**: Definitions of what configuration a link accepts (defined in `Params()` method)
- **Arguments**: Actual values provided when using a link (using `cfg.WithArg()`)

Configuration priority order (highest to lowest):
1. Programmatic Arguments (`cfg.WithArg()`)
2. CLI Arguments
3. Environment Variables
4. Default Values

## Common Patterns

### Creating Custom Links
All custom links should:
1. Embed `*chain.Base`
2. Implement the `Process` method
3. Define parameters via `Params()` method
4. Use `ps.Send()` to pass data to the next link

### Chain Construction
```go
c := chain.NewChain(
    // Add links in processing order
    links.NewSomeLink(),
    links.NewAnotherLink(),
).WithConfigs(
    cfg.WithArg("param-name", value),
).WithOutputters(
    output.NewJSONOutputter(),
)
```

### Error Handling
- Links should return errors from `Process()` method
- Chains propagate errors and can be checked with `c.Error()`
- Use `chain.RecvAs[T]()` for type-safe output retrieval

## Testing

The framework provides extensive testing utilities in `pkg/testutils/`:
- Mock links for testing chain behavior
- Test utilities for common operations
- Example implementations for reference

Run tests with race detection:
```bash
go test -race ./...
```

## Integration Notes

- The framework integrates with Chariot for security tool orchestration
- Uses Tabularium models for credential management
- Docker integration available via `pkg/links/docker/`
- NoseyParker integration for secret scanning