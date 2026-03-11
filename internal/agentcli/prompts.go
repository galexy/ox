package agentcli

import (
	"fmt"
	"strings"
)

// baseInstruction is the universal output format instruction appended to all prompts.
const baseInstruction = "Output concise markdown. No code fences or preamble.\n\n"

// writeGuidelines prepends team distillation guidelines if provided.
// Guidelines come from DISTILL.md in the team context — they let teams
// customize what gets emphasized, omitted, or structured differently.
func writeGuidelines(sb *strings.Builder, guidelines string) {
	if guidelines == "" {
		return
	}
	sb.WriteString("## Team Distillation Guidelines\n\n")
	sb.WriteString(guidelines)
	if !strings.HasSuffix(guidelines, "\n") {
		sb.WriteByte('\n')
	}
	sb.WriteByte('\n')
}

// DailyPrompt builds a prompt for distilling observations into a daily memory summary.
// If guidelines is non-empty, it is prepended as team-specific distillation preferences.
func DailyPrompt(observations []string, date, guidelines string) string {
	var sb strings.Builder

	writeGuidelines(&sb, guidelines)
	sb.WriteString("Distill these team observations into a daily memory summary.\n")
	sb.WriteString("Focus on decisions, patterns, and learnings. Omit routine actions.\n")
	sb.WriteString(baseInstruction)

	fmt.Fprintf(&sb, "## Observations (%s)\n\n", date)
	for i, obs := range observations {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, obs)
	}

	return sb.String()
}

// WeeklyPrompt builds a prompt for synthesizing daily summaries into a weekly memory.
func WeeklyPrompt(dailySummaries []string, weekID, guidelines string) string {
	var sb strings.Builder

	writeGuidelines(&sb, guidelines)
	sb.WriteString("Synthesize these daily summaries into a weekly memory.\n")
	sb.WriteString("Identify themes, key decisions, and unresolved work. Compress — shorter than the combined input.\n")
	sb.WriteString(baseInstruction)

	fmt.Fprintf(&sb, "## Dailies (%s)\n\n", weekID)
	for i, summary := range dailySummaries {
		fmt.Fprintf(&sb, "### Day %d\n\n%s\n\n", i+1, summary)
	}

	return sb.String()
}

// MonthlyPrompt builds a prompt for synthesizing weekly summaries into a monthly memory.
func MonthlyPrompt(weeklySummaries []string, month, guidelines string) string {
	var sb strings.Builder

	writeGuidelines(&sb, guidelines)
	sb.WriteString("Synthesize these weekly summaries into a monthly memory.\n")
	sb.WriteString("Focus on milestones, architecture changes, and strategic direction. Omit day-to-day details.\n")
	sb.WriteString(baseInstruction)

	fmt.Fprintf(&sb, "## Weeklies (%s)\n\n", month)
	for i, summary := range weeklySummaries {
		fmt.Fprintf(&sb, "### Week %d\n\n%s\n\n", i+1, summary)
	}

	return sb.String()
}
