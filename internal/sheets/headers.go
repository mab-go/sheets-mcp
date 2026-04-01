package sheets

import "reflect"

// DetectHeaders determines whether the first row of data looks like a header
// row. The heuristic:
//
// Strong signals (row 1 is headers):
//   - Row 1 is a contiguous span of non-empty strings (leading/trailing empty
//     cells are allowed for unused columns)
//   - Row 2+ contains mixed types (numbers, booleans alongside strings)
//
// Anti-signals (row 1 is data):
//   - Row 1 contains numbers, booleans, or dates (non-string cells)
//   - Row 1 has empty cells between two non-empty cells (see hasMiddleEmpty)
//
// If row 1 qualifies but row 2+ is only strings (nil cells skipped), mixed
// types are not present — returns false. Callers with text-only header rows
// should set headers: true on sheets_get.
//
// Returns false if there are fewer than 2 rows (nothing to compare against).
func DetectHeaders(values [][]any) bool {
	if len(values) < 2 {
		return false
	}

	firstRow := values[0]
	if len(firstRow) == 0 {
		return false
	}

	if !firstRowHeaderSpanOK(firstRow) {
		return false
	}
	// firstRowHeaderSpanOK implies !hasMiddleEmpty (no empty between first/last non-empty).

	// Strong signal: data rows (row 2+) contain a non-string type.
	for _, row := range values[1:] {
		for _, cell := range row {
			if cell == nil {
				continue
			}
			if reflect.TypeOf(cell).Kind() != reflect.String {
				return true
			}
		}
	}

	return false
}

// firstRowHeaderSpanOK reports whether row 1 has at least one non-empty string
// and every cell from the first through last non-empty string (inclusive) is a
// non-empty string. Leading and trailing empty cells (nil or "") are allowed.
// Any non-string cell (e.g. float64, bool) fails.
func firstRowHeaderSpanOK(row []any) bool {
	first := -1
	last := -1
	for i, cell := range row {
		if cell == nil {
			continue
		}
		s, ok := cell.(string)
		if !ok {
			return false
		}
		if s != "" {
			if first == -1 {
				first = i
			}
			last = i
		}
	}
	if first == -1 {
		return false
	}
	for i := first; i <= last; i++ {
		cell := row[i]
		if cell == nil {
			return false
		}
		s, ok := cell.(string)
		if !ok || s == "" {
			return false
		}
	}
	return true
}

// hasMiddleEmpty reports whether there exist indices i < j < k such that
// row[i] and row[k] are non-empty strings and row[j] is empty (nil or "").
// Trailing empties after the last non-empty cell do not count; leading
// empties before the first non-empty are not treated as a gap.
func hasMiddleEmpty(row []any) bool {
	for i := range row {
		if !isNonEmptyString(row[i]) {
			continue
		}
		for k := i + 2; k < len(row); k++ {
			if !isNonEmptyString(row[k]) {
				continue
			}
			for j := i + 1; j < k; j++ {
				if isEmptyCell(row[j]) {
					return true
				}
			}
		}
	}
	return false
}

func isEmptyCell(cell any) bool {
	if cell == nil {
		return true
	}
	s, ok := cell.(string)
	if !ok {
		return false
	}
	return s == ""
}

func isNonEmptyString(cell any) bool {
	s, ok := cell.(string)
	return ok && s != ""
}
