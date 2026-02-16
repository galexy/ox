package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Note: Entry and EntryType* constants are defined in session.go

// ExtractedEventType categorizes events extracted from sessions.
type ExtractedEventType string

const (
	ExtractedEventUserAsked      ExtractedEventType = "user_asked"
	ExtractedEventAgentResponded ExtractedEventType = "agent_responded"
	ExtractedEventCommandRun     ExtractedEventType = "command_run"
	ExtractedEventFileEdited     ExtractedEventType = "file_edited"
	ExtractedEventErrorOccurred  ExtractedEventType = "error_occurred"
	ExtractedEventResolved       ExtractedEventType = "resolved"
	ExtractedEventOxCommand      ExtractedEventType = "ox_command" // ox-specific commands
)

// ExtractedEvent represents a structured event extracted from a session.
type ExtractedEvent struct {
	Timestamp   time.Time          `json:"ts"`
	Type        ExtractedEventType `json:"type"`
	Summary     string             `json:"summary"`           // brief description
	Details     string             `json:"details,omitempty"` // additional context
	Success     *bool              `json:"success,omitempty"` // for commands
	ErrorMsg    string             `json:"error,omitempty"`
	RelatedFile string             `json:"file,omitempty"`
}

// EventLogMetadata contains context about the event log source.
type EventLogMetadata struct {
	SessionID   string    `json:"session_id,omitempty"`
	Agent       string    `json:"agent,omitempty"`
	ExtractedAt time.Time `json:"extracted_at"`
	EntryCount  int       `json:"entry_count"`
	EventCount  int       `json:"event_count"`
}

// EventLog is a collection of extracted events from a session.
type EventLog struct {
	Metadata *EventLogMetadata `json:"_meta"`
	Events   []ExtractedEvent  `json:"events"`
}

// pattern matchers for event detection
var (
	// error patterns
	eventLogErrorPatterns = []string{
		"error:", "Error:", "ERROR:",
		"failed", "Failed", "FAILED",
		"fatal:", "Fatal:", "FATAL:",
		"panic:", "Panic:",
		"exception", "Exception",
	}

	// success patterns
	eventLogSuccessPatterns = []string{
		"success", "Success", "SUCCESS",
		"completed", "Completed",
		"passed", "Passed",
		"done", "Done",
		"finished", "Finished",
	}

	// ox command pattern: matches "ox <subcommand>" at word boundaries
	eventLogOxCommandRegex = regexp.MustCompile(`\box\s+(\S+)`)

	// file path pattern: matches common file extensions
	// longer extensions first to avoid partial matches (json before js, yaml before y, etc.)
	eventLogFilePathRegex = regexp.MustCompile(`[/\w.-]+\.(jsonl|json|yaml|yml|bash|html|toml|go|py|ts|js|md|txt|sh|sql|css|xml)`)

	// common tool names
	eventLogToolNames = map[string]bool{
		"bash":    true,
		"read":    true,
		"write":   true,
		"edit":    true,
		"grep":    true,
		"glob":    true,
		"execute": true,
	}

	// command tools that produce command_run events
	eventLogCommandTools = map[string]bool{
		"bash":    true,
		"execute": true,
	}

	// edit tools that produce file_edited events
	eventLogEditTools = map[string]bool{
		"write": true,
		"edit":  true,
	}

	// read tools that are usually skipped unless they fail
	eventLogReadTools = map[string]bool{
		"read": true,
		"glob": true,
		"grep": true,
	}
)

// ExtractEventsFromEntries converts session entries to structured events.
// Uses simple heuristics to identify key events from conversation flow.
func ExtractEventsFromEntries(entries []Entry) []ExtractedEvent {
	events := make([]ExtractedEvent, 0, len(entries)/2) // rough estimate

	for i, entry := range entries {
		var event *ExtractedEvent

		switch entry.Type {
		case EntryTypeUser:
			event = extractEventFromUserEntry(entry)

		case EntryTypeAssistant:
			event = extractEventFromAssistantEntry(entry)

		case EntryTypeTool:
			event = extractEventFromToolEntry(entry, i, entries)
		}

		if event != nil {
			events = append(events, *event)
		}
	}

	return events
}

