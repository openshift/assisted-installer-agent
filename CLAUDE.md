# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The Assisted Installer Agent is part of the OpenShift Assisted Installer Service. See README.md for a complete list of executables and their purposes.

## Architecture

### Single Binary Design

All executables (agent, next_step_runner, inventory, free_addresses, logs_sender, disk_speed_check) are **symlinks to a single binary**. The binary checks `os.Args[0]` to determine which functionality to execute. See `src/agent/main/main.go:28-41` for the dispatch logic.

### Command Processing Flow

1. **next_step_runner** polls the assisted-service API for the next installation step
2. **step_processor.go** (`src/commands/step_processor.go`) orchestrates step execution
3. Individual step implementations live in `src/commands/actions/` (e.g., `install_cmd.go`, `inventory_cmd.go`, `connectivity_check_cmd.go`)
4. Steps send results back to the service via the API client

### Key Directories

- `src/agent/`: Main entry point and agent initialization
- `src/next_step_runner/`: Step polling loop
- `src/commands/`: Command framework and step processor
- `src/commands/actions/`: Individual step implementations (20+ different steps)
- `src/inventory/`: Hardware inventory collection using the ghw library
- `src/config/`: Configuration processing
- `src/session/`: API session management
- `pkg/journalLogger/`: Systemd journal logging

## Building

See README.md for full build instructions. Quick reference:

### With Skipper (Recommended)
```bash
skipper make              # Build executables
skipper make build-image  # Build container image
```

### Without Skipper
```bash
make build         # Build to build/agent
make build-image   # Build container image
```

The Dockerfile uses a multi-stage build (see `Dockerfile.assisted_installer_agent`). The final image is based on CentOS Stream 9 and includes various system tools (dmidecode, ipmitool, fio, nmap, etc.).

## Testing

### Unit Tests Only

To run only unit tests (excluding subsystem/integration tests):

```bash
# Using skipper
skipper make unit-test

# Direct command (recommended in Claude Code sandbox)
go list ./... | grep -v subsystem | xargs go test -v
```

This command:
- Lists all packages in the project
- Excludes the `subsystem` package (which contains integration tests)
- Runs tests with verbose output (`-v`)

### Running Tests for a Specific Package

```bash
go test -v ./src/inventory
go test -v ./pkg/journalLogger
go test -v ./src/commands/actions
```

### Running Tests Without Verbose Output

Remove the `-v` flag for less verbose output:

```bash
go list ./... | grep -v subsystem | xargs go test
```

### All Tests (Including Subsystem Tests)

Subsystem tests require Docker Compose and WireMock infrastructure:

```bash
# Using skipper
skipper make subsystem

# Run specific subsystem tests
skipper make subsystem FOCUS=register

# Direct (requires docker-compose)
make subsystem
```

See `subsystem/docker-compose.yml` for the test infrastructure setup. **Note:** Subsystem tests cannot run in sandboxed environments and require the default image name/tag.

### Test Organization

- **Unit tests**: Located alongside source code in `src/` and `pkg/` directories
- **Subsystem tests**: Located in `subsystem/` directory
- **Test framework**: Ginkgo (BDD) and Gomega (assertions)

### Known Issues

#### DNS Resolution Test in Sandbox Mode

The `src/domain_resolution` package contains a test that performs actual DNS lookups using the reserved `.invalid` TLD (RFC 2606). When running in a sandboxed environment (like Claude Code), this test may fail with:

```
lookup nonexistent.invalid: Temporary failure in name resolution
```

This is expected behavior due to network restrictions in the sandbox. In a normal environment, the DNS resolver returns an `IsNotFound` error (which the code suppresses), but in the sandbox the DNS lookup is blocked entirely, resulting in an `IsTemporary` error.

The test will pass when run outside the sandbox where DNS access is available.

## Linting

```bash
skipper make lint     # Run golangci-lint
skipper make ci-lint  # Vendor diff + Dockerfile sync + golangci-lint
```

## Code Generation

```bash
make generate   # Run go:generate directives
make go-import  # Format imports with goimports
```

## Key Dependencies

- **github.com/jaypipes/ghw**: Hardware inventory discovery (CPU, memory, disk, GPU, NIC)
- **github.com/openshift/assisted-service**: Service API client and models
- **github.com/vishvananda/netlink**: Network interface management
- **github.com/sirupsen/logrus**: Structured logging
- **github.com/onsi/ginkgo**: BDD testing framework

See `go.mod` for the complete dependency list.

## Configuration

Agent configuration is handled via command-line flags and environment variables. See README.md "Agent Flags" section for available options. Configuration processing is in `src/config/`.

## GPU Discovery

The inventory process supports configurable GPU discovery filtering. See README.md "Inventory Flags" section for details on the GPU configuration file format.
