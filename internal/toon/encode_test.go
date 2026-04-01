package toon

import (
	"strings"
	"testing"
)

func TestEncodeToolResponse_BasicTabular(t *testing.T) {
	metadata := []Field{
		{Key: "spreadsheet_id", Value: "abc123"},
		{Key: "sheet", Value: "Rentals"},
		{Key: "range", Value: "A1:D3"},
		{Key: "headers_detected", Value: true},
	}
	headers := []string{"Name", "Price", "Status"}
	rows := [][]any{
		{"Alice", int64(1450), "Active"},
		{"Bob", int64(1800), "Pending"},
	}

	result, err := EncodeToolResponse(metadata, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify metadata lines are present.
	assertContains(t, result, "spreadsheet_id: abc123")
	assertContains(t, result, "sheet: Rentals")
	// toon-go quotes values containing colons to avoid ambiguity with
	// key-value syntax, so "A1:D3" becomes quoted in the output.
	assertContains(t, result, "A1:D3")
	assertContains(t, result, "headers_detected: true")

	// Verify tabular header.
	assertContains(t, result, "Name")
	assertContains(t, result, "Price")
	assertContains(t, result, "Status")

	// Verify row data is present.
	assertContains(t, result, "Alice")
	assertContains(t, result, "1450")
	assertContains(t, result, "Active")
	assertContains(t, result, "Bob")
	assertContains(t, result, "1800")
	assertContains(t, result, "Pending")
}

func TestEncodeToolResponse_StringsWithCommas(t *testing.T) {
	headers := []string{"Address", "City"}
	rows := [][]any{
		{"123 4th Ave SW, Unit 2", "Calgary"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The address contains a comma and must be quoted so it doesn't split
	// across columns.
	assertContains(t, result, "123 4th Ave SW, Unit 2")
	assertContains(t, result, "Calgary")

	// Verify the comma-containing value is quoted (surrounded by quotes).
	if !strings.Contains(result, `"123 4th Ave SW, Unit 2"`) {
		t.Errorf("comma-containing string should be quoted, got:\n%s", result)
	}
}

func TestEncodeToolResponse_EmptyCells(t *testing.T) {
	headers := []string{"A", "B", "C"}
	rows := [][]any{
		{"first", "", "third"},
		{"", "middle", ""},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty cells must not collapse — each row must have the right number
	// of delimiters (two commas per row for three columns).
	data := tabularDataLines(t, result)
	if len(data) != 2 {
		t.Fatalf("want 2 data lines, got %d: %v", len(data), data)
	}
	assertTabularRowCommaCount(t, data[0], 2)
	assertTabularRowCommaCount(t, data[1], 2)
	// Ground-truth shape from toon-go (two spaces stripped by tabularDataLines).
	if data[0] != `first,"",third` {
		t.Errorf("row 0: want first,\"\",third, got %q", data[0])
	}
	if data[1] != `"",middle,""` {
		t.Errorf("row 1: want \"\",middle,\"\", got %q", data[1])
	}
}

func TestEncodeToolResponse_NewlinesInCells(t *testing.T) {
	headers := []string{"Note", "Other"}
	rows := [][]any{
		{"line1\nline2", "x"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Content must remain addressable in output (quoted multiline).
	assertContains(t, result, "line1")
	assertContains(t, result, "line2")

	data := tabularDataLines(t, result)
	if len(data) != 1 {
		t.Fatalf("want 1 data line for 1 row, got %d: %v", len(data), data)
	}
}

func TestEncodeToolResponse_NewlinesInCellsCRLF(t *testing.T) {
	headers := []string{"Col"}
	rows := [][]any{
		{"a\r\nb"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, result, "a")
	assertContains(t, result, "b")

	data := tabularDataLines(t, result)
	if len(data) != 1 {
		t.Fatalf("want 1 data line, got %d: %v", len(data), data)
	}
}

func TestEncodeToolResponse_QuotesAndBackslashes(t *testing.T) {
	headers := []string{"Q", "Path"}
	rows := [][]any{
		{`He said "hi"`, `C:\path\file`},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must remain distinguishable as cell content (quoted in TOON).
	if !strings.Contains(result, "hi") {
		t.Errorf("expected hi in output:\n%s", result)
	}
	if !strings.Contains(result, "path") {
		t.Errorf("expected path segment in output:\n%s", result)
	}
	data := tabularDataLines(t, result)
	if len(data) != 1 {
		t.Fatalf("want 1 data line, got %d", len(data))
	}
	line := data[0]
	if !strings.Contains(line, "He said") {
		t.Errorf("expected quoted speech in line: %q", line)
	}
}

func TestEncodeToolResponse_WideRowTruncated(t *testing.T) {
	headers := []string{"A", "B"}
	rows := [][]any{
		{"only", "shown", "__EXTRA_CELL__"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sec := tabularSection(result)
	if strings.Contains(sec, "__EXTRA_CELL__") {
		t.Errorf("truncated cell should not appear in tabular section:\n%s", sec)
	}
	assertContains(t, result, "only")
	assertContains(t, result, "shown")
}

func TestEncodeToolResponse_NumberLikeStrings(t *testing.T) {
	headers := []string{"Phone", "Postal"}
	rows := [][]any{
		{"4035551234", "T2P 1J9"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// These are Go strings, so they should encode as strings (not bare
	// numbers). The phone number should stay as-is.
	assertContains(t, result, "4035551234")
	assertContains(t, result, "T2P 1J9")
}

func TestEncodeToolResponse_BooleanLikeStrings(t *testing.T) {
	headers := []string{"Label", "Value"}
	rows := [][]any{
		{"active", "true"},
		{"deleted", "false"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Boolean-like strings must be distinguishable from actual booleans.
	// They should be quoted.
	assertContains(t, result, "active")
	assertContains(t, result, "deleted")

	// Verify the string "true" is quoted to distinguish from boolean true.
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "active") {
			if !strings.Contains(trimmed, `"true"`) {
				t.Errorf("boolean-like string 'true' should be quoted, got line: %s", trimmed)
			}
		}
		if strings.Contains(trimmed, "deleted") {
			if !strings.Contains(trimmed, `"false"`) {
				t.Errorf("boolean-like string 'false' should be quoted, got line: %s", trimmed)
			}
		}
	}
}

func TestEncodeToolResponse_NullValues(t *testing.T) {
	headers := []string{"Name", "Notes"}
	rows := [][]any{
		{"Alice", nil},
		{nil, "some note"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nil values should encode as null in TOON.
	assertContains(t, result, "null")
	assertContains(t, result, "Alice")
	assertContains(t, result, "some note")
}

func TestEncodeToolResponse_UnicodeContent(t *testing.T) {
	headers := []string{"Emoji", "CJK", "Accented"}
	rows := [][]any{
		{"🏠 Home", "日本語", "café résumé"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "🏠 Home")
	assertContains(t, result, "日本語")
	assertContains(t, result, "café résumé")
}

func TestEncodeToolResponse_EmptyRows(t *testing.T) {
	metadata := []Field{
		{Key: "spreadsheet_id", Value: "abc123"},
		{Key: "headers_detected", Value: false},
	}
	headers := []string{"A", "B"}
	rows := [][]any{}

	result, err := EncodeToolResponse(metadata, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "spreadsheet_id: abc123")
	// Empty rows array should still be valid TOON.
	assertContains(t, result, "rows")
}

func TestEncodeToolResponse_EmptyHeadersEmptyRows(t *testing.T) {
	metadata := []Field{
		{Key: "spreadsheet_id", Value: "abc123"},
		{Key: "sheet", Value: "Tab"},
		{Key: "range", Value: "A1:A1"},
		{Key: "headers_detected", Value: false},
		{Key: "row_count", Value: 0},
	}
	result, err := EncodeToolResponse(metadata, []string{}, [][]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Tabular block with zero rows (zero columns); must not match bare "rows" in metadata keys.
	if !strings.Contains(result, "rows[0]:") {
		t.Fatalf("expected rows tabular header line rows[0]:, got:\n%s", result)
	}
}

func TestEncodeToolResponse_FloatToIntConversion(t *testing.T) {
	headers := []string{"Count", "Price"}
	rows := [][]any{
		{float64(42), float64(19.99)},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Integer-valued floats should be converted to ints for cleaner output.
	assertContains(t, result, "42")
	assertContains(t, result, "19.99")

	// 42 should appear without a decimal point.
	if strings.Contains(result, "42.0") {
		t.Errorf("integer float should not have decimal, got:\n%s", result)
	}
}

func TestEncodeToolResponse_ActualBooleans(t *testing.T) {
	headers := []string{"Name", "Active"}
	rows := [][]any{
		{"Alice", true},
		{"Bob", false},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "Alice")
	assertContains(t, result, "Bob")
	// Actual booleans should be unquoted.
	assertContains(t, result, "true")
	assertContains(t, result, "false")
}

func TestEncodeKeyValue(t *testing.T) {
	fields := []Field{
		{Key: "status", Value: "ok"},
		{Key: "count", Value: 42},
	}

	result, err := EncodeKeyValue(fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "status: ok")
	assertContains(t, result, "count: 42")
}

func TestFormatError_MultilineMessage(t *testing.T) {
	t.Run("singleLineUnchanged", func(t *testing.T) {
		out := FormatError("no breaks")
		if out != "error: no breaks" {
			t.Errorf("want error: no breaks, got %q", out)
		}
	})
	t.Run("LF", func(t *testing.T) {
		out := FormatError("first\nsecond")
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 1 {
			t.Fatalf("multiline message must be one physical line (quoted), got %d lines: %q", len(lines), out)
		}
		assertContains(t, out, "error: ")
		assertContains(t, out, "first")
		assertContains(t, out, "second")
	})
	t.Run("CRLF", func(t *testing.T) {
		out := FormatError("x\r\ny")
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 1 {
			t.Fatalf("multiline message must be one physical line (quoted), got %d lines: %q", len(lines), out)
		}
		assertContains(t, out, "x")
		assertContains(t, out, "y")
	})
	t.Run("carriageReturnOnly", func(t *testing.T) {
		out := FormatError("a\rb")
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 1 {
			t.Fatalf("message with \\r must be one physical line (quoted), got %d lines: %q", len(lines), out)
		}
	})
}

func TestEncodeList(t *testing.T) {
	metadata := []Field{
		{Key: "count", Value: 2},
	}
	items := [][]Field{
		{
			{Key: "id", Value: "abc"},
			{Key: "title", Value: "Budget"},
		},
		{
			{Key: "id", Value: "def"},
			{Key: "title", Value: "Rentals"},
		},
	}

	result, err := EncodeList(metadata, "spreadsheets", items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "count: 2")
	assertContains(t, result, "Budget")
	assertContains(t, result, "Rentals")
}

func TestEncodeToolResponse_ShortRow(t *testing.T) {
	// A row shorter than headers should fill remaining with nil.
	headers := []string{"A", "B", "C"}
	rows := [][]any{
		{"only-one"},
	}

	result, err := EncodeToolResponse(nil, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "only-one")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q, got:\n%s", needle, haystack)
	}
}

// tabularSection returns the substring of full output starting at "rows[" (for
// assertions scoped to the tabular block).
func tabularSection(full string) string {
	if i := strings.Index(full, "rows["); i >= 0 {
		return full[i:]
	}
	return ""
}

// tabularDataLines returns the indented data lines under rows[N]{...}: (trimmed
// of the two-space prefix). Fails the test if no rows header is found.
func tabularDataLines(t *testing.T, result string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "rows[") && strings.Contains(line, "]{") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		t.Fatalf("no rows[...]{...}: header in:\n%s", result)
	}
	var out []string
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(line, "  ") {
			break
		}
		out = append(out, strings.TrimPrefix(line, "  "))
	}
	return out
}

func assertTabularRowCommaCount(t *testing.T, line string, wantCommas int) {
	t.Helper()
	n := strings.Count(line, ",")
	if n != wantCommas {
		t.Errorf("expected %d commas in data line, got %d: %q", wantCommas, n, line)
	}
}
