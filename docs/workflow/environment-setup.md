# Environment Setup

## Prerequisites

### Install Go

```bash
brew install go
```

Verify: `go version` should show Go 1.21+.

### CGO Requirement

The project uses `mattn/go-sqlite3` which requires CGO. On macOS this works out of the box with Xcode Command Line Tools:

```bash
xcode-select --install  # if not already installed
```

## Project Initialization (One-Time)

These commands should be run once when starting implementation:

```bash
go mod init github.com/leeovery/tick
```

Dependencies will be added incrementally as implementation proceeds via `go get`.

## Development Tooling

```bash
go install gotest.tools/gotestsum@latest  # better test output
```

## Verify Setup

```bash
go version        # Go 1.21+
go env GOPATH     # should be set
which gotestsum   # test runner available
```
