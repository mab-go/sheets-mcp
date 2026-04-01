// Package sheets provides a thin wrapper around the Google Sheets and Drive
// APIs for use by MCP tool handlers.
package sheets

import "time"

// SpreadsheetMetadata holds the metadata for a spreadsheet, as returned by
// GetSpreadsheetMetadata.
type SpreadsheetMetadata struct {
	SpreadsheetID string
	Title         string
	Sheets        []SheetInfo
}

// SheetInfo describes a single sheet/tab within a spreadsheet.
type SheetInfo struct {
	Title    string
	Index    int
	RowCount int
	ColCount int
	FirstRow []any
}

// RangeData holds the result of reading a range from a spreadsheet.
type RangeData struct {
	Range  string
	Values [][]any
}

// WriteResult holds the result of a write or append operation.
type WriteResult struct {
	UpdatedRange string
	UpdatedRows  int
	UpdatedCols  int
	UpdatedCells int
}

// SearchResult represents a single spreadsheet found by FindSpreadsheets.
type SearchResult struct {
	SpreadsheetID string
	Title         string
	ModifiedTime  time.Time
}
