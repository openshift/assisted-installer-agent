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

## Known Issues

### DNS Resolution Test in Sandbox Mode

The `src/domain_resolution` package contains a test that performs actual DNS lookups using the reserved `.invalid` TLD (RFC 2606). When running in a sandboxed environment (like Claude Code), this test may fail with:

```
lookup nonexistent.invalid: Temporary failure in name resolution
```

This is expected behavior due to network restrictions in the sandbox. In a normal environment, the DNS resolver returns an `IsNotFound` error (which the code suppresses), but in the sandbox the DNS lookup is blocked entirely, resulting in an `IsTemporary` error.

The test will pass when run outside the sandbox where DNS access is available.
