package session

import (
	"regexp"
	"strings"

	mermaidcmd "github.com/AlexanderGrooff/mermaid-ascii/cmd"
)

// mermaidBlockPattern matches ```mermaid ... ``` code blocks
var mermaidBlockPattern = regexp.MustCompile("(?s)```mermaid\\s*\n(.*?)```")

// RenderMermaidToASCII renders a mermaid diagram string to ASCII art.
// Returns the ASCII representation or an error if rendering fails.
// Uses the mermaid-ascii library compiled into the binary.
func RenderMermaidToASCII(mermaidCode string) (string, error) {
	// pass nil config to use defaults
	return mermaidcmd.RenderDiagram(mermaidCode, nil)
}

// ProcessMermaidBlocks finds all ```mermaid blocks in content and renders them to ASCII.
// Returns the content with mermaid blocks replaced by their ASCII representations.
// If rendering fails for a block, it's left as-is with an error comment.
func ProcessMermaidBlocks(content string) string {
	return mermaidBlockPattern.ReplaceAllStringFunc(content, func(match string) string {
		// extract the mermaid code (without the ``` markers)
		submatch := mermaidBlockPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}

		mermaidCode := strings.TrimSpace(submatch[1])
		if mermaidCode == "" {
			return match
		}

		// render to ASCII
		ascii, err := RenderMermaidToASCII(mermaidCode)
		if err != nil {
			// keep original with error note
			return match + "\n<!-- mermaid render error: " + err.Error() + " -->\n"
		}

		// wrap in code block for proper formatting
		return "```\n" + strings.TrimSpace(ascii) + "\n```"
	})
}

// HasMermaidBlocks checks if content contains any mermaid code blocks.
func HasMermaidBlocks(content string) bool {
	return mermaidBlockPattern.MatchString(content)
}

// ExtractMermaidBlocks finds all mermaid code blocks in content.
// Returns the raw mermaid code (without the ``` markers).
func ExtractMermaidBlocks(content string) []string {
	matches := mermaidBlockPattern.FindAllStringSubmatch(content, -1)
	var diagrams []string
	for _, match := range matches {
		if len(match) >= 2 {
			code := strings.TrimSpace(match[1])
			if code != "" {
				diagrams = append(diagrams, code)
			}
		}
	}
	return diagrams
}

// ExtractDiagramsFromEntries scans all entries for mermaid diagrams.
// Returns deduplicated list of diagrams found.
func ExtractDiagramsFromEntries(entries []map[string]any) []string {
	seen := make(map[string]bool)
	var diagrams []string

	for _, entry := range entries {
		// try content field
		if content, ok := entry["content"].(string); ok {
			for _, d := range ExtractMermaidBlocks(content) {
				if !seen[d] {
					seen[d] = true
					diagrams = append(diagrams, d)
				}
			}
		}

		// try nested data.content
		if data, ok := entry["data"].(map[string]any); ok {
			if content, ok := data["content"].(string); ok {
				for _, d := range ExtractMermaidBlocks(content) {
					if !seen[d] {
						seen[d] = true
						diagrams = append(diagrams, d)
					}
				}
			}
		}
	}

	return diagrams
}
