package handler

import (
	"context"
	"fmt"

	"github.com/mab-go/sheets-mcp/internal/toon"

	"github.com/mark3labs/mcp-go/mcp"
)

// SheetsUpdate handles the sheets_update tool: write values to a range.
func (h *SheetsHandler) SheetsUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if rangeStr == "" {
		return mcp.NewToolResultError("missing required argument: range"), nil
	}

	values, err := extractValues(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	fullRange, rangeErr := buildRangeOrError(sheet, rangeStr)
	if rangeErr != nil {
		return rangeErr, nil
	}
	result, apiErr := h.client.WriteRange(ctx, id, fullRange, values)
	if apiErr != nil {
		if res, ok := mapAPIError(apiErr); ok {
			return res, nil
		}
		return nil, apiErr
	}

	encoded, err := toon.EncodeKeyValue([]toon.Field{
		{Key: "spreadsheet_id", Value: id},
		{Key: "sheet", Value: sheet},
		{Key: "updated_range", Value: result.UpdatedRange},
		{Key: "updated_rows", Value: result.UpdatedRows},
		{Key: "updated_cols", Value: result.UpdatedCols},
		{Key: "updated_cells", Value: result.UpdatedCells},
	})
	if err != nil {
		return nil, err
	}

	return toonResult(encoded), nil
}

// SheetsAppend handles the sheets_append tool: append rows after existing data.
func (h *SheetsHandler) SheetsAppend(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	id, toolErr := h.resolveSpreadsheetID(args)
	if toolErr != nil {
		return toolErr, nil
	}

	sheet, _ := args["sheet"].(string)
	if sheet == "" {
		return mcp.NewToolResultError("missing required argument: sheet (the tab name, e.g. \"Sheet1\")"), nil
	}

	values, err := extractValues(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, apiErr := h.client.AppendRows(ctx, id, sheet, values)
	if apiErr != nil {
		if res, ok := mapAPIError(apiErr); ok {
			return res, nil
		}
		return nil, apiErr
	}

	encoded, err := toon.EncodeKeyValue([]toon.Field{
		{Key: "spreadsheet_id", Value: id},
		{Key: "sheet", Value: sheet},
		{Key: "updated_range", Value: result.UpdatedRange},
		{Key: "updated_rows", Value: result.UpdatedRows},
		{Key: "updated_cols", Value: result.UpdatedCols},
		{Key: "updated_cells", Value: result.UpdatedCells},
	})
	if err != nil {
		return nil, err
	}

	return toonResult(encoded), nil
}

// extractValues extracts a 2D array from the MCP JSON arguments. The values
// parameter arrives as []interface{} containing []interface{} inner rows.
func extractValues(args map[string]any) ([][]any, error) {
	raw, ok := args["values"]
	if !ok || raw == nil {
		return nil, fmt.Errorf("missing required argument: values")
	}

	outerSlice, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("values must be a 2D array (array of arrays)")
	}

	if len(outerSlice) == 0 {
		return nil, fmt.Errorf("values must not be empty")
	}

	rows := make([][]any, len(outerSlice))
	for i, rawRow := range outerSlice {
		innerSlice, ok := rawRow.([]interface{})
		if !ok {
			return nil, fmt.Errorf("values[%d] must be an array, got %T", i, rawRow)
		}
		rows[i] = innerSlice
	}

	return rows, nil
}
