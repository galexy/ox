package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

var (
	// semantic colors for status
	colorHealthy  = lipgloss.Color("2") // green
	colorWarning  = lipgloss.Color("3") // yellow
	colorCritical = lipgloss.Color("1") // red
	colorMuted    = lipgloss.Color("8") // gray

	// styles
	styleHealthy  = lipgloss.NewStyle().Foreground(colorHealthy).Bold(true)
	styleWarning  = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	styleCritical = lipgloss.NewStyle().Foreground(colorCritical).Bold(true)
	styleMuted    = lipgloss.NewStyle().Foreground(colorMuted)
	styleBold     = lipgloss.NewStyle().Bold(true)
	styleLabel    = lipgloss.NewStyle().Foreground(colorMuted)
)

// HealthStatus represents overall daemon health.
type HealthStatus int

const (
	HealthHealthy HealthStatus = iota
	HealthWarning
	HealthCritical
)

// FormatStatus renders the daemon status with improved UX.
// Uses compact format approved by user.
func FormatStatus(status *StatusData) string {
	var out strings.Builder

	// determine health status
	health := determineHealth(status)

	// header: ● Daemon: Healthy (2h 30m uptime)
	out.WriteString(formatHeader(health, status))
	out.WriteString("\n\n")

	// core info
	if status.WorkspacePath != "" {
		out.WriteString(formatKV("Workspace", shortenPath(status.WorkspacePath)))
		out.WriteString("\n")
	}
	out.WriteString(formatKV("PID", fmt.Sprintf("%d", status.Pid)))
	out.WriteString("\n")
	out.WriteString(formatKV("Version", status.Version))
	out.WriteString("\n")
	if status.AuthenticatedUser != nil && status.AuthenticatedUser.Email != "" {
		out.WriteString(formatKV("User", status.AuthenticatedUser.Email))
		out.WriteString("\n")
	}
	out.WriteString(formatKV("Ledger", shortenPath(status.LedgerPath)))
	out.WriteString("\n")
	if status.SyncIntervalRead > 0 {
		out.WriteString(formatKV("Sync every", formatDurationCompact(status.SyncIntervalRead)))
		out.WriteString("\n")
	}
	out.WriteString("\n")

	// sync line: Sync: ✓ 47 total  ✓ 12/hour  ✓ 1.2s avg
	out.WriteString(formatSyncLine(status))
	out.WriteString("\n")

	// last sync
	if !status.LastSync.IsZero() {
		out.WriteString(formatKV("Last sync", formatRelativeTime(time.Since(status.LastSync))))
	} else {
		out.WriteString(formatKV("Last sync", styleMuted.Render("never")))
	}
	out.WriteString("\n\n")

	// errors
	errStr := fmt.Sprintf("%d in last hour", status.RecentErrorCount)
	if status.RecentErrorCount > 0 {
		errStr = styleCritical.Render(errStr)
	}
	errStr += fmt.Sprintf(" (%d unviewed)", status.UnviewedErrorCount)
	out.WriteString(formatKV("Errors", errStr))
	out.WriteString("\n")

	// show last error if present
	if status.LastError != "" {
		out.WriteString("\n")
		lastErrLine := "  Last: " + status.LastError
		if status.LastErrorTime != "" {
			if t, err := time.Parse(time.RFC3339, status.LastErrorTime); err == nil {
				lastErrLine += " (" + formatRelativeTime(time.Since(t)) + ")"
			}
		}
		out.WriteString(styleMuted.Render(lastErrLine))
		out.WriteString("\n")

		// add hint based on error type
		hint := getErrorHint(status.LastError)
		if hint != "" {
			out.WriteString(styleMuted.Render("  Hint: " + hint))
			out.WriteString("\n")
		}
		out.WriteString(styleMuted.Render("  Logs: ox daemon logs"))
		out.WriteString("\n")
	}

	// show ledgers if any (from Workspaces map)
	if ledgers, ok := status.Workspaces["ledger"]; ok && len(ledgers) > 0 {
		out.WriteString("\n")
		out.WriteString(formatLedgers(ledgers))
	}

	// show team contexts if any
	if len(status.TeamContexts) > 0 {
		out.WriteString("\n")
		out.WriteString(formatTeamContexts(status.TeamContexts))
	}

	// show activity if any
	if activity := formatActivity(status.Activity); activity != "" {
		out.WriteString("\n")
		out.WriteString(activity)
	}

	// show issues if any
	if issues := formatIssues(status.NeedsHelp, status.Issues); issues != "" {
		out.WriteString("\n")
		out.WriteString(issues)
	}

	// show inactivity info if enabled
	if status.InactivityTimeout > 0 {
		out.WriteString("\n")
		remaining := status.InactivityTimeout - status.TimeSinceActivity
		if remaining < 0 {
			remaining = 0
		}
		out.WriteString(formatKV("Auto-exit", fmt.Sprintf("in %s (after %s inactivity)",
			formatDurationCompact(remaining),
			formatDurationCompact(status.InactivityTimeout))))
		out.WriteString("\n")
	}

	return out.String()
}

