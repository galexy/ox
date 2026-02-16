package main

import (
	"fmt"
	"sort"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/sageox/ox/internal/ui"
	"github.com/spf13/cobra"
)

// lipgloss styles for the thanks page - uses ox brand colors
var (
	thanksTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ui.ColorText)

	thanksSubtitleStyle = lipgloss.NewStyle().
				Foreground(ui.ColorTextDim)

	thanksSectionStyle = lipgloss.NewStyle().
				Foreground(ui.ColorAccent).
				Bold(true)

	thanksNameStyle = lipgloss.NewStyle().
			Foreground(ui.ColorPass)

	thanksDimStyle = lipgloss.NewStyle().
			Foreground(ui.ColorMuted)
)

// thanksBoxStyle returns a box style with dynamic width
func thanksBoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(ui.ColorAccent).
		Padding(1, 4).
		Width(width - 4). // account for border
		Align(lipgloss.Center)
}

// Static list of human contributors to the ox project.
// To update counts: git shortlog -sn --all
// Map of contributor name -> commit count, sorted by contribution count descending.
var oxContributors = map[string]int{
	"Ryan Snodgrass": 332,
	"Ajit Banerjee":  3,
}

var thanksCmd = &cobra.Command{
	Use:    "thanks",
	Short:  "Thank the human contributors to ox",
	Hidden: true, // hidden until we have at least 5 contributors
	Long:   `Display a thank you page listing all human contributors to the ox project.`,
	Run: func(cmd *cobra.Command, args []string) {
		printThanksPage()
	},
}

// getContributorsSorted returns contributors sorted by commit count descending
func getContributorsSorted() []string {
	type kv struct {
		name    string
		commits int
	}
	var sorted []kv
	for name, commits := range oxContributors {
		sorted = append(sorted, kv{name, commits})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].commits > sorted[j].commits
	})
	names := make([]string, len(sorted))
	for i, kv := range sorted {
		names[i] = kv.name
	}
	return names
}

// printThanksPage displays the thank you page
func printThanksPage() {
	fmt.Println()

	// get sorted contributors and split into top 20 and rest
	allContributors := getContributorsSorted()
	topN := 20
	if topN > len(allContributors) {
		topN = len(allContributors)
	}

	topContributors := allContributors[:topN]
	additionalContributors := allContributors[topN:]

	// calculate content width based on featured contributors columns
	contentWidth := calculateThanksColumnsWidth(topContributors, 4) + 4 // +4 for indent
	if contentWidth < 50 {
		contentWidth = 50 // minimum width for the box
	}

	// build header content with styled text
	title := thanksTitleStyle.Render("THANK YOU!")
	subtitle := thanksSubtitleStyle.Render("To all the humans who contributed to ox")
	header := title + "\n\n" + subtitle

	// render header in a bordered box matching content width
	fmt.Println(thanksBoxStyle(contentWidth).Render(header))
	fmt.Println()

	// print featured contributors section
	fmt.Println(thanksSectionStyle.Render("  Featured Contributors"))
	fmt.Println()
	printThanksColumns(topContributors, 4)

	// print additional contributors with line wrapping
	if len(additionalContributors) > 0 {
		fmt.Println()
		fmt.Println(thanksSectionStyle.Render("  Additional Contributors"))
		fmt.Println()
		printThanksWrappedList("", additionalContributors, contentWidth)
	}
	fmt.Println()
}

// calculateThanksColumnsWidth returns the total width needed for displaying names in columns
func calculateThanksColumnsWidth(names []string, cols int) int {
	if len(names) == 0 {
		return 0
	}

	maxWidth := 0
	for _, name := range names {
		if len(name) > maxWidth {
			maxWidth = len(name)
		}
	}
	if maxWidth > 20 {
		maxWidth = 20
	}
	colWidth := maxWidth + 2

	return colWidth * cols
}

// printThanksColumns prints names in n columns, sorted horizontally by input order
func printThanksColumns(names []string, cols int) {
	if len(names) == 0 {
		return
	}

	// find max width for alignment
	maxWidth := 0
	for _, name := range names {
		if len(name) > maxWidth {
			maxWidth = len(name)
		}
	}
	if maxWidth > 20 {
		maxWidth = 20
	}
	colWidth := maxWidth + 2

	// print in rows, reading left to right
	for i := 0; i < len(names); i += cols {
		fmt.Print("  ")
		for j := 0; j < cols && i+j < len(names); j++ {
			name := names[i+j]
			if len(name) > 20 {
				name = name[:17] + "..."
			}
			padded := fmt.Sprintf("%-*s", colWidth, name)
			fmt.Print(thanksNameStyle.Render(padded))
		}
		fmt.Println()
	}
}

// printThanksWrappedList prints a list with word wrapping at name boundaries
func printThanksWrappedList(label string, names []string, maxWidth int) {
	indent := "  "

	fmt.Print(indent)
	lineLen := len(indent)

	if label != "" {
		thanksLabelStyle := lipgloss.NewStyle().Foreground(ui.ColorWarn)
		fmt.Print(thanksLabelStyle.Render(label) + " ")
		lineLen += len(label) + 1
	}

	for i, name := range names {
		suffix := ", "
		if i == len(names)-1 {
			suffix = ""
		}
		entry := name + suffix

		if lineLen+len(entry) > maxWidth && lineLen > len(indent) {
			fmt.Println()
			fmt.Print(indent)
			lineLen = len(indent)
		}

		fmt.Print(thanksDimStyle.Render(entry))
		lineLen += len(entry)
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(thanksCmd)
}
