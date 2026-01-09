# Structured Outputs Implementation Plan

## Overview

This document outlines a comprehensive implementation plan for adding **Structured Outputs** support to the Go Claude Agent SDK, matching the functionality provided by the Python SDK.

**Reference Issue**: https://github.com/schlunsen/claude-agent-sdk-go/issues/32

**Reference Documentation**: https://platform.claude.com/docs/en/agent-sdk/structured-outputs

---

## Table of Contents

1. [Python SDK Analysis](#1-python-sdk-analysis)
2. [Go SDK Current State](#2-go-sdk-current-state)
3. [Implementation Architecture](#3-implementation-architecture)
4. [Detailed Implementation Steps](#4-detailed-implementation-steps)
5. [Testing Strategy](#5-testing-strategy)
6. [Example Usage](#6-example-usage)
7. [Validation Checklist](#7-validation-checklist)

---

## 1. Python SDK Analysis

### 1.1 Type Definitions (`types.py`)

**ClaudeAgentOptions.output_format** (lines 674-676):
```python
# Output format for structured outputs (matches Messages API structure)
# Example: {"type": "json_schema", "schema": {"type": "object", "properties": {...}}}
output_format: dict[str, Any] | None = None
```

**ResultMessage.structured_output** (line 600):
```python
structured_output: Any = None
```

### 1.2 Transport Layer (`subprocess_cli.py`)

In `_build_command()` method (lines 316-325):
```python
# Extract schema from output_format structure if provided
# Expected: {"type": "json_schema", "schema": {...}}
if (
    self._options.output_format is not None
    and isinstance(self._options.output_format, dict)
    and self._options.output_format.get("type") == "json_schema"
):
    schema = self._options.output_format.get("schema")
    if schema is not None:
        cmd.extend(["--json-schema", json.dumps(schema)])
```

**Key Point**: The CLI argument is `--json-schema` with the JSON-serialized schema as the value.

### 1.3 Message Parser (`message_parser.py`)

Line 156 in `parse_message()`:
```python
structured_output=data.get("structured_output"),
```

The parser extracts the `structured_output` field from the result message data.

### 1.4 Test Cases (`test_structured_output.py`)

Key test patterns:
- Simple schema with primitives (number, boolean)
- Nested objects with multiple levels
- Arrays of items
- Enum constraints
- Integration with tool use (Bash, Grep, etc.)

---

## 2. Go SDK Current State

### 2.1 Files Analyzed

| File | Purpose | Structured Outputs Support |
|------|---------|---------------------------|
| `types/options.go` | Configuration options | **MISSING** - No `OutputFormat` field |
| `types/messages.go` | Message types | **MISSING** - No `StructuredOutput` in `ResultMessage` |
| `internal/transport/subprocess_cli.go` | CLI subprocess management | **MISSING** - No `--json-schema` argument |
| `internal/message_parser.go` | Message parsing | **MISSING** - No `structured_output` field parsing |

### 2.2 Existing Patterns

The Go SDK follows these conventions:
- Builder pattern for options: `WithXxx()` methods
- Interface-based message types with discriminator (`GetMessageType()`)
- Custom JSON unmarshaling for complex types
- Type-safe errors using custom error types

---

## 3. Implementation Architecture

### 3.1 Type System Design

```
OutputFormat (new type)
├── Type: string ("json_schema")
└── Schema: JSONSchema (interface{} or map[string]interface{})

ResultMessage (modified)
└── StructuredOutput: interface{} (parsed JSON)
```

### 3.2 Data Flow

```
User Code
    │
    ▼
ClaudeAgentOptions.WithOutputFormat(schema)
    │
    ▼
SubprocessCLITransport.buildCommandArgs()
    │  adds: --json-schema <json>
    ▼
Claude CLI
    │  processes with schema validation
    ▼
Result Message with structured_output field
    │
    ▼
types.UnmarshalMessage()
    │  parses structured_output
    ▼
User receives ResultMessage.StructuredOutput
```

---

## 4. Detailed Implementation Steps

### Phase 1: Type Definitions (`types/`)

#### Step 1.1: Add `OutputFormat` type to `options.go`

**Location**: `types/options.go` (after line 248, before `NewClaudeAgentOptions`)

```go
// OutputFormat represents the output format configuration for structured outputs.
// This enables agents to return validated JSON matching a specific JSON Schema.
type OutputFormat struct {
	// Type must be "json_schema" for structured outputs
	Type string `json:"type"`

	// Schema is the JSON Schema definition for the output format
	// Can be a map[string]interface{} for dynamic schemas
	Schema interface{} `json:"schema"`
}

// NewOutputFormat creates a new OutputFormat with the given JSON schema.
// The schema can be a map[string]interface{} or any JSON-serializable type.
func NewOutputFormat(schema interface{}) *OutputFormat {
	return &OutputFormat{
		Type:   "json_schema",
		Schema: schema,
	}
}
```

#### Step 1.2: Add `OutputFormat` field to `ClaudeAgentOptions`

**Location**: `types/options.go` (in `ClaudeAgentOptions` struct, after line 248)

```go
// OutputFormat for structured outputs (validates JSON responses against a schema)
OutputFormat *OutputFormat `json:"output_format,omitempty"`
```

#### Step 1.3: Add builder method

**Location**: `types/options.go` (after `WithAllowDangerouslySkipPermissions`, around line 569)

```go
// WithOutputFormat sets the structured output format with a JSON schema.
// The agent's final response will be validated against this schema.
// The schema can be a map[string]interface{} representing a JSON Schema.
//
// Example:
//
//	schema := map[string]interface{}{
//	    "type": "object",
//	    "properties": map[string]interface{}{
//	        "name": map[string]interface{}{"type": "string"},
//	        "count": map[string]interface{}{"type": "number"},
//	    },
//	    "required": []string{"name", "count"},
//	}
//	options.WithOutputFormat(schema)
func (o *ClaudeAgentOptions) WithOutputFormat(schema interface{}) *ClaudeAgentOptions {
	o.OutputFormat = NewOutputFormat(schema)
	return o
}
```

#### Step 1.4: Add `StructuredOutput` to `ResultMessage`

**Location**: `types/messages.go` (in `ResultMessage` struct, after line 357)

```go
// StructuredOutput contains the validated JSON output when using structured outputs.
// This is present when OutputFormat is specified in the query options.
// The value is already parsed as interface{} and can be type-asserted or re-marshaled.
StructuredOutput interface{} `json:"structured_output,omitempty"`
```

---

### Phase 2: Transport Layer (`internal/transport/`)

#### Step 2.1: Modify `buildCommandArgs()` in `subprocess_cli.go`

**Location**: `internal/transport/subprocess_cli.go` (in `buildCommandArgs()`, after line 451, before `return args`)

```go
// Add structured output format (JSON schema) if specified
if t.options != nil && t.options.OutputFormat != nil {
	of := t.options.OutputFormat

	// Validate type is "json_schema"
	if of.Type != "json_schema" {
		t.logger.Warning("Invalid output format type: %s (expected 'json_schema')", of.Type)
	} else {
		// Marshal schema to JSON
		schemaJSON, err := json.Marshal(of.Schema)
		if err != nil {
			t.logger.Warning("Failed to marshal output format schema: %v", err)
		} else {
			// Add --json-schema argument with the schema JSON
			args = append(args, "--json-schema", string(schemaJSON))
			t.logger.Debug("Setting structured output format with JSON schema")
		}
	}
}
```

---

### Phase 3: Message Parsing (`internal/`)

#### Step 3.1: Update `message_parser.go` (no changes needed)

**Note**: The existing `types.UnmarshalMessage()` function automatically handles new fields via JSON unmarshaling. Since `ResultMessage` now has the `StructuredOutput` field, it will be parsed automatically without changes to `message_parser.go`.

However, we should add a test to verify this behavior (see Testing section).

---

### Phase 4: Helper Types (Optional Enhancement)

#### Step 4.1: Add JSON Schema helper types (optional but recommended)

**Location**: Create new file `types/json_schema.go`

```go
package types

// JSONSchema represents a JSON Schema for type-safe schema construction.
// This is optional - users can also use map[string]interface{} directly.
type JSONSchema map[string]interface{}

// Common JSON Schema keywords
const (
	JSONSchemaTypeObject  = "object"
	JSONSchemaTypeArray   = "array"
	JSONSchemaTypeString  = "string"
	JSONSchemaTypeNumber  = "number"
	JSONSchemaTypeInteger = "integer"
	JSONSchemaTypeBoolean = "boolean"
	JSONSchemaTypeNull    = "null"
)

// NewObjectSchema creates a new object schema with properties and required fields.
func NewObjectSchema(properties map[string]JSONSchema, required []string) JSONSchema {
	props := make(map[string]interface{}, len(properties))
	for k, v := range properties {
		props[k] = v
	}

	return JSONSchema{
		"type":       JSONSchemaTypeObject,
		"properties": props,
		"required":   required,
	}
}

// NewArraySchema creates a new array schema with item type.
func NewArraySchema(items JSONSchema) JSONSchema {
	return JSONSchema{
		"type":  JSONSchemaTypeArray,
		"items": items,
	}
}

// NewStringSchema creates a string schema with optional format/enum.
func NewStringSchema(format string, enum []string) JSONSchema {
	schema := JSONSchema{"type": JSONSchemaTypeString}
	if format != "" {
		schema["format"] = format
	}
	if len(enum) > 0 {
		schema["enum"] = enum
	}
	return schema
}

// NewNumberSchema creates a number schema.
func NewNumberSchema() JSONSchema {
	return JSONSchema{"type": JSONSchemaTypeNumber}
}

// NewBooleanSchema creates a boolean schema.
func NewBooleanSchema() JSONSchema {
	return JSONSchema{"type": JSONSchemaTypeBoolean}
}
```

**Note**: This is optional but provides a more idiomatic Go API for building schemas. Users can still use raw `map[string]interface{}`.

---

## 5. Testing Strategy

### 5.1 Unit Tests

#### Test 5.1.1: `types/options_test.go` - Test `WithOutputFormat()`

```go
func TestClaudeAgentOptions_WithOutputFormat(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
		"required": []string{"name"},
	}

	opts := NewClaudeAgentOptions().
		WithOutputFormat(schema).

	assert.NotNil(t, opts.OutputFormat)
	assert.Equal(t, "json_schema", opts.OutputFormat.Type)
	assert.Equal(t, schema, opts.OutputFormat.Schema)
}
```

#### Test 5.1.2: `types/messages_test.go` - Test `ResultMessage.StructuredOutput` parsing

```go
func TestUnmarshalResultMessage_WithStructuredOutput(t *testing.T) {
	jsonData := `{
		"type": "result",
		"subtype": "success",
		"duration_ms": 1000,
		"duration_api_ms": 500,
		"is_error": false,
		"num_turns": 1,
		"session_id": "test-session",
		"structured_output": {
			"name": "test",
			"count": 42
		}
	}`

	msg, err := UnmarshalMessage([]byte(jsonData))
	require.NoError(t, err)
	require.IsType(t, &ResultMessage{}, msg)

	result := msg.(*ResultMessage)
	require.NotNil(t, result.StructuredOutput)

	// Verify structured output content
	output, ok := result.StructuredOutput.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test", output["name"])
	assert.Equal(t, float64(42), output["count"])
}
```

#### Test 5.1.3: `internal/transport/transport_test.go` - Test CLI argument generation

```go
func TestBuildCommandArgs_WithOutputFormat(t *testing.T) {
	logger := log.NewNoOpLogger()
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{"type": "number"},
		},
	}

	opts := &types.ClaudeAgentOptions{
		OutputFormat: types.NewOutputFormat(schema),
	}

	transport := NewSubprocessCLITransport("claude", "", nil, logger, "", opts)
	args := transport.buildCommandArgs()

	// Find --json-schema argument
	found := false
	for i, arg := range args {
		if arg == "--json-schema" && i+1 < len(args) {
			// Verify the JSON schema
			var parsed map[string]interface{}
			err := json.Unmarshal([]byte(args[i+1]), &parsed)
			require.NoError(t, err)
			assert.Equal(t, "object", parsed["type"])
			found = true
			break
		}
	}

	assert.True(t, found, "--json-schema argument not found")
}
```

### 5.2 Integration Tests

#### Test 5.2.1: Simple structured output

```go
func TestStructuredOutput_Simple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_count": map[string]interface{}{"type": "number"},
			"has_tests":  map[string]interface{}{"type": "boolean"},
		},
		"required": []string{"file_count", "has_tests"},
	}

	opts := NewClaudeAgentOptions().
		WithOutputFormat(schema).
		WithPermissionMode(PermissionModeAcceptEdits)

	ctx := context.Background()
	messages, err := Query(ctx, "Count Go files and check for tests", opts)
	require.NoError(t, err)

	// Find result message
	var result *ResultMessage
	for msg := range messages {
		if rm, ok := msg.(*ResultMessage); ok {
			result = rm
			break
		}
	}

	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotNil(t, result.StructuredOutput)

	// Validate output
	output, ok := result.StructuredOutput.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, output, "file_count")
	assert.Contains(t, output, "has_tests")
}
```

#### Test 5.2.2: Nested structured output

```go
func TestStructuredOutput_Nested(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"analysis": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"word_count": map[string]interface{}{"type": "integer"},
				},
				"required": []string{"word_count"},
			},
			"items": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
		},
		"required": []string{"analysis", "items"},
	}

	opts := NewClaudeAgentOptions().WithOutputFormat(schema)
	// ... rest of test
}
```

---

## 6. Example Usage

### 6.1 Basic Usage

```go
package main

import (
    "context"
    "fmt"

    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
    ctx := context.Background()

    // Define JSON schema for expected output
    schema := map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "file_count": map[string]interface{}{"type": "number"},
            "has_readme": map[string]interface{}{"type": "boolean"},
            "languages": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{"type": "string"},
            },
        },
        "required": []string{"file_count", "has_readme", "languages"},
    }

    opts := types.NewClaudeAgentOptions().
        WithOutputFormat(schema).
        WithPermissionMode(types.PermissionModeAcceptEdits)

    messages, err := sdk.Query(ctx, "Analyze this codebase", opts)
    if err != nil {
        panic(err)
    }

    for msg := range messages {
        if result, ok := msg.(*types.ResultMessage); ok {
            if result.StructuredOutput != nil {
                output := result.StructuredOutput.(map[string]interface{})
                fmt.Printf("Files: %v\n", output["file_count"])
                fmt.Printf("Has README: %v\n", output["has_readme"])
                fmt.Printf("Languages: %v\n", output["languages"])
            }
        }
    }
}
```

### 6.2 Using Helper Types (if implemented)

```go
package main

import (
    "context"
    "fmt"

    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
    ctx := context.Background()

    // Build schema using helper types
    schema := types.NewObjectSchema(
        map[string]types.JSONSchema{
            "todo": types.NewObjectSchema(
                map[string]types.JSONSchema{
                    "text": types.NewStringSchema("", nil),
                    "done": types.NewBooleanSchema(),
                },
                []string{"text", "done"},
            ),
        },
        []string{"todo"},
    )

    opts := types.NewClaudeAgentOptions().
        WithOutputFormat(schema)

    messages, err := sdk.Query(ctx, "Create a todo item", opts)
    // ... process messages
}
```

---

## 7. Validation Checklist

Use this checklist to validate the implementation:

### Code Completeness

- [ ] `OutputFormat` type defined in `types/options.go`
- [ ] `ClaudeAgentOptions.OutputFormat` field added
- [ ] `WithOutputFormat()` builder method added
- [ ] `ResultMessage.StructuredOutput` field added
- [ ] `buildCommandArgs()` includes `--json-schema` argument
- [ ] JSON marshaling of schema works correctly

### Functionality

- [ ] CLI receives `--json-schema` with valid JSON
- [ ] Result message includes `structured_output` field when schema is provided
- [ ] Structured output is parsed correctly from JSON
- [ ] Type assertion to `map[string]interface{}` works
- [ ] Nested objects and arrays are parsed correctly

### Error Handling

- [ ] Invalid JSON schema produces a warning (doesn't crash)
- [ ] Invalid output format type produces a warning
- [ ] Missing `structured_output` in result is handled gracefully (nil)

### Testing

- [ ] Unit tests for `WithOutputFormat()`
- [ ] Unit tests for `ResultMessage.StructuredOutput` parsing
- [ ] Unit tests for CLI argument generation
- [ ] Integration test with simple schema
- [ ] Integration test with nested schema
- [ ] Integration test with array schema
- [ ] Integration test with enum constraints

### Documentation

- [ ] Go doc comments on all new types
- [ ] Example usage in comments
- [ ] README update with structured outputs section
- [ ] Feature parity document updated

### Compatibility

- [ ] Python SDK behavior matched
- [ ] CLI argument format matches Python SDK (`--json-schema`)
- [ ] Result message format matches Python SDK
- [ ] No breaking changes to existing API

---

## 8. Files to Modify

| File | Changes | Lines (approx) |
|------|---------|----------------|
| `types/options.go` | Add `OutputFormat` type, field, and `WithOutputFormat()` | +60 |
| `types/messages.go` | Add `StructuredOutput` field to `ResultMessage` | +2 |
| `internal/transport/subprocess_cli.go` | Add `--json-schema` argument in `buildCommandArgs()` | +20 |
| `types/json_schema.go` | (Optional) New file with helper types | +80 |
| `types/options_test.go` | Add unit tests | +40 |
| `types/messages_test.go` | Add parsing tests | +50 |
| `internal/transport/transport_test.go` | Add CLI arg tests | +60 |
| `tests/integration_test.go` | Add integration tests | +150 |

**Total**: ~462 lines added (excluding optional helpers)

---

## 9. Estimated Effort

| Phase | Task | Effort |
|-------|------|--------|
| 1 | Type definitions | 1-2 hours |
| 2 | Transport layer | 1 hour |
| 3 | Message parsing | 0.5 hours (verify) |
| 4 | Optional helpers | 1-2 hours |
| 5 | Unit tests | 2-3 hours |
| 6 | Integration tests | 2-3 hours |
| 7 | Documentation | 1 hour |
| **Total** | | **8-12 hours** |

---

## 10. References

- **Python SDK**: `/home/ubuntu/dev/claude-agent-sdk-python/`
  - `src/claude_agent_sdk/types.py` (lines 674-676, 600)
  - `src/claude_agent_sdk/_internal/transport/subprocess_cli.py` (lines 316-325)
  - `src/claude_agent_sdk/_internal/message_parser.py` (line 156)
  - `e2e-tests/test_structured_output.py`

- **Go SDK**: `/home/ubuntu/dev/claude-agent-sdk-go/`
  - `types/options.go`
  - `types/messages.go`
  - `internal/transport/subprocess_cli.go`
  - `internal/message_parser.go`

- **Official Documentation**: https://platform.claude.com/docs/en/agent-sdk/structured-outputs
- **Issue**: https://github.com/schlunsen/claude-agent-sdk-go/issues/32
