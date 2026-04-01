package handler

import (
	"context"

	"github.com/mab-go/sheets-mcp/internal/toon"

	"github.com/mark3labs/mcp-go/mcp"
)

// SheetsFind handles the sheets_find tool: search for spreadsheets by name.
func (h *SheetsHandler) SheetsFind(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	query, _ := args["query"].(string)
	if query == "" {
		return mcp.NewToolResultError("missing required argument: query"), nil
	}

	results, err := h.client.FindSpreadsheets(ctx, query)
	if err != nil {
		if res, ok := mapAPIError(err); ok {
			return res, nil
		}
		return nil, err
	}

	results = h.filterByAllowedList(results)

	items := make([][]toon.Field, len(results))
	for i, r := range results {
		fields := []toon.Field{
			{Key: "spreadsheet_id", Value: r.SpreadsheetID},
			{Key: "title", Value: r.Title},
		}
		if !r.ModifiedTime.IsZero() {
			fields = append(fields, toon.Field{Key: "modified", Value: r.ModifiedTime.Format("2006-01-02")})
		}
		items[i] = fields
	}

	encoded, err := toon.EncodeList(
		toTOONFields("query", query, "count", len(results)),
		"spreadsheets",
		items,
	)
	if err != nil {
		return nil, err
	}

	return toonResult(encoded), nil
}
