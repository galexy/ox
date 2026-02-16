package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RemovalItem represents a SageOx section found in an agent file
type RemovalItem struct {
	FilePath    string // absolute path to the file
	FileName    string // base filename (AGENTS.md or CLAUDE.md)
	StartLine   int    // line number where SageOx section starts (1-indexed)
	EndLine     int    // line number where SageOx section ends (1-indexed)
	IsSymlink   bool   // whether this is a symlink to another file
	IsEmptyFile bool   // whether file will be empty after removal
}

// FindAgentFileEntries locates SageOx sections in AGENTS.md and CLAUDE.md
// Returns a list of removal items that can be used for cleanup
func FindAgentFileEntries(repoRoot string) ([]RemovalItem, error) {
	var items []RemovalItem

	agentsPath := filepath.Join(repoRoot, "AGENTS.md")
	claudePath := filepath.Join(repoRoot, "CLAUDE.md")

	// check AGENTS.md
	if item, err := findSageOxSection(agentsPath, "AGENTS.md"); err == nil && item != nil {
		items = append(items, *item)
	} else if err != nil {
		slog.Debug("error checking AGENTS.md", "error", err)
	}

	// check CLAUDE.md (skip if it's a symlink to AGENTS.md)
	claudeInfo, err := os.Lstat(claudePath)
	if err == nil {
		isSymlink := claudeInfo.Mode()&os.ModeSymlink != 0
		if isSymlink {
			// it's a symlink, check if it points to AGENTS.md
			target, err := os.Readlink(claudePath)
			if err == nil && target == "AGENTS.md" {
				slog.Debug("CLAUDE.md is symlink to AGENTS.md, will be removed separately")
			}
		} else {
			// regular file, check for SageOx section
			if item, err := findSageOxSection(claudePath, "CLAUDE.md"); err == nil && item != nil {
				items = append(items, *item)
			} else if err != nil {
				slog.Debug("error checking CLAUDE.md", "error", err)
			}
		}
	}

	return items, nil
}

// findSageOxSection finds the SageOx section in a file
// Returns nil if file doesn't exist or doesn't contain SageOx section
func findSageOxSection(filePath, fileName string) (*RemovalItem, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // file doesn't exist, not an error
		}
		return nil, fmt.Errorf("failed to read %s: %w", fileName, err)
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// look for the SageOx section marker: "- **SageOx**:"
	// The section includes the line with the marker and the indented continuation line
	var startLine, endLine int
	for i, line := range lines {
		if strings.Contains(line, "- **SageOx**:") {
			startLine = i + 1 // convert to 1-indexed
			// find the end of the section (next line that's not indented with "  -")
			endLine = startLine
			for j := i + 1; j < len(lines); j++ {
				// continuation lines start with "  - " (two spaces, dash, space)
				if strings.HasPrefix(lines[j], "  - ") {
					endLine = j + 1 // convert to 1-indexed
				} else {
					break
				}
			}
			break
		}
	}

	if startLine == 0 {
		// no SageOx section found
		return nil, nil
	}

	// check if file would be empty after removal
	// (only contains the SageOx section and optional whitespace)
	nonSageOxLines := 0
	for i, line := range lines {
		lineNum := i + 1
		if lineNum < startLine || lineNum > endLine {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && trimmed != "# AI Agent Instructions" {
				nonSageOxLines++
			}
		}
	}

	item := &RemovalItem{
		FilePath:    filePath,
		FileName:    fileName,
		StartLine:   startLine,
		EndLine:     endLine,
		IsSymlink:   false,
		IsEmptyFile: nonSageOxLines == 0,
	}

	return item, nil
}

// CleanupAgentFiles removes SageOx sections from AGENTS.md and CLAUDE.md
// If dryRun is true, only reports what would be done without making changes
func CleanupAgentFiles(repoRoot string, dryRun bool) error {
	items, err := FindAgentFileEntries(repoRoot)
	if err != nil {
		return fmt.Errorf("finding agent file entries: %w", err)
	}

	if len(items) == 0 {
		slog.Debug("no SageOx sections found in agent files")
		return nil
	}

	for _, item := range items {
		if dryRun {
			if item.IsEmptyFile {
				slog.Info("would delete file", "file", item.FileName)
			} else {
				slog.Info("would remove SageOx section", "file", item.FileName, "lines", fmt.Sprintf("%d-%d", item.StartLine, item.EndLine))
			}
			continue
		}

		if item.IsEmptyFile {
			// file will be empty after removal, delete it entirely
			if err := removeFileWithGit(item.FilePath); err != nil {
				slog.Warn("failed to remove file", "file", item.FileName, "error", err)
			} else {
				slog.Info("removed file", "file", item.FileName)
			}
		} else {
			// remove just the SageOx section
			if err := removeSageOxSection(&item); err != nil {
				slog.Warn("failed to remove SageOx section", "file", item.FileName, "error", err)
			} else {
				slog.Info("removed SageOx section", "file", item.FileName, "lines", fmt.Sprintf("%d-%d", item.StartLine, item.EndLine))
			}
		}
	}

	// remove CLAUDE.md symlink if it exists and points to AGENTS.md
	claudePath := filepath.Join(repoRoot, "CLAUDE.md")
	claudeInfo, err := os.Lstat(claudePath)
	if err == nil && claudeInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(claudePath)
		if err == nil && target == "AGENTS.md" {
			if dryRun {
				slog.Info("would remove symlink", "file", "CLAUDE.md")
			} else {
				if err := removeFileWithGit(claudePath); err != nil {
					slog.Warn("failed to remove symlink", "file", "CLAUDE.md", "error", err)
				} else {
					slog.Info("removed symlink", "file", "CLAUDE.md")
				}
			}
		}
	}

	return nil
}

// removeSageOxSection removes the SageOx section from a file
func removeSageOxSection(item *RemovalItem) error {
	content, err := os.ReadFile(item.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// remove lines from startLine to endLine (both inclusive, 1-indexed)
	var newLines []string
	for i, line := range lines {
		lineNum := i + 1
		if lineNum < item.StartLine || lineNum > item.EndLine {
			newLines = append(newLines, line)
		}
	}

	// clean up extra blank lines (remove duplicate consecutive blank lines)
	var cleaned []string
	prevBlank := false
	for _, line := range newLines {
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && prevBlank {
			continue // skip consecutive blank lines
		}
		cleaned = append(cleaned, line)
		prevBlank = isBlank
	}

	// trim trailing blank lines
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	// write back
	newContent := strings.Join(cleaned, "\n")
	if len(cleaned) > 0 {
		newContent += "\n" // ensure file ends with newline
	}

	if err := os.WriteFile(item.FilePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// removeFileWithGit removes a file using git rm if it's tracked, otherwise os.Remove
func removeFileWithGit(filePath string) error {
	// check if file is tracked in git
	cmd := exec.Command("git", "ls-files", "--error-unmatch", filePath)
	if err := cmd.Run(); err == nil {
		// file is tracked, use git rm
		cmd := exec.Command("git", "rm", filePath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git rm failed: %w", err)
		}
		return nil
	}

	// file is not tracked, use os.Remove
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}
