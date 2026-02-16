package cli

import "fmt"

// Section spacing for CLI output
const (
	// SectionBreak adds visual separation between major sections
	SectionBreak = "\n"

	// SubsectionBreak adds minimal separation within sections
	SubsectionBreak = ""
)

// PrintSectionBreak prints consistent spacing between sections
func PrintSectionBreak() {
	fmt.Println()
}

// PrintSubsectionBreak prints minimal spacing within sections
func PrintSubsectionBreak() {
	// intentionally no-op for now, but provides hook for future changes
}
