package server

import "github.com/mark3labs/mcp-go/mcp"

var toolSheetsFind = mcp.NewTool(
	"sheets_find",
	mcp.WithDescription("Search for Google Sheets spreadsheets by name. Returns matching spreadsheet IDs, titles, and last modified dates."),
	mcp.WithString("query", mcp.Required(), mcp.Description("Search term to match against spreadsheet names")),
)

var toolSheetsInfo = mcp.NewTool(
	"sheets_info",
	mcp.WithDescription("Get metadata for a spreadsheet: sheet/tab names, dimensions, and detected header rows for each sheet."),
	mcp.WithString("spreadsheet_id", mcp.Description("Spreadsheet ID; uses default if omitted and configured")),
)

var toolSheetsGet = mcp.NewTool(
	"sheets_get",
	mcp.WithDescription("Read data from a Google Sheets range. Returns rows as keyed objects if header row is detected, otherwise as arrays. Omit range to read the entire sheet."),
	mcp.WithString("spreadsheet_id", mcp.Description("Spreadsheet ID; uses default if omitted and configured")),
	mcp.WithString("sheet", mcp.Required(), mcp.Description("Sheet/tab name")),
	mcp.WithString("range", mcp.Description("A1 notation range; omit to read entire sheet")),
	mcp.WithBoolean("headers", mcp.Description("Override header auto-detection: true = treat row 1 as headers, false = treat row 1 as data (columns labeled A/B/C/...), omit = auto-detect")),
)

var toolSheetsUpdate = mcp.NewTool(
	"sheets_update",
	mcp.WithDescription("Write values to a Google Sheets range. Overwrites existing cell contents within the specified range."),
	mcp.WithString("spreadsheet_id", mcp.Description("Spreadsheet ID; uses default if omitted and configured")),
	mcp.WithString("sheet", mcp.Required(), mcp.Description("Sheet/tab name")),
	mcp.WithString("range", mcp.Required(), mcp.Description("A1 notation range to write to")),
	mcp.WithArray("values", mcp.Required(), mcp.Description("2D array of cell values")),
)

var toolSheetsAppend = mcp.NewTool(
	"sheets_append",
	mcp.WithDescription("Append rows after the last row of data in a sheet. Values are added below existing content."),
	mcp.WithString("spreadsheet_id", mcp.Description("Spreadsheet ID; uses default if omitted and configured")),
	mcp.WithString("sheet", mcp.Required(), mcp.Description("Sheet/tab name")),
	mcp.WithArray("values", mcp.Required(), mcp.Description("2D array of row values to append")),
)

var toolSheetsLookup = mcp.NewTool(
	"sheets_lookup",
	mcp.WithDescription("Find rows where a column matches a value. Returns keyed rows when a header row applies (auto-detected or set via headers), otherwise arrays."),
	mcp.WithString("spreadsheet_id", mcp.Description("Spreadsheet ID; uses default if omitted and configured")),
	mcp.WithString("sheet", mcp.Required(), mcp.Description("Sheet/tab name")),
	mcp.WithString("match_column", mcp.Required(), mcp.Description("Header name or column letter to match against")),
	mcp.WithString("match_value", mcp.Required(), mcp.Description("Value to search for")),
	mcp.WithString("match_mode", mcp.Description("Match strategy: exact, contains, or prefix"), mcp.Enum("exact", "contains", "prefix")),
	mcp.WithArray("return_columns", mcp.Description("Header names of columns to include in results; omit for all columns")),
	mcp.WithBoolean("headers", mcp.Description("Override header auto-detection: true = treat row 1 as headers, false = treat row 1 as data (columns labeled A/B/C/...), omit = auto-detect")),
)

var toolSheetsClear = mcp.NewTool(
	"sheets_clear",
	mcp.WithDescription("Clear all values in a range without deleting cells or structure."),
	mcp.WithString("spreadsheet_id", mcp.Description("Spreadsheet ID; uses default if omitted and configured")),
	mcp.WithString("sheet", mcp.Required(), mcp.Description("Sheet/tab name")),
	mcp.WithString("range", mcp.Required(), mcp.Description("A1 notation range to clear")),
)
