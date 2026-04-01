package sheets

import (
	"strings"
	"testing"
)

func TestValidateA1RangeFragment(t *testing.T) {
	t.Parallel()
	valid := []string{
		"A1",
		"$B$2",
		"a1",
		"A1:Z99",
		"$A$1:$ZZ$999",
		"A:A",
		"AA:ZZ",
		"1:1",
		"10:20",
		"SalesData",
		"Q1_Sales",
		"A",
	}
	for _, s := range valid {
		t.Run("ok_"+s, func(t *testing.T) {
			t.Parallel()
			if err := ValidateA1RangeFragment(s); err != nil {
				t.Errorf("ValidateA1RangeFragment(%q) = %v, want nil", s, err)
			}
		})
	}

	invalid := []struct {
		in string
	}{
		{""},
		{"   "},
		{"Sheet1!A1"},
		{"A1;B1"},
		{"A1\nB1"},
		{"A1:"},
		{":B2"},
		{"!A1"},
		{"A1:B2:C3"},
	}
	for _, tt := range invalid {
		t.Run("bad_"+strings.ReplaceAll(strings.ReplaceAll(tt.in, "\n", "newline"), " ", "space"), func(t *testing.T) {
			t.Parallel()
			if err := ValidateA1RangeFragment(tt.in); err == nil {
				t.Errorf("ValidateA1RangeFragment(%q) = nil, want error", tt.in)
			}
		})
	}
}

func TestQuoteSheetNameForA1(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "Sheet1", "'Sheet1'"},
		{"spaces", "My Sheet", "'My Sheet'"},
		{"embedded_quote", "O'Brien", "'O''Brien'"},
		{"empty", "", "''"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := QuoteSheetNameForA1(tt.in)
			if got != tt.want {
				t.Errorf("QuoteSheetNameForA1(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
