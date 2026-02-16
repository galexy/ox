package cli

import (
	"fmt"
	"strings"
)

// ColumnWidths calculates optimal widths for columns based on content
func ColumnWidths(rows [][]string, minWidths []int, maxWidths []int) []int {
	if len(rows) == 0 {
		return minWidths
	}
	widths := make([]int, len(rows[0]))

	// find max width for each column
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// apply min/max constraints
	for i := range widths {
		if i < len(minWidths) && widths[i] < minWidths[i] {
			widths[i] = minWidths[i]
		}
		if i < len(maxWidths) && maxWidths[i] > 0 && widths[i] > maxWidths[i] {
			widths[i] = maxWidths[i]
		}
	}

	return widths
}

// FormatRow formats a row with dynamic widths
func FormatRow(row []string, widths []int) string {
	var parts []string
	for i, cell := range row {
		width := 10 // default
		if i < len(widths) {
			width = widths[i]
		}
		parts = append(parts, fmt.Sprintf("%-*s", width, cell))
	}
	return strings.Join(parts, " ")
}