// extractEventFromUserEntry creates an event from a user message
func extractEventFromUserEntry(entry Entry) *ExtractedEvent {
	content := strings.TrimSpace(entry.Content)
	if content == "" {
		return nil
	}

	summary := eventLogSummarizeContent(content, 100)

	return &ExtractedEvent{
		Timestamp: entry.Timestamp,
		Type:      ExtractedEventUserAsked,
		Summary:   summary,
	}
}

// extractEventFromAssistantEntry creates an event from an assistant message
func extractEventFromAssistantEntry(entry Entry) *ExtractedEvent {
	content := strings.TrimSpace(entry.Content)
	if content == "" {
		return nil
	}

	event := &ExtractedEvent{
		Timestamp: entry.Timestamp,
		Type:      ExtractedEventAgentResponded,
		Summary:   eventLogSummarizeContent(content, 100),
	}

	// check for ox commands mentioned in response
	if matches := eventLogOxCommandRegex.FindAllStringSubmatch(content, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				// create separate ox_command event
				oxEvent := &ExtractedEvent{
					Timestamp: entry.Timestamp,
					Type:      ExtractedEventOxCommand,
					Summary:   "ox " + match[1],
				}
				// only return if this appears to be command execution context
				if strings.Contains(content, "run") || strings.Contains(content, "execute") {
					return oxEvent
				}
			}
		}
	}

	// check for errors in response
	hasError, errMsg := eventLogDetectError(content)
	if hasError {
		event.Type = ExtractedEventErrorOccurred
		event.ErrorMsg = errMsg
	} else if eventLogDetectSuccess(content) {
		// only mark as resolved if no error detected
		event.Type = ExtractedEventResolved
	}

	// extract file references
	if file := eventLogExtractFilePath(content); file != "" {
		event.RelatedFile = file
	}

	return event
}

// extractEventFromToolEntry creates an event from a tool execution
func extractEventFromToolEntry(entry Entry, _ int, _ []Entry) *ExtractedEvent {
	toolName := strings.ToLower(entry.ToolName)
	content := entry.Content
	if entry.ToolOutput != "" {
		content = entry.ToolOutput
	}
	input := entry.ToolInput

	event := &ExtractedEvent{
		Timestamp: entry.Timestamp,
	}

	// determine event type based on tool category
	switch {
	case eventLogCommandTools[toolName]:
		event.Type = ExtractedEventCommandRun
		event.Summary = eventLogSummarizeCommand(input)

	case eventLogEditTools[toolName]:
		event.Type = ExtractedEventFileEdited
		event.Summary = "edited file"
		if file := eventLogExtractFilePath(input); file != "" {
			event.RelatedFile = file
			event.Summary = "edited " + file
		}

	case eventLogReadTools[toolName]:
		// read operations are less significant; skip unless they fail
		if hasError, _ := eventLogDetectError(content); !hasError {
			return nil
		}
		event.Type = ExtractedEventErrorOccurred
		event.Summary = toolName + " failed"

	default:
		if !eventLogToolNames[toolName] {
			return nil // skip unknown tools
		}
		event.Type = ExtractedEventCommandRun
		event.Summary = toolName + " executed"
	}

	// detect success/failure from tool output
	if hasError, errMsg := eventLogDetectError(content); hasError {
		event.Type = ExtractedEventErrorOccurred
		event.ErrorMsg = errMsg
		success := false
		event.Success = &success
	} else if eventLogDetectSuccess(content) {
		success := true
		event.Success = &success
	}

	// extract file from tool output
	if event.RelatedFile == "" {
		event.RelatedFile = eventLogExtractFilePath(content)
	}

	return event
}

// eventLogDetectError checks content for error indicators
// returns (hasError, errorMessage)
func eventLogDetectError(content string) (bool, string) {
	contentLower := strings.ToLower(content)

	for _, pattern := range eventLogErrorPatterns {
		patternLower := strings.ToLower(pattern)
		if strings.Contains(contentLower, patternLower) {
			// extract a snippet around the error
			idx := strings.Index(contentLower, patternLower)
			end := idx + 80
			if end > len(content) {
				end = len(content)
			}
			snippet := strings.TrimSpace(content[idx:end])
			// truncate at newline
			if newline := strings.Index(snippet, "\n"); newline > 0 {
				snippet = snippet[:newline]
			}
			return true, snippet
		}
	}

	// check for exit codes
	if strings.Contains(contentLower, "exit code") && !strings.Contains(contentLower, "exit code 0") {
		return true, "non-zero exit code"
	}

	return false, ""
}

