# Claude Code Development Guide

## Running Tests

### Unit Tests Only

To run only the unit tests (excluding subsystem/integration tests), use:

```bash
go list ./... | grep -v subsystem | xargs go test -v
```

This command:
- Lists all packages in the project
- Excludes the `subsystem` package (which contains integration tests)
- Runs tests with verbose output (`-v`)

### All Tests (Including Subsystem Tests)

To run all tests including subsystem tests:

```bash
go test -v ./...
```

**Note:** Subsystem tests require additional infrastructure (WireMock server) and will fail without proper setup.

### Running Tests for a Specific Package

To run tests for a specific package:

```bash
go test -v ./src/inventory
go test -v ./pkg/journalLogger
```

### Running Tests Without Verbose Output

Remove the `-v` flag for less verbose output:

```bash
go list ./... | grep -v subsystem | xargs go test
```

## Test Organization

- **Unit tests**: Located alongside source code in various `src/` and `pkg/` directories
- **Subsystem tests**: Located in the `subsystem/` directory (require additional setup)
