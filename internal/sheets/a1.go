package sheets

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func isNamedRangeByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9') || b == '_'
}

func isNamedRangeFirstByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_'
}

const maxNamedRangeLen = 255

// QuoteSheetNameForA1 returns a sheet title quoted for use in A1 notation
// (e.g. 'My Sheet'!A1). Internal single quotes are doubled per Sheets rules.
func QuoteSheetNameForA1(name string) string {
	var b strings.Builder
	b.Grow(len(name) + 2)
	b.WriteByte('\'')
	for i := range len(name) {
		if name[i] == '\'' {
			b.WriteString("''")
		} else {
			b.WriteByte(name[i])
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// ValidateA1RangeFragment checks the range part after sheet! for Values API calls.
// It accepts common A1 cell/column/row ranges, a single-segment named-range-style
// identifier, and rejects obvious mistakes (!, ;, control characters, malformed colons).
func ValidateA1RangeFragment(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("range is empty after trimming whitespace")
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return errors.New("range must not contain control characters or newlines")
		}
	}
	if strings.ContainsAny(s, "!;") {
		return errors.New("invalid range: use the sheet parameter for the tab name, not ! or ; inside range")
	}
	if utf8.RuneCountInString(s) > maxNamedRangeLen {
		return fmt.Errorf("range exceeds maximum length (%d)", maxNamedRangeLen)
	}

	if strings.Contains(s, ":") {
		if strings.Count(s, ":") != 1 {
			return errors.New("invalid range: expected a single ':' for A1 ranges (e.g. A1:B2 or A:A)")
		}
		left, right, _ := strings.Cut(s, ":")
		left, right = strings.TrimSpace(left), strings.TrimSpace(right)
		if left == "" || right == "" {
			return errors.New("invalid range: empty side of ':' (e.g. avoid 'A1:' or ':B2')")
		}
		lCell, rCell := isA1CellRef(left), isA1CellRef(right)
		lCol, rCol := isA1ColumnRef(left), isA1ColumnRef(right)
		lRow, rRow := isA1RowRef(left), isA1RowRef(right)
		switch {
		case lCell && rCell:
			return nil
		case lCol && rCol:
			return nil
		case lRow && rRow:
			return nil
		default:
			return errors.New("invalid range: both sides of ':' must be cells (A1:B2), columns (A:B), or rows (1:2)")
		}
	}

	if isA1CellRef(s) {
		return nil
	}
	if isNamedRangeToken(s) {
		return nil
	}
	return errors.New("invalid range: expected A1 notation (e.g. A1, A1:B10, A:A, 1:100) or a simple named range (letters, digits, underscore)")
}

func isColLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isA1CellRef(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if i < len(s) && s[i] == '$' {
		i++
	}
	startCol := i
	for i < len(s) && isColLetter(s[i]) {
		i++
	}
	if i == startCol {
		return false
	}
	if i < len(s) && s[i] == '$' {
		i++
	}
	startRow := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == startRow {
		return false
	}
	return i == len(s)
}

func isA1ColumnRef(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if i < len(s) && s[i] == '$' {
		i++
	}
	start := i
	for i < len(s) && isColLetter(s[i]) {
		i++
	}
	return i > start && i == len(s)
}

func isA1RowRef(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if i < len(s) && s[i] == '$' {
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i > start && i == len(s)
}

func isNamedRangeToken(s string) bool {
	if len(s) == 0 || utf8.RuneCountInString(s) > maxNamedRangeLen {
		return false
	}
	if !isNamedRangeFirstByte(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isNamedRangeByte(s[i]) {
			return false
		}
	}
	return true
}
