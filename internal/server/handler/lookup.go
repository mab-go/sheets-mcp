package handler

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/mab-go/sheets-mcp/internal/toon"

	"github.com/mark3labs/mcp-go/mcp"
)

// SheetsLookup handles the sheets_lookup tool: find rows by column value match.
func (h *SheetsHandler) SheetsLookup(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	id, toolErr := h.resolveSpreadsheetID(args)
	if toolErr != nil {
		return toolErr, nil
	}

	sheet, _ := args["sheet"].(string)
	if sheet == "" {
		return mcp.NewToolResultError("missing required argument: sheet (the tab name, e.g. \"Sheet1\")"), nil
	}

	matchColumn, _ := args["match_column"].(string)
	if matchColumn == "" {
		return mcp.NewToolResultError("missing required argument: match_column"), nil
	}

	matchValue, _ := args["match_value"].(string)
	if matchValue == "" {
		return mcp.NewToolResultError("missing required argument: match_value"), nil
	}

	matchMode, _ := args["match_mode"].(string)
	if matchMode == "" {
		matchMode = "contains"
	}
	if matchMode != "contains" && matchMode != "exact" && matchMode != "prefix" {
		return mcp.NewToolResultError(
			fmt.Sprintf("invalid match_mode %q: must be \"exact\", \"contains\", or \"prefix\"", matchMode),
		), nil
	}

	// Read the entire sheet.
	fullRange, rangeErr := buildRangeOrError(sheet, "")
	if rangeErr != nil {
		return rangeErr, nil
	}
	data, err := h.client.ReadRange(ctx, id, fullRange)
	if err != nil {
		if res, ok := mapAPIError(err); ok {
			return res, nil
		}
		return nil, err
	}

	if len(data.Values) == 0 {
		return emptyLookupResult(id, sheet, matchColumn, matchValue, matchMode)
	}

	headersOverride, hasOverride := args["headers"].(bool)
	headersDetected, headers, dataRows := headerLayoutForValues(data.Values, hasOverride, headersOverride)

	// Find the target column index.
	colIdx := findColumnIndex(headers, matchColumn)
	if colIdx < 0 {
		return mcp.NewToolResultError(
			fmt.Sprintf("column %q not found. Available columns: %s", matchColumn, strings.Join(headers, ", ")),
		), nil
	}

	// Filter rows.
	matchedRows := filterRows(dataRows, colIdx, matchValue, matchMode)

	// Determine return columns.
	returnHeaders := headers
	returnRows := matchedRows
	if rawCols, ok := args["return_columns"]; ok {
		var ferr error
		returnHeaders, returnRows, ferr = filterColumns(headers, matchedRows, rawCols)
		if ferr != nil {
			return mcp.NewToolResultError(ferr.Error()), nil
		}
	}

	metadata := []toon.Field{
		{Key: "spreadsheet_id", Value: id},
		{Key: "sheet", Value: sheet},
		{Key: "match_column", Value: matchColumn},
		{Key: "match_value", Value: matchValue},
		{Key: "match_mode", Value: matchMode},
		{Key: "headers_detected", Value: headersDetected},
		{Key: "row_count", Value: len(returnRows)},
	}

	encoded, err := toon.EncodeToolResponse(metadata, returnHeaders, returnRows)
	if err != nil {
		return nil, err
	}

	return toonResult(encoded), nil
}

// emptyLookupResult returns a valid TOON response for zero matches.
func emptyLookupResult(id, sheet, matchColumn, matchValue, matchMode string) (*mcp.CallToolResult, error) {
	metadata := []toon.Field{
		{Key: "spreadsheet_id", Value: id},
		{Key: "sheet", Value: sheet},
		{Key: "match_column", Value: matchColumn},
		{Key: "match_value", Value: matchValue},
		{Key: "match_mode", Value: matchMode},
		{Key: "headers_detected", Value: false},
		{Key: "row_count", Value: 0},
	}

	encoded, err := toon.EncodeToolResponse(metadata, []string{}, [][]any{})
	if err != nil {
		return nil, err
	}

	return toonResult(encoded), nil
}

// findColumnIndex returns the 0-based index of the column matching name.
// If headers are detected, it matches case-insensitively against header names.
// Otherwise it tries to parse as a column letter (A, B, C, ...).
func findColumnIndex(headers []string, name string) int {
	// Try header name match first (case-insensitive).
	lower := strings.ToLower(name)
	for i, h := range headers {
		if strings.ToLower(h) == lower {
			return i
		}
	}

	// Try column letter (A=0, B=1, ..., AA=26, etc.).
	if idx := columnLetterToIndex(name); idx >= 0 && idx < len(headers) {
		return idx
	}

	return -1
}

// columnLetterToIndex converts a spreadsheet column letter to a 0-based index.
// Returns -1 if the string is not a valid column letter.
func columnLetterToIndex(s string) int {
	s = strings.ToUpper(s)
	if s == "" {
		return -1
	}
	idx := 0
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return -1
		}
		idx = idx*26 + int(c-'A') + 1
	}
	return idx - 1
}

// filterRows returns rows where the cell at colIdx matches the value
// according to the given mode. All comparisons are case-insensitive.
func filterRows(rows [][]any, colIdx int, value string, mode string) [][]any {
	lowerValue := strings.ToLower(value)
	var matched [][]any

	for _, row := range rows {
		var cellStr string
		if colIdx < len(row) && row[colIdx] != nil {
			cellStr = strings.ToLower(fmt.Sprintf("%v", row[colIdx]))
		}

		if matchCell(cellStr, lowerValue, mode) {
			matched = append(matched, row)
		}
	}

	return matched
}

// matchCell checks whether a cell value matches the target value using the
// specified mode. Both values should already be lowercased.
func matchCell(cell, target, mode string) bool {
	switch mode {
	case "exact":
		return cell == target
	case "prefix":
		return strings.HasPrefix(cell, target)
	case "contains":
		return strings.Contains(cell, target)
	default:
		return strings.Contains(cell, target)
	}
}

// filterColumns filters the headers and row data to only include the
// specified columns. If colNames is non-empty but none resolve, it returns
// an error listing available columns (fail-fast).
func filterColumns(allHeaders []string, rows [][]any, rawCols any) ([]string, [][]any, error) {
	colNames := extractStringSlice(rawCols)
	if len(colNames) == 0 {
		return allHeaders, rows, nil
	}

	// Build the index mapping.
	indices := make([]int, 0, len(colNames))
	filteredHeaders := make([]string, 0, len(colNames))
	for _, name := range colNames {
		idx := findColumnIndex(allHeaders, name)
		if idx >= 0 {
			indices = append(indices, idx)
			filteredHeaders = append(filteredHeaders, allHeaders[idx])
		}
	}

	if len(indices) == 0 {
		return nil, nil, fmt.Errorf(
			"return_columns: none of [%s] matched. Available columns: %s",
			strings.Join(colNames, ", "),
			strings.Join(allHeaders, ", "),
		)
	}

	filteredRows := make([][]any, len(rows))
	for i, row := range rows {
		newRow := make([]any, len(indices))
		for j, idx := range indices {
			if idx < len(row) {
				newRow[j] = row[idx]
			}
		}
		filteredRows[i] = newRow
	}

	return filteredHeaders, filteredRows, nil
}

// extractStringSlice extracts a []string from a JSON-decoded value, which
// may be []interface{} from MCP parameter parsing.
func extractStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return slices.Clone(v)
	case string:
		// Single column name passed as a string.
		return []string{v}
	default:
		return nil
	}
}
