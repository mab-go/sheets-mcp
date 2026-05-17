# AGENTS.md — sheets-mcp

This file is the authoritative briefing for any AI agent working on this project. Read it in full before writing any code or making any changes.

---

## Project Summary

`sheets-mcp` is a Go MCP server that provides tools for reading and writing Google Sheets data via the Sheets API. The module path is `github.com/mab-go/sheets-mcp` and the project lives at `/home/matt/Projects/mcp/sheets-mcp`.

`docs/DESIGN.md` is the authoritative source for all design decisions. Read it before implementing anything.

---

## Project Structure

```
cmd/
  sheets-mcp/
    main.go               — cobra commands; 'serve' calls server.RunStdioServer(),
                            'auth' handles OAuth flow, 'version' prints build info
internal/
  auth/
    oauth.go              — OAuth2 flow: browser launch, local callback server,
                            token exchange
    token.go              — token storage, refresh, expiry checking
    config.go             — config.json loading (client_id, client_secret, etc.)
  server/
    server.go             — MCP server setup and tool registration
    tools.go              — tool definitions as package-level var declarations
    hooks.go              — MCP lifecycle hooks (before/after initialize)
    handler/
      handler.go          — SheetsHandler struct and shared utilities
                            (textResult, toonResult, errorResult, etc.)
      apierr.go           — mapAPIError: Google API and OAuth transport errors → tool errors
      find.go             — sheets_find handler (Drive metadata search)
      info.go             — sheets_info handler (spreadsheet metadata)
      read.go             — sheets_get handler (read range with smart headers)
      write.go            — sheets_update, sheets_append handlers
      lookup.go           — sheets_lookup handler (column value search)
      header_layout_test.go — tests for headerLayoutForValues (shared get/lookup layout)
      clear.go            — sheets_clear handler
  sheets/
    client.go             — Google Sheets API client wrapper
    a1.go                 — A1 notation helpers (sheet name quoting, range fragment validation)
    a1_test.go            — A1 helper tests
    headers.go            — header detection heuristic
    headers_test.go       — header detection tests
    types.go              — internal types for ranges, cell data, responses
  toon/
    encode.go             — TOON tabular encoding for tool responses
    encode_test.go        — encoding tests
  version/
    version.go            — build metadata (Version, Commit, Date) via ldflags
  logging/                — context-carried logging helpers; do not modify
docs/
  DESIGN.md              — product/API design specification (authoritative)
  SETUP.md               — step-by-step GCP OAuth setup guide
.agents/
  skills/
    ask-questions/         — structured question-gathering with tiered unknowns
    commit-message/        — Git commit message generation from diffs
    explore-codebase/      — systematic pre-implementation codebase exploration
    make-todos/            — structured task list creation and tracking
    review-plan/           — plan review for accuracy, correctness, and task items
    verify-changes/        — build/test/lint verification and doc update checks
.claude/
  skills/                 — symlink → ../.agents/skills/
Makefile                  — build, test, lint, fmt targets
.golangci.yml             — golangci-lint v2 config
.goreleaser.yaml          — goreleaser config; cross-builds Linux/Windows archives on `v*` tags
.editorconfig             — editor formatting rules
CLAUDE.md                 — symlink to AGENTS.md (Claude project instructions)
```

---

## Repo Exploration & Discovery

**Prefer LSP for semantic Go queries.** For finding definitions, references,
implementations, call hierarchy, or type signatures in Go code, use the LSP
tool over grep -- gopls is configured for this repo and returns semantic
answers that disambiguate same-named symbols across packages. Grep is still
fine for plain text searches, non-code files, and quick locate-by-string.

---

## Tool Surface (7 tools)

All tool names are prefixed with `sheets_` to avoid collisions in multi-server environments. The `spreadsheet_id` parameter is optional on all tools when `default_spreadsheet` is set in config.

