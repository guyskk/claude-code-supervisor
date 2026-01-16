# Claude Agent SDK Auto-Compact Verification Tool

## Purpose

This tool verifies and tests the Claude Agent SDK's auto-compact functionality. It sends multiple queries to accumulate context tokens and triggers automatic compaction when the token threshold is reached.

## Background

The Claude Agent SDK automatically compacts the conversation context when it grows too large. This tool:

1. Sends multiple queries with controlled token sizes
2. Accumulates context tokens to reach the target threshold (200k tokens by default)
3. Detects when auto-compact is triggered via the pre-compact hook
4. Records all compact events and generates a detailed report

## Usage

### Prerequisites

- Have `ccc` configured with at least one provider
- The provider must have valid API credentials

### Running the Test

```bash
# From the project root directory
go run ./cmd/test-sdk-auto-compact/

# Or build and run
go build -o test-sdk-auto-compact ./cmd/test-sdk-auto-compact/
./test-sdk-auto-compact
```

### Configuration

The test uses the following default configuration (can be modified in `main.go`):

```go
TargetTokens: 200000  // Target 200k tokens to trigger auto-compact
QuerySize:    15000   // 15k tokens per query
MaxQueries:   1000    // Maximum number of queries to send
OutputDir:    "./tmp/test-sdk-auto-compact"
WorkDir:      "./tmp/agent-work-compact-test"
SessionID:    "test-auto-compact"
```

### Output

The test generates the following output files:

- `test_result.json` - Complete test results in JSON format
- `test_report.txt` - Human-readable test report
- `queries.json` - Detailed record of all queries
- `compact_events.json` - Record of all auto-compact events

### Exit Codes

- `0` - Auto-compact was successfully triggered
- `1` - Auto-compact was NOT triggered (test failed)

## How It Works

1. **Query Generation**: The tool generates technical content (software engineering topics) with approximately `QuerySize` tokens each

2. **Iterative Queries**: Sends queries repeatedly, accumulating context tokens

3. **Hook Detection**: Uses the SDK's `PreCompactHook` to detect when auto-compact occurs

4. **Event Recording**: Records all compact events with metadata:
   - Timestamp
   - Trigger type
   - Query index
   - Estimated tokens before compaction
   - Custom instructions (if any)
   - Session ID

5. **Report Generation**: Creates detailed reports and statistics

## Test Duration

The test may take several minutes to complete depending on:
- API response times
- Number of queries needed to reach target tokens
- Network latency

Typical duration: 5-15 minutes

## Example Output

```
=== Claude Agent SDK Auto-Compact Verification Test ===
Model: glm-4.7
Target Tokens: 200000
Query Size: 15000 tokens per query
Max Queries: 1000
Output Directory: ./tmp/test-sdk-auto-compact
============================================================

Query #1
Query Tokens: 15271
Estimated Context Before: 0 tokens
Progress: 0 / 200000 (0.0%)
...

============================================================
🔄 AUTO COMPACT TRIGGERED!
  Trigger Type: token_threshold
  Query Index: 14
  Estimated Tokens Before: 201358
  Timestamp: 2026-01-14T16:53:45Z
============================================================

✅ Auto-compact functionality is working!
```

## Troubleshooting

### Test Times Out

If the test takes too long, consider reducing:
- `TargetTokens` (e.g., 100000 instead of 200000)
- `QuerySize` (fewer tokens per query)

### No Auto-Compact Triggered

Possible reasons:
- SDK auto-compact threshold is higher than target
- SDK auto-compact is disabled
- API response tokens are very small

Check the generated `test_report.txt` for details.

### API Errors

Ensure your provider configuration is correct:
```bash
# Check current provider
ccc provider list

# Validate provider
ccc validate
```
