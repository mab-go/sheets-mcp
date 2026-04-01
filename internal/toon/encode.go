// Package toon provides TOON tabular encoding for MCP tool responses.
//
// All tool responses use TOON (Token-Oriented Object Notation) with key-value
// metadata at the top and tabular data below. This package wraps toon-go and
// handles any edge cases specific to Google Sheets data.
package toon

import (
	"fmt"
	"strings"

	toonlib "github.com/toon-format/toon-go"
)

// EncodeToolResponse produces a complete TOON-encoded tool response with
// key-value metadata lines at the top and tabular data below.
//
// The metadata fields are emitted in the order provided. The rows field is
// appended as a tabular array with the given headers.
//
// If headers is nil, no tabular section is emitted (metadata only). To emit a
// rows key with zero columns and zero data rows (e.g. empty read or no lookup
// matches), pass an empty slice []string{}, not nil.
func EncodeToolResponse(metadata []Field, headers []string, rows [][]any) (string, error) {
	fields := make([]toonlib.Field, 0, len(metadata)+1)
	for _, f := range metadata {
		fields = append(fields, toonlib.Field{Key: f.Key, Value: f.Value})
	}

	if headers != nil {
		toonRows := buildTabularRows(headers, rows)
		fields = append(fields, toonlib.Field{Key: "rows", Value: toonRows})
	}

	obj := toonlib.NewObject(fields...)
	return toonlib.MarshalString(obj)
}

// EncodeKeyValue produces a TOON-encoded response with only key-value
// metadata (no tabular data).
func EncodeKeyValue(metadata []Field) (string, error) {
	fields := make([]toonlib.Field, len(metadata))
	for i, f := range metadata {
		fields[i] = toonlib.Field{Key: f.Key, Value: f.Value}
	}

	obj := toonlib.NewObject(fields...)
	return toonlib.MarshalString(obj)
}

// Field is a key-value pair for metadata in tool responses. Defined here to
// avoid leaking toon-go types into handler code.
type Field struct {
	Key   string
	Value any
}

// buildTabularRows converts raw row data into toon.Object slices that the
// toon-go encoder will detect as tabular. Each cell value is sanitized to
// ensure correct TOON encoding.
func buildTabularRows(headers []string, rows [][]any) []toonlib.Object {
	result := make([]toonlib.Object, len(rows))
	for i, row := range rows {
		fields := make([]toonlib.Field, len(headers))
		for j, header := range headers {
			var val any
			if j < len(row) {
				val = sanitizeValue(row[j])
			}
			fields[j] = toonlib.Field{Key: header, Value: val}
		}
		result[i] = toonlib.NewObject(fields...)
	}
	return result
}

// sanitizeValue ensures a cell value will encode correctly in TOON tabular
// format. Specifically it handles:
//   - Strings that look like booleans ("true", "false") — must remain strings
//   - Strings that look like numbers ("4035551234") — must remain strings
//   - nil values — passed through as nil (encodes as null)
//   - Empty strings — passed through (encodes as empty between delimiters)
//
// The toon-go library handles comma-containing strings and unicode correctly
// via its quoting logic. Boolean-like and number-like strings are already Go
// strings, so toon-go encodes them as quoted strings — no workaround needed
// as long as the values arrive here with correct Go types.
func sanitizeValue(v any) any {
	if v == nil {
		return nil
	}

	// Google Sheets API returns float64 for all numbers. Check if a float64
	// is actually an integer and convert to int for cleaner TOON output.
	if f, ok := v.(float64); ok {
		if f == float64(int64(f)) && f >= -1e15 && f <= 1e15 {
			return int64(f)
		}
	}

	return v
}

// EncodeList produces a TOON-encoded response with metadata and a named list
// of objects (not tabular — each item has its own set of fields). Useful for
// sheets_find and sheets_info where rows aren't uniform tabular data.
func EncodeList(metadata []Field, listKey string, items [][]Field) (string, error) {
	fields := make([]toonlib.Field, 0, len(metadata)+1)
	for _, f := range metadata {
		fields = append(fields, toonlib.Field{Key: f.Key, Value: f.Value})
	}

	if items != nil {
		toonItems := make([]toonlib.Object, len(items))
		for i, item := range items {
			itemFields := make([]toonlib.Field, len(item))
			for j, f := range item {
				itemFields[j] = toonlib.Field{Key: f.Key, Value: f.Value}
			}
			toonItems[i] = toonlib.NewObject(itemFields...)
		}
		fields = append(fields, toonlib.Field{Key: listKey, Value: toonItems})
	}

	obj := toonlib.NewObject(fields...)
	return toonlib.MarshalString(obj)
}

// FormatError produces a simple TOON-encoded error response.
func FormatError(errMsg string) string {
	// Errors are simple enough to format directly.
	return fmt.Sprintf("error: %s", escapeTOONValue(errMsg))
}

// escapeTOONValue escapes a string value for use in a TOON key-value line.
func escapeTOONValue(s string) string {
	if strings.ContainsAny(s, "\n\r") {
		// Multi-line values need quoting.
		return fmt.Sprintf("%q", s)
	}
	return s
}
