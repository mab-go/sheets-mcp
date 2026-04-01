// Package handler implements MCP tool handlers for sheets-mcp.
package handler

import (
	"fmt"
	"slices"

	"github.com/mab-go/sheets-mcp/internal/auth"
	"github.com/mab-go/sheets-mcp/internal/sheets"
	"github.com/mab-go/sheets-mcp/internal/toon"

	"github.com/mark3labs/mcp-go/mcp"
)

// SheetsHandler handles tool calls for Google Sheets operations.
type SheetsHandler struct {
	client *sheets.Client
	config *auth.Config
}

// NewSheetsHandler returns a new handler with the given client and config.
func NewSheetsHandler(client *sheets.Client, config *auth.Config) *SheetsHandler {
	return &SheetsHandler{client: client, config: config}
}

// headerLayoutForValues splits a value grid into header labels and data rows using
// the same rules as sheets_get: optional override, else DetectHeaders.
func headerLayoutForValues(values [][]any, hasOverride bool, override bool) (headersDetected bool, headers []string, dataRows [][]any) {
	switch {
	case len(values) == 0:
		return false, []string{}, [][]any{}
	case hasOverride:
		headersDetected = override
		if headersDetected {
			return true, toStringSlice(values[0]), values[1:]
		}
		return false, generateColumnHeaders(len(values[0])), values
	default:
		headersDetected = sheets.DetectHeaders(values)
		if headersDetected {
			return headersDetected, toStringSlice(values[0]), values[1:]
		}
		return false, generateColumnHeaders(len(values[0])), values
	}
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: text}},
	}
}

func toonResult(encoded string) *mcp.CallToolResult {
	return textResult(encoded)
}

// resolveSpreadsheetID tries the argument value, falls back to the configured
// default, checks the allow list, and returns a tool error if resolution fails.
func (h *SheetsHandler) resolveSpreadsheetID(args map[string]any) (string, *mcp.CallToolResult) {
	id, _ := args["spreadsheet_id"].(string)
	if id == "" {
		id = h.config.DefaultSpreadsheet
	}
	if id == "" {
		return "", mcp.NewToolResultError("missing required argument: spreadsheet_id (no default configured)")
	}

	if len(h.config.AllowedSpreadsheets) > 0 {
		if !slices.Contains(h.config.AllowedSpreadsheets, id) {
			return "", mcp.NewToolResultError(
				fmt.Sprintf("spreadsheet %q is not in the allowed list. Allowed spreadsheets: %v", id, h.config.AllowedSpreadsheets),
			)
		}
	}

	return id, nil
}

// buildRangeOrError builds a Sheets API range string with a quoted sheet title.
// If rangeStr is empty, the range is the whole sheet (quoted title only).
// If rangeStr is non-empty, it is validated as an A1 fragment before appending.
func buildRangeOrError(sheet, rangeStr string) (string, *mcp.CallToolResult) {
	if sheet == "" {
		return "", mcp.NewToolResultError("missing required argument: sheet")
	}
	quoted := sheets.QuoteSheetNameForA1(sheet)
	if rangeStr == "" {
		return quoted, nil
	}
	if err := sheets.ValidateA1RangeFragment(rangeStr); err != nil {
		return "", mcp.NewToolResultError(
			fmt.Sprintf(
				"invalid range: %v. Use A1 notation (e.g. A1:D10, A:A, 1:100) or a simple named range; put the tab name in the sheet argument.",
				err,
			),
		)
	}
	return quoted + "!" + rangeStr, nil
}

// headersFromTOON converts a first-row slice from SheetInfo into TOON Field items
// suitable for display in sheets_info output.
func headersFromFirstRow(firstRow []any) string {
	if len(firstRow) == 0 {
		return ""
	}
	allStrings := true
	for _, v := range firstRow {
		if _, ok := v.(string); !ok || v == "" {
			allStrings = false
			break
		}
	}
	if !allStrings {
		return ""
	}

	result := ""
	for i, v := range firstRow {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%v", v)
	}
	return result
}

// filterByAllowedList filters search results against the allowed spreadsheets
// list. If the list is empty, all results are returned.
func (h *SheetsHandler) filterByAllowedList(results []sheets.SearchResult) []sheets.SearchResult {
	if len(h.config.AllowedSpreadsheets) == 0 {
		return results
	}
	filtered := make([]sheets.SearchResult, 0, len(results))
	for _, r := range results {
		if slices.Contains(h.config.AllowedSpreadsheets, r.SpreadsheetID) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// toTOONFields converts a slice of toon.Field pairs from variadic key-value arguments.
func toTOONFields(pairs ...any) []toon.Field {
	fields := make([]toon.Field, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		key, _ := pairs[i].(string)
		fields = append(fields, toon.Field{Key: key, Value: pairs[i+1]})
	}
	return fields
}
