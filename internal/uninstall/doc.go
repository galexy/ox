// Package uninstall provides functionality for removing SageOx from repositories.
//
// The package handles the complete uninstall process including:
//   - Finding and removing the .sageox directory
//   - Cleaning up git hooks installed by SageOx
//   - Removing agent integration files (AGENTS.md, CLAUDE.md, etc.)
//   - Optionally removing user-level integrations
//
// The uninstall process uses git commands to properly handle tracked files
// and ensures user content is preserved when cleaning up shared files.
package uninstall
