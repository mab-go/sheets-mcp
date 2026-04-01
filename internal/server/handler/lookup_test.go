package handler

import (
	"strings"
	"testing"
)

func TestMatchCell_Exact(t *testing.T) {
	tests := []struct {
		cell, target string
		want         bool
	}{
		{"alice", "alice", true},
		{"alice", "alic", false},
		{"alice", "alice smith", false},
		{"", "", true},
		{"alice", "", false},
	}

	for _, tt := range tests {
		got := matchCell(tt.cell, tt.target, "exact")
		if got != tt.want {
			t.Errorf("matchCell(%q, %q, exact) = %v, want %v", tt.cell, tt.target, got, tt.want)
		}
	}
}

func TestMatchCell_Contains(t *testing.T) {
	tests := []struct {
		cell, target string
		want         bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "xyz", false},
		{"hello world", "", true},
		{"", "hello", false},
	}

	for _, tt := range tests {
		got := matchCell(tt.cell, tt.target, "contains")
		if got != tt.want {
			t.Errorf("matchCell(%q, %q, contains) = %v, want %v", tt.cell, tt.target, got, tt.want)
		}
	}
}

func TestMatchCell_Prefix(t *testing.T) {
	tests := []struct {
		cell, target string
		want         bool
	}{
		{"hello world", "hello", true},
		{"hello world", "world", false},
		{"hello world", "hello world", true},
		{"hello world", "", true},
		{"", "hello", false},
	}

	for _, tt := range tests {
		got := matchCell(tt.cell, tt.target, "prefix")
		if got != tt.want {
			t.Errorf("matchCell(%q, %q, prefix) = %v, want %v", tt.cell, tt.target, got, tt.want)
		}
	}
}

func TestFilterRows(t *testing.T) {
	rows := [][]any{
		{"Alice", float64(30), "Calgary"},
		{"Bob", float64(25), "Edmonton"},
		{"Carol", float64(35), "Calgary"},
		{"Dave", float64(28), "Red Deer"},
	}

	// Contains match on city column (index 2).
	matched := filterRows(rows, 2, "Calgary", "contains")
	if len(matched) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matched))
	}
	if matched[0][0] != "Alice" || matched[1][0] != "Carol" {
		t.Errorf("unexpected matches: %v", matched)
	}

	// Exact match.
	matched = filterRows(rows, 2, "calgary", "exact")
	if len(matched) != 2 {
		t.Fatalf("expected 2 matches (case-insensitive), got %d", len(matched))
	}

	// Prefix match on name column (index 0).
	matched = filterRows(rows, 0, "Ca", "prefix")
	if len(matched) != 1 || matched[0][0] != "Carol" {
		t.Errorf("expected Carol for prefix 'Ca', got %v", matched)
	}

	// No matches.
	matched = filterRows(rows, 2, "Vancouver", "exact")
	if len(matched) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matched))
	}
}

func TestFilterRows_ShortRows(t *testing.T) {
	rows := [][]any{
		{"Alice", "Calgary"},
		{"Bob"}, // Short row — missing city.
	}

	matched := filterRows(rows, 1, "Calgary", "exact")
	if len(matched) != 1 || matched[0][0] != "Alice" {
		t.Errorf("expected Alice, got %v", matched)
	}
}

