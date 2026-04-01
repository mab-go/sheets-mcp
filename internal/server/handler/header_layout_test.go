package handler

import (
	"reflect"
	"testing"
)

func TestHeaderLayoutForValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		values       [][]any
		hasOverride  bool
		override     bool
		wantDetected bool
		wantHeaders  []string
		wantData     [][]any
	}{
		{
			name:         "empty",
			values:       [][]any{},
			wantDetected: false,
			wantHeaders:  []string{},
			wantData:     [][]any{},
		},
		{
			name: "override_true",
			values: [][]any{
				{"Name", "Qty"},
				{"x", float64(1)},
			},
			hasOverride:  true,
			override:     true,
			wantDetected: true,
			wantHeaders:  []string{"Name", "Qty"},
			wantData:     [][]any{{"x", float64(1)}},
		},
		{
			name: "override_false",
			values: [][]any{
				{"a", "b"},
				{"1", "2"},
			},
			hasOverride:  true,
			override:     false,
			wantDetected: false,
			wantHeaders:  []string{"A", "B"},
			wantData:     [][]any{{"a", "b"}, {"1", "2"}},
		},
		{
			name: "auto_mixed_types_row2",
			values: [][]any{
				{"Name", "Qty"},
				{"x", float64(1)},
			},
			hasOverride:  false,
			wantDetected: true,
			wantHeaders:  []string{"Name", "Qty"},
			wantData:     [][]any{{"x", float64(1)}},
		},
		{
			name: "auto_all_strings_no_header_signal",
			values: [][]any{
				{"a", "b"},
				{"c", "d"},
			},
			hasOverride:  false,
			wantDetected: false,
			wantHeaders:  []string{"A", "B"},
			wantData:     [][]any{{"a", "b"}, {"c", "d"}},
		},
		{
			name: "override_true_single_data_row",
			values: [][]any{
				{"H1", "H2"},
			},
			hasOverride:  true,
			override:     true,
			wantDetected: true,
			wantHeaders:  []string{"H1", "H2"},
			wantData:     [][]any{},
		},
		{
			name: "auto_single_row_insufficient_for_detect",
			values: [][]any{
				{"only", "row"},
			},
			hasOverride:  false,
			wantDetected: false,
			wantHeaders:  []string{"A", "B"},
			wantData:     [][]any{{"only", "row"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotDetected, gotHeaders, gotData := headerLayoutForValues(tt.values, tt.hasOverride, tt.override)
			if gotDetected != tt.wantDetected {
				t.Errorf("headersDetected = %v, want %v", gotDetected, tt.wantDetected)
			}
			if !reflect.DeepEqual(gotHeaders, tt.wantHeaders) {
				t.Errorf("headers = %#v, want %#v", gotHeaders, tt.wantHeaders)
			}
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("dataRows = %#v, want %#v", gotData, tt.wantData)
			}
		})
	}
}