// eventLogDetectSuccess checks content for success indicators
func eventLogDetectSuccess(content string) bool {
	contentLower := strings.ToLower(content)
	for _, pattern := range eventLogSuccessPatterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return true
		}
	}

	// check for exit code 0
	if strings.Contains(contentLower, "exit code 0") {
		return true
	}

	return false
}

// eventLogExtractFilePath finds the first file path in content
func eventLogExtractFilePath(content string) string {
	match := eventLogFilePathRegex.FindString(content)
	if match != "" {
		// clean up path - ensure it starts with / or is relative
		if strings.HasPrefix(match, "/") || !strings.Contains(match, "/") {
			return match
		}
		// find the path start
		slashIdx := strings.Index(match, "/")
		if slashIdx > 0 {
			idx := strings.LastIndex(match[:slashIdx], " ")
			if idx > 0 {
				return match[idx+1:]
			}
		}
	}
	return match
}

// eventLogSummarizeContent creates a brief summary of content
func eventLogSummarizeContent(content string, maxLen int) string {
	// normalize whitespace
	content = strings.Join(strings.Fields(content), " ")

	if len(content) <= maxLen {
		return content
	}

	// truncate at word boundary
	truncated := content[:maxLen]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// eventLogSummarizeCommand creates a brief summary of a command
func eventLogSummarizeCommand(input string) string {
	// normalize and get first line
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "command executed"
	}

	cmd := strings.TrimSpace(lines[0])
	if len(cmd) > 60 {
		// truncate long commands
		if spaceIdx := strings.Index(cmd[30:], " "); spaceIdx > 0 {
			cmd = cmd[:30+spaceIdx] + "..."
		} else {
			cmd = cmd[:60] + "..."
		}
	}

	return cmd
}

// WriteEventLog writes event log to a JSONL file.
// Returns any error from writing or closing the file.
// TODO(server-side): move to server-side for MVP+1; client should not write to ledger directly.
func WriteEventLog(path string, log *EventLog) (err error) {
	if path == "" {
		return fmt.Errorf("%w: event log path", ErrEmptyPath)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create event log file=%s: %w", path, err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close event log file=%s: %w", path, cerr)
		}
	}()

	// write metadata as first line
	metaBytes, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("marshal event log metadata file=%s: %w", path, err)
	}
	if _, err := file.Write(metaBytes); err != nil {
		return fmt.Errorf("write event log metadata file=%s: %w", path, err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("write event log newline file=%s: %w", path, err)
	}

	// write each event as a line
	for i, event := range log.Events {
		eventBytes, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event index=%d file=%s: %w", i, path, err)
		}
		if _, err := file.Write(eventBytes); err != nil {
			return fmt.Errorf("write event index=%d file=%s: %w", i, path, err)
		}
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("write event newline index=%d file=%s: %w", i, path, err)
		}
	}

	return nil
}

// ReadEventLog reads event log from a JSONL file
func ReadEventLog(path string) (*EventLog, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: event log path", ErrEmptyPath)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open event log file=%s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	log := &EventLog{
		Events: make([]ExtractedEvent, 0),
	}

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		if lineNum == 0 {
			// first line is metadata
			var meta EventLogMetadata
			if err := json.Unmarshal(line, &meta); err != nil {
				return nil, fmt.Errorf("parse event log metadata file=%s: %w", path, err)
			}
			log.Metadata = &meta
		} else {
			// subsequent lines are events
			var event ExtractedEvent
			if err := json.Unmarshal(line, &event); err != nil {
				return nil, fmt.Errorf("parse event line=%d file=%s: %w", lineNum+1, path, err)
			}
			log.Events = append(log.Events, event)
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read event log file=%s: %w", path, err)
	}

	return log, nil
}

// NewEventLog creates an EventLog from session entries with metadata
func NewEventLog(entries []Entry, sessionID, agent string) *EventLog {
	events := ExtractEventsFromEntries(entries)

	return &EventLog{
		Metadata: &EventLogMetadata{
			SessionID:   sessionID,
			Agent:       agent,
			ExtractedAt: time.Now(),
			EntryCount:  len(entries),
			EventCount:  len(events),
		},
		Events: events,
	}
}