// FormatStatusVerbose includes additional diagnostic details.
func FormatStatusVerbose(status *StatusData, history []SyncEvent) string {
	out := FormatStatus(status)

	// add sync history table
	if len(history) > 0 {
		out += "\n" + formatSyncHistory(history)
	}

	return out
}

func formatHeader(health HealthStatus, status *StatusData) string {
	var indicator, statusText string
	var style lipgloss.Style

	switch health {
	case HealthHealthy:
		indicator = "●"
		statusText = "Healthy"
		style = styleHealthy
	case HealthWarning:
		indicator = "◐"
		statusText = "Warning"
		style = styleWarning
	case HealthCritical:
		indicator = "○"
		statusText = "Critical"
		style = styleCritical
	}

	uptime := formatDurationCompact(status.Uptime)
	return style.Render(fmt.Sprintf("%s Daemon: %s", indicator, statusText)) +
		styleMuted.Render(fmt.Sprintf(" (%s uptime)", uptime))
}

func formatSyncLine(status *StatusData) string {
	var parts []string

	// total syncs
	if status.TotalSyncs > 0 {
		parts = append(parts, styleHealthy.Render("✓")+" "+fmt.Sprintf("%d total", status.TotalSyncs))
	} else {
		parts = append(parts, styleMuted.Render("0 total"))
	}

	// syncs per hour
	if status.SyncsLastHour > 0 {
		parts = append(parts, styleHealthy.Render("✓")+" "+fmt.Sprintf("%d/hour", status.SyncsLastHour))
	} else {
		parts = append(parts, styleMuted.Render("0/hour"))
	}

	// avg duration
	if status.AvgSyncTime > 0 {
		parts = append(parts, styleHealthy.Render("✓")+" "+formatDurationCompact(status.AvgSyncTime)+" avg")
	}

	return formatKV("Sync", strings.Join(parts, "  "))
}

// formatLedgers renders ledger sync status.
func formatLedgers(ledgers []WorkspaceSyncStatus) string {
	var out strings.Builder

	out.WriteString(styleBold.Render("Ledgers"))
	out.WriteString(styleMuted.Render(fmt.Sprintf(" (%d)", len(ledgers))))
	out.WriteString("\n")

	for _, l := range ledgers {
		var status string
		if !l.Exists {
			status = styleMuted.Render("○ not cloned")
		} else if l.LastErr != "" {
			status = styleCritical.Render("○ " + l.LastErr)
		} else if l.Syncing {
			status = styleWarning.Render("◐ syncing...")
		} else if l.LastSync.IsZero() {
			status = styleWarning.Render("◐ not synced")
		} else {
			status = styleHealthy.Render("● synced " + formatRelativeTime(time.Since(l.LastSync)))
		}

		// use path basename as display name
		name := filepath.Base(l.Path)
		if name == "" || name == "." {
			name = l.ID
		}
		out.WriteString(fmt.Sprintf("  %-20s %s\n", name+":", status))
	}

	return out.String()
}

// formatTeamContexts renders team context sync status.
func formatTeamContexts(contexts []TeamContextSyncStatus) string {
	var out strings.Builder

	out.WriteString(styleBold.Render("Team Contexts"))
	out.WriteString(styleMuted.Render(fmt.Sprintf(" (%d)", len(contexts))))
	out.WriteString("\n")

	for _, tc := range contexts {
		var status string
		if !tc.Exists {
			status = styleMuted.Render("○ not cloned")
		} else if tc.LastErr != "" {
			status = styleCritical.Render("○ " + tc.LastErr)
		} else if tc.LastSync.IsZero() {
			status = styleWarning.Render("◐ not synced")
		} else {
			status = styleHealthy.Render("● synced " + formatRelativeTime(time.Since(tc.LastSync)))
		}

		out.WriteString(fmt.Sprintf("  %-20s %s\n", tc.TeamName+":", status))
	}

	return out.String()
}

