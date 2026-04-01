package handler

import (
	"context"

	"github.com/mab-go/sheets-mcp/internal/toon"

	"github.com/mark3labs/mcp-go/mcp"
)

// SheetsInfo handles the sheets_info tool: get spreadsheet metadata.
func (h *SheetsHandler) SheetsInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	id, toolErr := h.resolveSpreadsheetID(args)
	if toolErr != nil {
		return toolErr, nil
	}

	meta, err := h.client.GetSpreadsheetMetadata(ctx, id)
	if err != nil {
		if res, ok := mapAPIError(err); ok {
			return res, nil
		}
		return nil, err
	}

	items := make([][]toon.Field, len(meta.Sheets))
	for i, s := range meta.Sheets {
		fields := []toon.Field{
			{Key: "title", Value: s.Title},
			{Key: "index", Value: s.Index},
			{Key: "rows", Value: s.RowCount},
			{Key: "cols", Value: s.ColCount},
		}
		if headers := headersFromFirstRow(s.FirstRow); headers != "" {
			fields = append(fields, toon.Field{Key: "headers", Value: headers})
		}
		items[i] = fields
	}

	encoded, err := toon.EncodeList(
		toTOONFields("spreadsheet_id", meta.SpreadsheetID, "title", meta.Title, "sheet_count", len(meta.Sheets)),
		"sheets",
		items,
	)
	if err != nil {
		return nil, err
	}

	return toonResult(encoded), nil
}
