# Agent Integration Tests

These tests verify that coding agents properly understand and use ox CLI features
after running `ox agent prime`. The framework is designed to support multiple
agents, though currently only Claude Code is implemented.

## Supported Agents

| Agent | Status | CLI | Notes |
|-------|--------|-----|-------|
| Claude Code | Implemented | `claude` | Primary supported agent |
| OpenCode | Planned | `opencode` | Open source coding agent |
| Codex | Planned | `codex` | OpenAI Codex CLI |

Note: We focus on legitimate CLI-based coding agents that can run in terminal environments.

## Prerequisites

1. **Agent CLI**: The agent's CLI must be installed and authenticated
2. **Go**: For running the tests
3. **ox CLI**: Built from this repository (tests build it automatically)

## Running Tests

```bash
# Run all agent integration tests (slow, requires agent CLI)
go test -v -tags=integration ./tests/integration/agents/...

# Run Claude-specific tests only
go test -v -tags=integration -timeout 5m ./tests/integration/agents/claude/...

# Skip if agent CLI not available (graceful degradation)
go test -v -tags=integration -short ./tests/integration/agents/...

# Debug mode - see full agent output
AGENT_TEST_DEBUG=1 go test -v -tags=integration ./tests/integration/agents/...
```

## Test Architecture

### Environment Isolation

Tests use isolated environments to avoid affecting user configuration:
- Custom `XDG_*` environment variables point to test fixtures
- `OX_OFFLINE=1` prevents cloud API calls
- Pre-cached guidance responses in fixtures

### Test Fixtures

```
fixtures/
  mock-project/           # Simulated project directory
    .sageox/
      local.json          # Config pointing to mock team context
      cache/guidance/     # Pre-cached guidance
    .git/                 # Git repo (required for ox detection)
    AGENTS.md             # Project-level agents file
  mock-ledger/            # Simulated ledger git repo
  mock-team-context/      # Simulated team context
    coworkers/ai/claude/
      CLAUDE.md           # Team Claude config
      AGENTS.md           # Team agents index
      agents/             # Team subagent definitions
      commands/           # Team slash commands
```

### Test Harness

The harness (`harness.go`):
1. Builds ox CLI to a temp directory
2. Sets up isolated environment
3. Runs Claude with structured prompts
4. Parses JSON responses from Claude
5. Verifies understanding of ox features

### What We Test

1. **Prime Output Understanding**
   - Claude knows about key ox commands
   - Claude understands attribution requirements
   - Claude sees team context and subagents

2. **Guidance System Usage**
   - Claude fetches guidance for relevant paths
   - Claude uses progressive disclosure (fetches deeper guidance when needed)

## Adding New Tests

1. Add test fixtures if needed
2. Create test function with `//go:build integration` tag
3. Use `env.RunAgentPrompt()` from harness
4. Parse response and assert expectations

## Debugging

Set `AGENT_TEST_DEBUG=1` to see the agent's full output:

```bash
AGENT_TEST_DEBUG=1 go test -v -tags=integration ./tests/integration/agents/...
```
