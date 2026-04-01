package handler

import (
	"context"

	"github.com/mab-go/sheets-mcp/internal/toon"

	"github.com/mark3labs/mcp-go/mcp"
)

// SheetsClear handles the sheets_clear tool: clear a range without deleting structure.
func (h *SheetsHandler) SheetsClear(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	fullRange, rangeErr := buildRangeOrError(sheet, rangeStr)
	if rangeErr != nil {
		return rangeErr, nil
	}
	clearedRange, err := h.client.ClearRange(ctx, id, fullRange)
	if err != nil {
		if res, ok := mapAPIError(err); ok {
			return res, nil
		}
		return nil, err
	}

	encoded, encErr := toon.EncodeKeyValue([]toon.Field{
		{Key: "spreadsheet_id", Value: id},
		{Key: "sheet", Value: sheet},
		{Key: "cleared_range", Value: clearedRange},
	})
	if encErr != nil {
		return nil, encErr
	}

	return toonResult(encoded), nil
}
