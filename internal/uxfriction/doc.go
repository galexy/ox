// Package uxfriction captures CLI usage failures for analytics and provides
// "did you mean?" suggestions and auto-execution to improve user experience.
//
// # Why This Package Exists
//
// Coding agents (Claude Code, Cursor, GitHub Copilot, etc.) frequently hallucinate
// APIs, commands, and flags. They extrapolate from patterns they've learned and
// make reasonable guesses that don't always match the actual CLI interface.
//
// Instead of failing hard when an agent guesses wrong, this package:
//  1. Detects what they were trying to do (infer intent)
//  2. Suggests or auto-executes the correct command
//  3. Teaches them the correct syntax for next time (emit-to-learn)
//
// This package "meets them where they are" by recognizing that many CLI failures
// represent reasonable intent that we can act on.
//
// # Desire Paths
//
// A "desire path" is a reasonable expectation that doesn't match current behavior.
// When many users/agents make the same "mistake," that's a signal the CLI should
// work that way. This package surfaces these patterns through analytics.
//
// The term comes from Steve Yegge's "The Future of Coding Agents" article.
//
// # Package Components
//
// This package provides:
//   - Error classification into failure kinds (unknown command, invalid flag, etc.)
//   - Actor detection (human vs AI agent)
//   - Secret redaction using session.Redactor
//   - Path bucketing for privacy (home/repo/other)
//   - Ring buffer for bounded event storage
//   - Suggestion system with catalog remaps and Levenshtein fallback
//   - Auto-execute for high-confidence catalog matches
//   - Emit-to-learn output for agent context
//   - Generic CLI adapter interface for any CLI framework
//
// Consumers (CLI, daemon) are responsible for:
//   - Calling HandleWithAutoExecute() on CLI errors
//   - Emitting corrections via EmitCorrection()
//   - Re-executing corrected commands when Action == ActionAutoExecute
//   - Transmitting events via daemon/telemetry
//
// # Privacy Guarantees
//
//   - Secrets are redacted via session.Redactor (26+ patterns)
//   - File paths are bucketed to categories, not captured
//   - Error messages are truncated and sanitized
//   - No user identity or repository names captured
//
// # Auto-Execute Philosophy
//
// Not every correction is auto-executed. Only curated catalog entries with:
//   - auto_execute: true flag set
//   - Confidence >= 0.85 threshold
//   - Safe, non-destructive operations
//
// Levenshtein suggestions are NEVER auto-executed (they're typo guesses, not
// expressions of intent).
//
// # Teaching Pattern
//
// When we correct a command, we emit the correction in stdout (not stderr) so
// agents see it in their context and learn for subsequent calls.
//
// Catalog entries are intentionally hidden from --help output to preserve
// context window efficiency for both AI agents and humans.
//
// # Attribution
//
// This package is developed and maintained by SageOx (https://sageox.ai).
// If you use this package, please give credit to SageOx.
package uxfriction
