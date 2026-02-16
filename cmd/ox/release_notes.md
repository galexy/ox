# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-14

Initial public release of the SageOx CLI (`ox`).

### Highlights

- **Agent-aware CLI**: Detects 14+ coding agents (Claude Code, Cursor, Windsurf, Copilot, Aider, Cody, and more) with automatic context injection
- **Progressive guidance**: On-demand infrastructure guidance via `ox agent prime` and domain-specific `ox agent guidance <path>` commands
- **Session recording**: Capture, view, and export human-AI coding sessions with HTML and Markdown output
- **Background daemon**: Automatic git sync for ledgers and team contexts with self-healing clone recovery
- **Doctor diagnostics**: Comprehensive health checks with granular fix levels (auto, suggested, confirm) and agent-vs-human routing
- **Team context**: Shared team conventions, coworker discovery, and collaborative agent configurations
- **Multi-endpoint support**: Per-endpoint credentials, endpoint-aware init and login, and normalized endpoint resolution
- **Secure by default**: Ed25519 signature verification, OS keychain credential storage, and cache obfuscation
- **Cross-platform**: macOS, Linux, and Windows support with XDG Base Directory compliance

[0.1.0]: https://github.com/sageox/ox/releases/tag/v0.1.0
