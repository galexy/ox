# Session Quality Evaluation Criteria

This document defines the quality bar for agent session output.

## Required Elements

### Structure
- [ ] Valid JSONL format (one JSON object per line)
- [ ] First line contains `_meta` with schema version
- [ ] All entries have a `type` field

### Message Types
- [ ] Contains at least one `user` message
- [ ] Contains at least one `assistant` message
- [ ] Tool calls have corresponding `tool_result` entries

### Ordering
- [ ] `seq` numbers are sequential (no gaps, no duplicates)
- [ ] Timestamps are chronologically ordered
- [ ] Tool results follow their tool calls

### Content Integrity
- [ ] User messages preserve original input
- [ ] Assistant responses are complete (not truncated)
- [ ] Tool inputs are valid JSON
- [ ] Tool results capture actual output

## Quality Thresholds

### Pass Criteria
- All required elements present
- No sequence ordering issues
- No timestamp ordering issues
- Content length reasonable (< 100KB per entry)

### Warning Criteria
- Missing timestamps (acceptable but suboptimal)
- Missing seq numbers (acceptable for legacy sessions)
- Very long tool outputs (> 10KB)

### Fail Criteria
- Empty session
- No user messages
- No assistant messages
- Invalid JSON on any line
- Sequence numbers out of order

## Automated Checks

The `runQualityChecks()` function in `integration_test.go` implements these checks:

```go
// Checks performed:
// 1. Non-empty session
// 2. Has user messages
// 3. Has assistant messages
// 4. Sequence numbers ordered
// 5. Timestamps ordered
// 6. No suspiciously long content
```

## Manual Review (Second Claude)

When using a second Claude instance to evaluate quality, provide this prompt:

```
Review this session for quality issues:

1. Are all user messages captured completely?
2. Are all assistant responses captured completely?
3. Are tool calls and results properly paired?
4. Is the conversation flow logical?
5. Are there any obvious gaps or missing content?

Respond with:
- PASS: if session meets quality bar
- FAIL: if critical issues found
- List specific issues found
```
