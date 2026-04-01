package handler

import (
	"context"
	"fmt"

	"github.com/mab-go/sheets-mcp/internal/toon"

	"github.com/mark3labs/mcp-go/mcp"
)

// SheetsGet handles the sheets_get tool: read data from a range.
func (h *SheetsHandler) SheetsGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	id, toolErr := h.resolveSpreadsheetID(args)
	if toolErr != nil {
		return toolErr, nil
	}

	sheet, _ := args["sheet"].(string)
	if sheet == "" {
		return mcp.NewToolResultError("missing required argument: sheet (the tab name, e.g. \"Sheet1\")"), nil
	}

	rangeStr, _ := args["range"].(string)
	fullRange, rangeErr := buildRangeOrError(sheet, rangeStr)
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

	headersOverride, hasOverride := args["headers"].(bool)
	headersDetected, headerRow, dataRows := headerLayoutForValues(data.Values, hasOverride, headersOverride)

	metadata := []toon.Field{
		{Key: "spreadsheet_id", Value: id},
		{Key: "sheet", Value: sheet},
		{Key: "range", Value: data.Range},
		{Key: "headers_detected", Value: headersDetected},
		{Key: "row_count", Value: len(dataRows)},
	}

	encoded, err := toon.EncodeToolResponse(metadata, headerRow, dataRows)
	if err != nil {
		return nil, err
	}

	return toonResult(encoded), nil
}

// toStringSlice converts a row of any values to strings for use as headers.
func toStringSlice(row []any) []string {
	result := make([]string, len(row))
	for i, v := range row {
		result[i] = fmt.Sprintf("%v", v)
	}
	return result
}

// generateColumnHeaders creates A, B, C, ... AA, AB, ... style headers.
func generateColumnHeaders(count int) []string {
	headers := make([]string, count)
	for i := range count {
		headers[i] = columnLetter(i)
	}
	return headers
}

// columnLetter converts a 0-based column index to a spreadsheet column letter.
func columnLetter(index int) string {
	result := ""
	for {
		result = string(rune('A'+index%26)) + result
		index = index/26 - 1
		if index < 0 {
			break
		}
	}
	return result
}