// formatActivity renders heartbeat activity tracking.
func formatActivity(activity *ActivitySummary) string {
	if activity == nil {
		return ""
	}

	type category struct {
		label   string
		entries []ActivityEntry
	}

	categories := []category{
		{"Repos", activity.Repos},
		{"Teams", activity.Teams},
		{"Agents", activity.Agents},
	}

	// check if all categories are empty
	hasAny := false
	for _, c := range categories {
		if len(c.entries) > 0 {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return ""
	}

	var out strings.Builder
	out.WriteString(styleBold.Render("Activity"))
	out.WriteString("\n")

	for _, c := range categories {
		if len(c.entries) == 0 {
			continue
		}
		// find most recent across all entries in this category
		var mostRecent time.Time
		for _, e := range c.entries {
			if e.Last.After(mostRecent) {
				mostRecent = e.Last
			}
		}
		val := fmt.Sprintf("%d", len(c.entries))
		if !mostRecent.IsZero() {
			val += " (last " + formatRelativeTime(time.Since(mostRecent)) + ")"
		}
		out.WriteString(formatKV(c.label, val))
		out.WriteString("\n")
	}

	return out.String()
}

// formatIssues renders daemon issues that need attention.
func formatIssues(needsHelp bool, issues []DaemonIssue) string {
	if !needsHelp && len(issues) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString(styleBold.Render("Issues"))
	out.WriteString(styleMuted.Render(fmt.Sprintf(" (%d)", len(issues))))
	out.WriteString("\n")

	for _, issue := range issues {
		var icon string
		var style lipgloss.Style
		switch issue.Severity {
		case "warning":
			icon = "⚠"
			style = styleWarning
		default: // error, critical
			icon = "○"
			style = styleCritical
		}

		line := fmt.Sprintf("  %s %-10s ", icon, issue.Severity)
		desc := issue.Summary
		if issue.Repo != "" {
			desc = issue.Repo + ": " + desc
		}
		if !issue.Since.IsZero() {
			desc += " (since " + formatRelativeTime(time.Since(issue.Since)) + ")"
		}
		line += desc

		out.WriteString(style.Render(line))
		out.WriteString("\n")
	}

	return out.String()
}

// formatSyncHistory renders recent sync history as a table (for verbose mode).
func formatSyncHistory(history []SyncEvent) string {
	var out strings.Builder

	out.WriteString(styleBold.Render("Recent Syncs"))
	out.WriteString(styleMuted.Render(" (last 10)"))
	out.WriteString("\n\n")

	// header
	fmt.Fprintf(&out, "%s\n", styleLabel.Render(fmt.Sprintf("  %-12s %-8s %-10s %s",
		"Time", "Type", "Duration", "Files")))

	// show last 10
	start := len(history) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(history); i++ {
		event := history[i]
		timeStr := event.Time.Format("15:04:05")
		typeStr := event.Type
		durationStr := formatDurationCompact(event.Duration)
		filesStr := fmt.Sprintf("%d", event.FilesChanged)

		fmt.Fprintf(&out, "  %-12s %-8s %-10s %s\n",
			timeStr, typeStr, durationStr, filesStr)
	}

	return out.String()
}

// determineHealth calculates overall health status.
func determineHealth(status *StatusData) HealthStatus {
	// critical: many recent errors or sync very stale
	if status.RecentErrorCount >= 5 {
		return HealthCritical
	}

	if !status.LastSync.IsZero() {
		sinceLast := time.Since(status.LastSync)

		// if last sync is > 2x expected interval, something's wrong
		if sinceLast > status.SyncIntervalRead*2 {
			return HealthCritical
		}
	}

	// warning: some errors
	if status.RecentErrorCount > 0 {
		return HealthWarning
	}

	// healthy
	return HealthHealthy
}

// formatKV formats a key-value pair with consistent spacing.
func formatKV(key, value string) string {
	return fmt.Sprintf("  %s %s",
		styleLabel.Render(fmt.Sprintf("%-12s", key+":")),
		value,
	)
}

// formatDurationCompact formats a duration in compact form (e.g., "2h 30m").
func formatDurationCompact(d time.Duration) string {
	d = d.Round(time.Second)

	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

// formatRelativeTime formats a duration as relative time (e.g., "5m ago").
func formatRelativeTime(d time.Duration) string {
	d = d.Round(time.Second)

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

// shortenPath replaces home directory with ~ for display.
func shortenPath(path string) string {
	if path == "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// getErrorHint returns actionable hint for common error types.
func getErrorHint(err string) string {
	switch {
	case strings.Contains(err, "authentication"):
		return "Run: git config credential.helper store"
	case strings.Contains(err, "permission denied"):
		return "Check SSH key: ssh -T git@github.com"
	case strings.Contains(err, "push failed"):
		return "Pull may be needed. Run: ox ledger sync"
	default:
		return ""
	}
}

// FormatDaemonList formats a list of daemons for display.
func FormatDaemonList(daemons []DaemonInfo) string {
	if len(daemons) == 0 {
		return styleMuted.Render("No ox daemons running") + "\n"
	}

	var out strings.Builder

	out.WriteString(styleBold.Render(fmt.Sprintf("Running Daemons (%d)", len(daemons))))
	out.WriteString("\n")

	// header
	out.WriteString(styleLabel.Render(fmt.Sprintf("%-42s %-8s %-10s %s",
		"Workspace", "PID", "Version", "Uptime")))
	out.WriteString("\n")

	for _, d := range daemons {
		uptime := formatDurationCompact(time.Since(d.StartedAt))
		workspace := shortenPath(d.WorkspacePath)
		if len(workspace) > 40 {
			workspace = "..." + filepath.Base(d.WorkspacePath)
		}
		fmt.Fprintf(&out, "%-42s %-8d %-10s %s\n",
			workspace, d.PID, d.Version, uptime)
	}

	return out.String()
}
