package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColumnWidths(t *testing.T) {
	tests := []struct {
		name       string
		rows       [][]string
		minWidths  []int
		maxWidths  []int
		wantWidths []int
	}{
		{
			name: "basic calculation",
			rows: [][]string{
				{"ID", "Name", "Status"},
				{"1", "Alice", "Active"},
				{"22", "Bob", "Inactive"},
			},
			minWidths:  []int{2, 4, 6},
			maxWidths:  []int{10, 20, 10},
			wantWidths: []int{2, 5, 8},
		},
		{
			name: "respect minimum widths",
			rows: [][]string{
				{"A", "B", "C"},
				{"1", "2", "3"},
			},
			minWidths:  []int{5, 10, 15},
			maxWidths:  []int{20, 30, 40},
			wantWidths: []int{5, 10, 15},
		},
		{
			name: "respect maximum widths",
			rows: [][]string{
				{"ID", "Very Long Column Name Here", "Status"},
				{"1", "Some text", "OK"},
			},
			minWidths:  []int{2, 4, 4},
			maxWidths:  []int{10, 15, 10},
			wantWidths: []int{2, 15, 6},
		},
		{
			name:       "empty rows",
			rows:       [][]string{},
			minWidths:  []int{5, 10, 15},
			maxWidths:  []int{20, 30, 40},
			wantWidths: []int{5, 10, 15},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWidths := ColumnWidths(tt.rows, tt.minWidths, tt.maxWidths)
			require.Len(t, gotWidths, len(tt.wantWidths), "ColumnWidths() returned wrong number of widths")
			for i := range gotWidths {
				assert.Equal(t, tt.wantWidths[i], gotWidths[i], "ColumnWidths()[%d]", i)
			}
		})
	}
}

func TestFormatRow(t *testing.T) {
	tests := []struct {
		name   string
		row    []string
		widths []int
		want   string
	}{
		{
			name:   "basic formatting",
			row:    []string{"ID", "Name", "Status"},
			widths: []int{5, 10, 8},
			want:   "ID    Name       Status  ",
		},
		{
			name:   "handles longer content",
			row:    []string{"12345", "Alice", "Active"},
			widths: []int{5, 10, 8},
			want:   "12345 Alice      Active  ",
		},
		{
			name:   "content exceeds width gets truncated padding",
			row:    []string{"VeryLongID", "Bob", "OK"},
			widths: []int{5, 10, 8},
			want:   "VeryLongID Bob        OK      ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRow(tt.row, tt.widths)
			// normalize spaces for comparison
			gotNorm := strings.TrimRight(got, " ")
			wantNorm := strings.TrimRight(tt.want, " ")

			// check that column spacing is preserved
			assert.Equal(t, tt.want, got, "FormatRow()\n  normalized got  = %q\n  normalized want = %q", gotNorm, wantNorm)
		})
	}
}
