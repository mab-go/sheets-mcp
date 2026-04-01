# Contributing to sheets-mcp

## Development setup

**Prerequisites:** Go 1.26.1 or later.

Install project-local tools (golangci-lint and goimports into `./bin`):

```
make setup
```

Build the binary to `./bin/sheets-mcp`:

```
make build
```

Run directly without building:

```
make run
```

## Running tests

Run all tests with the race detector enabled (default):

```
make test
```

Generate a coverage report (opens HTML in browser; skips open when `CI` is set):

```
make test:cover
```

Disable the race detector if needed: `RACE=0 make test`.

## Code style

Format with goimports (handles both `gofmt` formatting and import grouping):

```
make fmt
```

Lint with golangci-lint:

```
make lint
```

Run `go vet`:

```
make vet
```

CI enforces both lint and tests on every push and pull request.

## Submitting changes

1. Fork the repository and create a feature branch.
2. Make your changes, ensuring `make test`, `make lint`, and `make vet` all
   pass.
3. Open a pull request against `main`. CI must pass before a PR is merged.

## Project conventions

- **Tool errors vs. Go errors:** User-facing problems (bad arguments, API
  errors, resolution failures) are returned as MCP tool results using
  `mcp.NewToolResultError`, not as Go errors. Go errors are reserved for
  unexpected protocol-level failures.
- **TOON responses:** All tool responses are encoded in TOON (Token-Oriented
  Object Notation) using `internal/toon`. JSON in, TOON out.
- **A1 validation:** Range strings are validated locally before any Sheets API
  call. See the `sheet` and `range` section in [docs/DESIGN.md](docs/DESIGN.md)
  for accepted forms.
