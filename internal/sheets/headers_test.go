package sheets

import "testing"

func TestDetectHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values [][]any
		want   bool
	}{
		{
			name: "mixed types row2 float64",
			values: [][]any{
				{"Name", "Age"},
				{"Bob", 30.0},
			},
			want: true,
		},
		{
			name: "mixed types row2 bool",
			values: [][]any{
				{"A", "B"},
				{"x", true},
			},
			want: true,
		},
		{
			name: "all string rows no header signal",
			values: [][]any{
				{"a", "b"},
				{"c", "d"},
			},
			want: false,
		},
		{
			name: "middle empty in row1",
			values: [][]any{
				{"A", "", "B"},
				{"x", "y", "z"},
			},
			want: false,
		},
		{
			name: "trailing empty row1 not middle empty mixed row2",
			values: [][]any{
				{"A", "B", ""},
				{"x", 1.0, ""},
			},
			want: true,
		},
		{
			name: "trailing empty row1 all string row2",
			values: [][]any{
				{"A", "B", ""},
				{"x", "y", ""},
			},
			want: false,
		},
		{
			name: "row1 float64",
			values: [][]any{
				{1.0, "a"},
				{"2", "3"},
			},
			want: false,
		},
		{
			name: "row1 bool",
			values: [][]any{
				{true, "a"},
				{"2", "3"},
			},
			want: false,
		},
		{
			name: "single row",
			values: [][]any{
				{"a", "b"},
			},
			want: false,
		},
		{
			name: "empty row1",
			values: [][]any{
				{},
				{"a"},
			},
			want: false,
		},
		{
			name: "row2 all nil cells skipped no mixed types",
			values: [][]any{
				{"Name", "Age"},
				{nil, nil},
			},
			want: false,
		},
		{
			name: "leading empty row1 not middle empty all string body",
			values: [][]any{
				{"", "A", "B"},
				{"1", "2", "3"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DetectHeaders(tt.values); got != tt.want {
				t.Errorf("DetectHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasMiddleEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  []any
		want bool
	}{
		{"A empty B", []any{"A", "", "B"}, true},
		{"A B trailing empty", []any{"A", "B", ""}, false},
		{"leading empty A B", []any{"", "A", "B"}, false},
		{"no gap", []any{"A", "B"}, false},
		{"A empty empty B", []any{"A", "", "", "B"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasMiddleEmpty(tt.row); got != tt.want {
				t.Errorf("hasMiddleEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
