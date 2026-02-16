package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// Generate creates markdown documentation for the given cobra command tree
func Generate(root *cobra.Command, outDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate markdown with custom frontmatter and link handling
	err := doc.GenMarkdownTreeCustom(root, outDir, filePrepender, linkHandler)
	if err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	return nil
}

// filePrepender adds YAML frontmatter to each generated markdown file
func filePrepender(filename string) string {
	// Extract base filename without extension
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))

	// Convert filename to title format
	// "ox_agent_prime.md" -> "ox agent prime"
	title := strings.ReplaceAll(name, "_", " ")

	// Generate frontmatter
	frontmatter := fmt.Sprintf(`---
title: "%s"
description: "CLI reference for %s"
---

`, title, title)

	return frontmatter
}

// linkHandler generates relative links between documentation files.
// Cobra passes names like "ox_agent_prime.md" (already has .md extension).
func linkHandler(name string) string {
	base := strings.TrimSuffix(name, ".md")
	base = strings.ReplaceAll(base, " ", "_")
	return base + ".md"
}