func TestFindColumnIndex_HeaderMatch(t *testing.T) {
	headers := []string{"Name", "Age", "City"}

	tests := []struct {
		name string
		want int
	}{
		{"Name", 0},
		{"name", 0},
		{"NAME", 0},
		{"City", 2},
		{"Missing", -1},
	}

	for _, tt := range tests {
		got := findColumnIndex(headers, tt.name)
		if got != tt.want {
			t.Errorf("findColumnIndex(headers, %q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestFindColumnIndex_ColumnLetter(t *testing.T) {
	headers := []string{"A", "B", "C", "D"}

	// When headers are letters themselves, the name match takes priority.
	// But let's test with generated-style headers.
	genHeaders := generateColumnHeaders(30)

	tests := []struct {
		letter string
		want   int
	}{
		{"A", 0},
		{"B", 1},
		{"Z", 25},
		{"AA", 26},
		{"AD", 29},
	}

	for _, tt := range tests {
		got := findColumnIndex(genHeaders, tt.letter)
		if got != tt.want {
			t.Errorf("findColumnIndex(genHeaders, %q) = %d, want %d", tt.letter, got, tt.want)
		}
	}

	_ = headers // Avoid unused.
}

func TestColumnLetterToIndex(t *testing.T) {
	tests := []struct {
		letter string
		want   int
	}{
		{"A", 0},
		{"B", 1},
		{"Z", 25},
		{"AA", 26},
		{"AZ", 51},
		{"BA", 52},
		{"", -1},
		{"1", -1},
		{"A1", -1},
	}

	for _, tt := range tests {
		got := columnLetterToIndex(tt.letter)
		if got != tt.want {
			t.Errorf("columnLetterToIndex(%q) = %d, want %d", tt.letter, got, tt.want)
		}
	}
}

func TestExtractValues(t *testing.T) {
	// Simulate MCP JSON-decoded values parameter.
	args := map[string]any{
		"values": []interface{}{
			[]interface{}{"Alice", float64(30)},
			[]interface{}{"Bob", float64(25)},
		},
	}

	rows, err := extractValues(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "Alice" || rows[1][0] != "Bob" {
		t.Errorf("unexpected values: %v", rows)
	}
}

func TestExtractValues_Missing(t *testing.T) {
	_, err := extractValues(map[string]any{})
	if err == nil {
		t.Error("expected error for missing values")
	}
}

func TestExtractValues_NotArray(t *testing.T) {
	_, err := extractValues(map[string]any{"values": "not an array"})
	if err == nil {
		t.Error("expected error for non-array values")
	}
}

func TestExtractValues_InnerNotArray(t *testing.T) {
	args := map[string]any{
		"values": []interface{}{"not an inner array"},
	}
	_, err := extractValues(args)
	if err == nil {
		t.Error("expected error for non-array inner value")
	}
}

func TestExtractValues_Empty(t *testing.T) {
	args := map[string]any{
		"values": []interface{}{},
	}
	_, err := extractValues(args)
	if err == nil {
		t.Error("expected error for empty values")
	}
}

func TestExtractStringSlice(t *testing.T) {
	// []interface{} (typical MCP JSON).
	result := extractStringSlice([]interface{}{"A", "B", "C"})
	if len(result) != 3 || result[0] != "A" {
		t.Errorf("expected [A B C], got %v", result)
	}

	// Single string.
	result = extractStringSlice("Name")
	if len(result) != 1 || result[0] != "Name" {
		t.Errorf("expected [Name], got %v", result)
	}

	// nil.
	result = extractStringSlice(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestFilterColumns(t *testing.T) {
	headers := []string{"Name", "Age", "City", "Email"}
	rows := [][]any{
		{"Alice", float64(30), "Calgary", "alice@example.com"},
		{"Bob", float64(25), "Edmonton", "bob@example.com"},
	}

	filteredHeaders, filteredRows, err := filterColumns(headers, rows, []interface{}{"Name", "City"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filteredHeaders) != 2 || filteredHeaders[0] != "Name" || filteredHeaders[1] != "City" {
		t.Errorf("expected [Name City], got %v", filteredHeaders)
	}
	if len(filteredRows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(filteredRows))
	}
	if filteredRows[0][0] != "Alice" || filteredRows[0][1] != "Calgary" {
		t.Errorf("unexpected filtered row: %v", filteredRows[0])
	}
}

func TestFilterColumns_NoneMatch(t *testing.T) {
	headers := []string{"Name", "Age"}
	rows := [][]any{{"Alice", float64(30)}}

	_, _, err := filterColumns(headers, rows, []interface{}{"Nope", "AlsoBad"})
	if err == nil {
		t.Fatal("expected error when no return_columns match")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Name") || !strings.Contains(msg, "Age") {
		t.Errorf("error should list available columns: %q", msg)
	}
	if !strings.Contains(msg, "Nope") || !strings.Contains(msg, "AlsoBad") {
		t.Errorf("error should mention requested names: %q", msg)
	}
}
