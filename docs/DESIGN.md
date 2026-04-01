# DESIGN — sheets-mcp

API specification for [sheets-mcp](https://github.com/mab-go/sheets-mcp). Tool
definitions live in `internal/server/tools.go`; behavior is implemented under
`internal/server/handler/`. For OAuth setup, see [SETUP.md](SETUP.md).

---

## Configuration

Config file: `~/.config/sheets-mcp/config.json` (see
[SETUP.md](SETUP.md) for creation).

| Field                  | Required | Role                                                            |
|------------------------|----------|-----------------------------------------------------------------|
| `client_id`            | Yes      | OAuth2 client ID                                                |
| `client_secret`        | Yes      | OAuth2 client secret                                            |
| `default_spreadsheet`  | No       | Used when a tool omits `spreadsheet_id` (see below)             |
| `allowed_spreadsheets` | No       | When non-empty, restricts which spreadsheet IDs may be accessed |

Token file: `~/.config/sheets-mcp/token.json` (managed by the server; not
user-edited).

### OAuth scopes

Hardcoded in `internal/auth/oauth.go`:

- `https://www.googleapis.com/auth/spreadsheets`
- `https://www.googleapis.com/auth/drive.metadata.readonly`

---

## Spreadsheet ID resolution

Tools that accept `spreadsheet_id` use shared resolution
(`internal/server/handler/handler.go`):

1. Use the `spreadsheet_id` argument if non-empty.
2. Else use `default_spreadsheet` from config if set.
3. If still empty, return a tool error: missing `spreadsheet_id` (no default
   configured).
4. If `allowed_spreadsheets` is non-empty, the resolved ID must appear in that
   list; otherwise a tool error is returned.

`sheets_find` does **not** take `spreadsheet_id`. If `allowed_spreadsheets` is
set, search results are **filtered** to only spreadsheets in the list.

---

## Response format (TOON)

- **Tabular tools** (`sheets_get`, `sheets_lookup`): key-value metadata lines,
  then a `rows` tabular block when the encoder includes tabular data (see
  `internal/toon/encode.go` `EncodeToolResponse`).
- **List tools** (`sheets_find`, `sheets_info`): metadata plus a named list
  (`spreadsheets` or `sheets`).
- **Key-value only** (`sheets_update`, `sheets_append`, `sheets_clear`):
  metadata only, no `rows`.

Edge-case expectations for cell values (commas, empty cells, type preservation)
are exercised in `internal/toon/encode_test.go`.

---

## Header detection

Used by `sheets_get` (unless overridden) and `sheets_lookup`. Implemented in
`internal/sheets/headers.go`. Detection uses **Go types** from the Sheets API
(`[][]any`), not display formatting.

**Strong signals (both required) for treating row 1 as headers:**

- Row 1 has a contiguous span of non-empty strings from the first through last
  populated column (leading/trailing empty cells are allowed for unused
  columns).
- Row 2+ contains at least one non-string cell (numbers, booleans, etc.; `nil`
  cells are ignored).

**Anti-signals (row 1 is data):**

- Row 1 contains a non-string cell, or an empty cell between two non-empty cells
  in that span.
- Row 2+ contains only strings (after skipping `nil`) — no mixed-type signal.

**All-string tables:** If the first row and body are entirely strings,
auto-detection does **not** treat row 1 as headers. Use `headers: true` on
`sheets_get` or `sheets_lookup` to force keyed rows.

Every `sheets_get` / `sheets_lookup` response that includes tabular data
includes `headers_detected: true|false`. Both tools accept an optional boolean
`headers` to override auto-detection.

---

## Cross-cutting rules

### Empty results

Empty reads or no lookup matches are **not** errors. Intended behavior: valid
TOON with zero data rows where applicable. (`sheets_lookup` uses an empty header
list and empty rows for no matches.)

### Error handling philosophy

- **Tool errors**: Bad arguments, resolution failures, spreadsheet not allowed,
  user-recoverable API conditions. Returned as MCP tool results with `IsError`
  semantics; the Go handler returns `(result, nil)`.
- **Protocol errors**: Unexpected failures (I/O, unhandled API errors).
  Returned as `(nil, err)` from the handler.
- **Auth failures**: Should be surfaced as tool errors when possible so the
  client can show a clear message (e.g. re-run `sheets-mcp auth`).

### `sheet` and `range` (A1)

- **`sheet`**: Tab title as the user sees it in Google Sheets. The server quotes
  it for the Sheets API when building range strings (e.g. `'My Sheet'!A1`).
- **`range`** (when present on `sheets_get`, `sheets_update`, `sheets_clear`):
  The part after `sheet!`—not a second tab name. It is **validated locally**
  before any API call. Accepted forms include typical A1 references (`A1`,
  `A1:B2`, `A:A`, `1:100`, optional `$`), and a **simple named-range token**
  (ASCII: starts with a letter or `_`, then letters, digits, `_`; max length
  255). Rejected examples: embedded `!` or `;`, malformed `:` (e.g. `A1:` or
  `:B2`), or newlines. If validation passes but Google still rejects the range,
  the existing 400 → tool error path applies.

---

## Tool reference

All tool names are prefixed with `sheets_`. Parameters below match
`internal/server/tools.go`; defaults and extra behavior are from handlers
where noted.

### `sheets_find`

Drive metadata search by spreadsheet name. **No `spreadsheet_id` parameter.**

| Parameter | Type   | Required | Notes                                         |
|-----------|--------|----------|-----------------------------------------------|
| `query`   | string | Yes      | Search term matched against spreadsheet names |

**Response (TOON):** `query`, `count`, list `spreadsheets` with items:
`spreadsheet_id`, `title`, optional `modified` (date `YYYY-MM-DD`). Results
respect `allowed_spreadsheets` filtering when that list is non-empty.

---

### `sheets_info`

| Parameter        | Type   | Required | Notes                                                |
|------------------|--------|----------|------------------------------------------------------|
| `spreadsheet_id` | string | No*      | *Required unless `default_spreadsheet` is configured |

**Response (TOON):** `spreadsheet_id`, `title`, `sheet_count`, list `sheets`
with per-sheet: `title`, `index`, `rows`, `cols`, optional `headers` (first-row
preview string when present).

---

### `sheets_get`

| Parameter        | Type    | Required | Notes                                                                                                                                  |
|------------------|---------|----------|----------------------------------------------------------------------------------------------------------------------------------------|
| `spreadsheet_id` | string  | No*      | *Unless default configured                                                                                                             |
| `sheet`          | string  | Yes      | Tab name                                                                                                                               |
| `range`          | string  | No       | A1 fragment after the tab name; omit to read the **entire** sheet. Validated locally (see [Cross-cutting rules](#sheet-and-range-a1)). |
| `headers`        | boolean | No       | Overrides header auto-detection when present                                                                                           |

**Response (TOON):** `spreadsheet_id`, `sheet`, `range` (resolved range from
API), `headers_detected`, `row_count`, and tabular `rows` on every successful
read. Empty ranges have zero data rows and an empty column list in the tabular
block (no first row to infer columns).

---

### `sheets_update`

| Parameter        | Type           | Required | Notes                                                                                     |
|------------------|----------------|----------|-------------------------------------------------------------------------------------------|
| `spreadsheet_id` | string         | No*      | *Unless default configured                                                                |
| `sheet`          | string         | Yes      | Tab name                                                                                  |
| `range`          | string         | Yes      | A1 fragment to write; validated locally (see [Cross-cutting rules](#sheet-and-range-a1)). |
| `values`         | array of array | Yes      | 2D cell values; must be non-empty                                                         |

**Response (TOON):** `spreadsheet_id`, `sheet`, `updated_range`, `updated_rows`,
`updated_cols`, `updated_cells`.

---

### `sheets_append`

| Parameter        | Type           | Required | Notes                             |
|------------------|----------------|----------|-----------------------------------|
| `spreadsheet_id` | string         | No*      | *Unless default configured        |
| `sheet`          | string         | Yes      | Tab name                          |
| `values`         | array of array | Yes      | Rows to append; must be non-empty |

**Response (TOON):** Same shape as `sheets_update` (`updated_*` fields from the
append result).

---

### `sheets_lookup`

Reads the **entire sheet**, then filters rows by column match.

| Parameter        | Type            | Required | Notes                                                                                                                                                                        |
|------------------|-----------------|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `spreadsheet_id` | string          | No*      | *Unless default configured                                                                                                                                                   |
| `sheet`          | string          | Yes      | Tab name                                                                                                                                                                     |
| `match_column`   | string          | Yes      | Header name (case-insensitive) or column letters (`A`, `B`, …)                                                                                                               |
| `match_value`    | string          | Yes      | Value to search for                                                                                                                                                          |
| `match_mode`     | string          | No       | `exact`, `contains`, `prefix`; default **`contains`** if omitted or empty                                                                                                    |
| `return_columns` | array of string | No       | Header names to include; omit for all columns. Unknown names are skipped when at least one name matches; if none match, the tool returns an error listing available columns. |
| `headers`        | boolean         | No       | Overrides header auto-detection when present (same as `sheets_get`)                                                                                                          |

Matching is **case-insensitive** for cell text and for `match_value`.

**Response (TOON):** `spreadsheet_id`, `sheet`, `match_column`, `match_value`,
`match_mode`, `headers_detected`, `row_count`, plus tabular `rows` for matches.

---

### `sheets_clear`

| Parameter        | Type   | Required | Notes                                                                                                   |
|------------------|--------|----------|---------------------------------------------------------------------------------------------------------|
| `spreadsheet_id` | string | No*      | *Unless default configured                                                                              |
| `sheet`          | string | Yes      | Tab name                                                                                                |
| `range`          | string | Yes      | A1 fragment to clear (values only); validated locally (see [Cross-cutting rules](#sheet-and-range-a1)). |

**Response (TOON):** `spreadsheet_id`, `sheet`, `cleared_range`. Row/column/cell
counts are not returned because the Google Sheets API's clear operation does not
provide count metadata in its response.

---

## Explicit non-goals

Do not implement the following. If asked, decline and point here.

- Formatting (bold, colors, borders, conditional formatting)
- Charts and pivot tables
- Creating new spreadsheets (only operate on existing ones)
- Sheet structure modification (adding/deleting sheets, columns, rows)
- Formula-specific tooling (writing `=SUM(...)` into a cell works naturally as
  a string)
- Real-time / push updates
- Multi-user / workspace features
- Service account authentication