| Tool | Purpose | Key parameters |
|---|---|---|
| `sheets_find` | Search for spreadsheets by name | `query` |
| `sheets_info` | Spreadsheet metadata (tabs, dimensions, headers) | `spreadsheet_id` |
| `sheets_get` | Read a range (smart header detection) | `spreadsheet_id`, `sheet`, `range`, `headers` |
| `sheets_update` | Write values to a range | `spreadsheet_id`, `sheet`, `range`, `values` |
| `sheets_append` | Append rows after existing data | `spreadsheet_id`, `sheet`, `values` |
| `sheets_lookup` | Find rows by column value match | `spreadsheet_id`, `sheet`, `match_column`, `match_value`, `match_mode`, `return_columns`, `headers` |
| `sheets_clear` | Clear a range without deleting structure | `spreadsheet_id`, `sheet`, `range` |

For `sheets_get`, `sheets_update`, and `sheets_clear`, the `range` argument (when provided) is validated locally before calling Google; see **sheet and range (A1)** in `docs/DESIGN.md`.

Full parameter specifications are in `docs/DESIGN.md`.

---

## Response Format: TOON

All tool responses use TOON (Token-Oriented Object Notation) tabular encoding, not JSON. This is a core design decision. Tool inputs remain JSON (MCP's default parameter format). The architecture is asymmetric by design: JSON in, TOON out.

Use `github.com/toon-format/toon-go` for encoding. If edge cases arise, `internal/toon/` is the place to wrap or patch the library.

A typical tool response:

```
spreadsheet_id: abc123
sheet: Rentals
range: A1:H50
headers_detected: true
rows[49]{Address,Price,Beds,Baths,Status,Notes,Link,Updated}:
  123 4th Ave SW,1450,2,1,Active,,https://...,2025-03-15
  456 Centre St NE,1800,3,1,Contacted,Called Tuesday,https://...,2025-03-12
```

Key-value metadata at the top, tabular data below. Field names appear once in the header, not on every row.

### TOON encoding edge cases

Validate these patterns with tests in `internal/toon/encode_test.go`:

- Strings containing commas (e.g., addresses) — must be quoted
- Empty cells — must produce correct delimiter count, not collapse
- Strings that look like numbers (phone numbers, postal codes) — must stay strings
- Strings that look like booleans (`"true"`, `"false"`) — must be quoted
- Null vs empty string distinction
- Unicode content (emoji, CJK characters)

---

## Header Detection (Smart Read)

Implemented in `internal/sheets/headers.go`. Default behavior for `sheets_get` and `sheets_lookup`.

Detection uses **Go types** in `[][]any` as returned by the Sheets read path (not inferred sheet formatting).

**Strong signals that row 1 is headers (both required):**
- Row 1 has a contiguous span of non-empty strings from the first through last populated column (leading/trailing empty cells are allowed for unused columns)
- Row 2+ contains at least one non-string cell (numbers, booleans, etc.; `nil` cells are ignored)

**Anti-signals (row 1 is data):**
- Row 1 contains a non-string cell, or an empty cell between two non-empty cells in that span
- Row 2+ contains only strings (after skipping `nil`) — the mixed-type signal is absent

**All-string tables:** If the header row and body are entirely strings, auto-detection does **not** treat row 1 as headers. Set `headers: true` on `sheets_get` to force keyed rows.

Every response includes `headers_detected: true|false`. The optional `headers` parameter on `sheets_get` overrides auto-detection.

---

## Authentication

### OAuth flow (`internal/auth/oauth.go`)

1. Bind a local HTTP server to `localhost` on port 0 (OS-assigned)
2. Build Google OAuth consent URL with the actual port in the redirect URI
3. Open system browser via `xdg-open` (Linux)
4. User consents in browser
5. Google redirects to `localhost:PORT/callback?code=...`
6. Server catches the code, exchanges for access + refresh tokens
7. Store tokens to `~/.config/sheets-mcp/token.json` (0600 permissions)
8. Shut down the local server

Timeout: 2 minutes. On timeout, return a clear error directing the user to run `sheets-mcp auth` manually.

### Scopes

Exactly two scopes — hardcoded, not configurable:
- `https://www.googleapis.com/auth/spreadsheets`
- `https://www.googleapis.com/auth/drive.metadata.readonly`

### Token refresh (`internal/auth/token.go`)

Before every API call, check access token expiry. If expired, refresh silently. If refresh fails (revoked, network), surface a clear auth error as a tool error (not a protocol error) so the LLM can relay it to the user.

Token refresh, revocation (`auth --revoke`), and the Sheets/Drive HTTP client use bounded timeouts so network calls cannot hang indefinitely.

### Configuration (`~/.config/sheets-mcp/config.json`)

```json
{
  "client_id": "...",
  "client_secret": "...",
  "default_spreadsheet": "",
  "allowed_spreadsheets": []
}
```

- `client_id` / `client_secret`: required. If missing, refuse to start with a message directing the user to `docs/SETUP.md`.
- `default_spreadsheet`: fallback `spreadsheet_id` when tool calls omit the parameter.
- `allowed_spreadsheets`: when non-empty, restricts all tool access to listed IDs.

Token storage at `~/.config/sheets-mcp/token.json` — managed by the server, not user-edited.

---

## CLI

```
sheets-mcp serve              # MCP server mode (stdio), used by Claude Desktop
sheets-mcp auth               # Interactive OAuth flow
sheets-mcp auth --status      # Check token state without triggering a flow
sheets-mcp auth --revoke      # Revoke and delete stored token
sheets-mcp version            # Print version and exit
```

MCP registration in `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "sheets": {
      "command": "/home/matt/go/bin/sheets-mcp",
      "args": ["serve"]
    }
  }
}
```

---

## Conventions

These patterns are established conventions. Follow them exactly.

### Tool definitions

Define tools as package-level `var` declarations in `internal/server/tools.go`:

```go
var toolSheetsGet = mcp.NewTool(
    "sheets_get",
    mcp.WithDescription("Read data from a Google Sheets range. Returns rows as keyed objects if header row is detected, otherwise as arrays. Omit range to read the entire sheet."),
    mcp.WithString("spreadsheet_id", mcp.Description("Spreadsheet ID; uses default if omitted and configured")),
    mcp.WithString("sheet", mcp.Required(), mcp.Description("Sheet/tab name")),
    mcp.WithString("range", mcp.Description("A1 notation range; omit to read entire sheet")),
    mcp.WithBoolean("headers", mcp.Description("Override header auto-detection")),
)
```

Register them in `server.go`:

```go
s.AddTool(toolSheetsGet, h.SheetsGet)
```

### Tool descriptions

Write descriptions for the LLM, not for humans. Single sentence. No examples in descriptions. Every token is overhead on every conversation.

### Handler method signature

```go
func (h *SheetsHandler) SheetsGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
```

### Response helpers (defined in `handler.go`)

| Helper | Purpose |
|--------|---------|
| `headerLayoutForValues` | Split `[][]any` grid into header labels and data rows for `sheets_get` and `sheets_lookup` (override + `DetectHeaders`); see `read.go` / `lookup.go` |

```go
// textResult returns a successful text response.
func textResult(text string) *mcp.CallToolResult {
    return &mcp.CallToolResult{
        Content: []mcp.Content{mcp.TextContent{Type: "text", Text: text}},
    }
}

// toonResult encodes data as TOON and returns it as a text response.
func toonResult(data any) (*mcp.CallToolResult, error) {
    encoded, err := toon.Encode(data)
    if err != nil {
        return nil, fmt.Errorf("encode TOON response: %w", err)
    }
    return textResult(string(encoded)), nil
}
```

### Argument extraction and spreadsheet ID resolution

```go
args := req.GetArguments()

spreadsheetID, _ := args["spreadsheet_id"].(string)
if spreadsheetID == "" {
    spreadsheetID = h.config.DefaultSpreadsheet
}
if spreadsheetID == "" {
    return mcp.NewToolResultError("missing required argument: spreadsheet_id (no default configured)"), nil
}
```

Every handler accepting `spreadsheet_id` must: try the argument, fall back to `config.DefaultSpreadsheet`, check against `config.AllowedSpreadsheets` if non-empty, and return a tool error on failure. Use the shared helper in `handler.go`.

### Logging

Do not modify `internal/logging/`. Use the context-carried logger:

```go
log, _ := logging.FromContext(ctx)
log.WithField("spreadsheet_id", id).Debug("Reading range")
```

### Dependencies

```
github.com/mark3labs/mcp-go          — MCP protocol
github.com/toon-format/toon-go       — TOON encoding
github.com/sirupsen/logrus           — logging (via internal/logging)
github.com/spf13/cobra               — CLI
github.com/spf13/viper               — config
golang.org/x/oauth2                  — OAuth2 flow
google.golang.org/api/sheets/v4      — Sheets API client
google.golang.org/api/drive/v3       — Drive API client (metadata only)
google.golang.org/api/googleapi      — Google API error type (shared with Sheets/Drive clients)
```

Do not add dependencies beyond this set without good reason. For HTTP and JSON handling, use Go stdlib.

### Verification

Treat any change as incomplete until **`make build test lint`** passes. The Makefile runs golangci-lint (including revive); do not rely on `go test` alone.

**Cyclomatic complexity (report-only):** After `make setup`, **`make cyclo`** runs [gocyclo](https://github.com/fzipp/gocyclo) with **`-over 10`** on the module tree (`gocyclo` takes a directory path, not a Go package pattern—see the Makefile). It lists functions with complexity **strictly greater than 10** (including `*_test.go`). This is a stricter bar than [Go Report Card](https://goreportcard.com/)'s public check (**cyclo-over=15**); the badge's **percentage is file-based** (any over-threshold function fails the file), while the CLI lists **individual functions**, so do not expect the same headline number. CI uploads a **`cyclo.txt`** artifact from a **non-blocking** job; it does not fail the workflow when violations exist.

---

## Error Handling

Two distinct failure modes — do not conflate them.

### Tool errors (expected / recoverable)

Bad arguments, sheet not found, spreadsheet not accessible, auth expired:

```go
return mcp.NewToolResultError("sheet 'Rentals' not found in spreadsheet. Available sheets: Budget, Listings, Archive"), nil
```

Signature: `(*mcp.CallToolResult, nil)`. The Go error is `nil`.

**Error messages must suggest the fix.** Not "range not found" but "Range 'Rentals!A1:Z100' not found. Available sheets: Budget, Listings, Archive."

### Protocol errors (unexpected / unrecoverable)

I/O failures, internal bugs, and **unclassified** API errors (after attempting `mapAPIError`):

```go
return nil, fmt.Errorf("sheets API: %w", err)
```

Reserve for genuinely unrecoverable situations. Expected Google API failures (HTTP 400/401/403/404/429, OAuth refresh errors) should be tool errors via `mapAPIError`, not protocol errors.

### Auth errors

Auth failures are tool errors, not protocol errors — the LLM must relay the message to the user:

```go
return mcp.NewToolResultError("authentication expired and refresh failed. Run 'sheets-mcp auth' to re-authenticate."), nil
```

### Google API errors

After a call to `h.client` (Sheets/Drive), classify failures with `mapAPIError` in [`internal/server/handler/apierr.go`](internal/server/handler/apierr.go). If it returns `ok`, return that tool result and `nil` error; otherwise propagate the error as a protocol error (unexpected or non-API).

```go
result, err := h.client.ReadRange(ctx, id, rng)
if err != nil {
    if res, ok := mapAPIError(err); ok {
        return res, nil
    }
    return nil, err
}
```

### Empty results

Not errors. `sheets_get` on an empty range returns valid TOON with a tabular `rows` block (`rows[0]:` with zero data rows and no column headers). `sheets_lookup` with no matches uses the same shape.

---

## Explicit Non-Goals

Do not implement these. If a request falls into this list, decline it and reference `docs/DESIGN.md`.

- Formatting (bold, colors, borders, conditional formatting)
- Charts and pivot tables
- Creating new spreadsheets (only operate on existing ones)
- Sheet structure modification (adding/deleting sheets, columns, rows)
- Formula-specific tooling (writing `=SUM(...)` into a cell works naturally as a string)
- Real-time / push updates
- Multi-user / workspace features
- Service Account authentication

---

## Documentation Maintenance

After any structural change, update the corresponding documentation:

| Change made | What to review/update |
|---|---|
| New file added or removed | `Project Structure` tree in this file |
| Tool parameters, response shapes, or non-goals changed | `docs/DESIGN.md` and Tool Surface table in this file |
| New handler helper added to `handler.go` | Conventions section in this file |
| New tool registered | `docs/DESIGN.md` tool reference and Tool Surface table in this file |
| New convention established | Conventions section in this file |
| New dependency added | Dependencies list in this file |
| Auth flow changed | Authentication section in this file |

Only update docs when something has genuinely changed — no cosmetic edits.
