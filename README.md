# sheets-mcp

<p align="center">
  <a href="https://github.com/mab-go/sheets-mcp/actions"><img src="https://img.shields.io/github/check-runs/mab-go/sheets-mcp/main?style=flat&labelColor=555555&label=checks" alt="Build Status" /></a>
  <a href="https://goreportcard.com/report/github.com/mab-go/sheets-mcp"><img src="https://goreportcard.com/badge/github.com/mab-go/sheets-mcp?cachebuster=5000" alt="Go Report Card" /></a>
  <a href="https://pkg.go.dev/github.com/mab-go/sheets-mcp"><img src="https://img.shields.io/badge/-reference-00ADD8?style=flat&logo=go&logoColor=white&labelColor=555555" alt="Go Reference" /></a>
  <a href="https://deepwiki.com/mab-go/sheets-mcp"><img src="https://img.shields.io/badge/DeepWiki-sheets--mcp-blue?style=flat&logoColor=white&labelColor=555555" alt="Ask DeepWiki"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/mab-go/sheets-mcp" alt="License: MIT" /></a>
</p>

A purpose-built MCP server for Google Sheets. Sheets MCP gives any
MCP-compatible AI client conversational read/write access to your spreadsheets.
Find sheets by name, read ranges, look up rows, append data, and update cells.

Responses use [TOON](https://github.com/toon-format/toon-go) (Token-Oriented
Object Notation). Field names appear once per response rather than on every row,
cutting 40–60% of output tokens versus JSON for typical Sheets data. Tool
definitions are compact by design: ~1,500 tokens on load.

Sheets MCP is distributed as a single compiled binary, works over stdio
transport, and uses OAuth authentication against your personal Google account.

---

## Tools

| Tool            | What it does                                                       |
|-----------------|--------------------------------------------------------------------|
| `sheets_find`   | Search for spreadsheets by name                                    |
| `sheets_info`   | Get sheet/tab names, dimensions, and header rows for a spreadsheet |
| `sheets_get`    | Read a range; returns keyed objects when headers are detected      |
| `sheets_update` | Write values to a range (overwrites existing content)              |
| `sheets_append` | Append rows after the last row of data                             |
| `sheets_lookup` | Find rows where a column matches a value                           |
| `sheets_clear`  | Clear all values in a range without deleting cells or structure    |

`sheets_get` auto-detects whether the first row is headers and returns rows as
keyed objects when it is. Every response includes a `headers_detected` flag, and
you can override the heuristic with an explicit `headers` parameter if needed.
`sheets_lookup` uses the same auto-detection and optional `headers` override.

`sheets_lookup` supports `exact`, `contains` (default), and `prefix` match
modes, plus a `return_columns` parameter to narrow responses to only the
columns you need.

---

## Requirements

- Go (current stable)
- Linux with `xdg-open` (developed and tested on Ubuntu)
- A personal Google account (not Google Workspace)
- Claude Desktop

---

## Installation

```bash
go install github.com/mab-go/sheets-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/mab-go/sheets-mcp
cd sheets-mcp
make build
```

---

## Setup

### 1. Create a GCP project and OAuth credentials

See **`docs/SETUP.md`** for a step-by-step walkthrough (~10 minutes, one-time).

The short version: create a GCP project, enable the Google Sheets and Drive
APIs, create a Desktop OAuth client, and note your client ID and secret.

### 2. Create the config file

```bash
mkdir -p ~/.config/sheets-mcp
```

Create `~/.config/sheets-mcp/config.json`:

```json
{
  "client_id": "YOUR_CLIENT_ID",
  "client_secret": "YOUR_CLIENT_SECRET",
  "default_spreadsheet": "",
  "allowed_spreadsheets": []
}
```

`default_spreadsheet` is optional — set it to a spreadsheet ID if you want tool
calls to work without specifying one explicitly.

`allowed_spreadsheets` is optional — when non-empty, the server restricts all
access to only the listed spreadsheet IDs.

### 3. Authenticate

```bash
sheets-mcp auth
```

This opens a browser window for Google's OAuth consent flow. After you approve,
the token is stored at `~/.config/sheets-mcp/token.json`.

### 4. Register with Claude Desktop

Add to `claude_desktop_config.json` (typically at
`~/.config/claude-desktop/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "sheets": {
      "command": "/home/YOUR_USERNAME/go/bin/sheets-mcp",
      "args": ["serve"]
    }
  }
}
```

Restart Claude Desktop. sheets-mcp will appear in the tools list.

---

## CLI Reference

```
sheets-mcp serve          # Start MCP server (stdio) — used by Claude Desktop
sheets-mcp auth           # Run OAuth flow
sheets-mcp auth --status  # Check token state without triggering a flow
sheets-mcp auth --revoke  # Revoke and delete stored token
sheets-mcp version        # Print version and exit
```

---

## Re-authentication

Google's OAuth tokens for apps in Testing mode expire after 7 days of
inactivity. If Claude reports an authentication error, run:

```bash
sheets-mcp auth
```

This is a Google limitation for unverified apps — see `docs/SETUP.md` for
details.

---

## What This Doesn't Do

By design, sheets-mcp focuses on data access only:

- No formatting (bold, colors, borders, conditional formatting)
- No charts or pivot tables
- No creating new spreadsheets (operates on existing ones only)
- No adding or deleting sheets, columns, or rows
- No real-time or push updates

Writing formulas works naturally — any string starting with `=` is written as-is
by the Sheets API.

