// Package llmparser provides a fault-tolerant JSON parser for LLM outputs.
//
// LLMs often generate JSON with formatting issues such as:
// - Missing quotes around keys
// - Single quotes instead of double quotes
// - Trailing commas
// - Comments (/* */ or //)
// - Markdown code block wrappers (```json ... ```)
// - Python constants (True, False, None)
// - Truncated JSON
//
// This package handles all these issues by first attempting standard JSON parsing,
// then falling back to jsonrepair if needed, and optionally validating against a JSON Schema.
package llmparser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kaptinlin/jsonrepair"
	"github.com/xeipuuv/gojsonschema"
)

// codeBlockRE is a precompiled regex for extracting markdown code blocks.
// Matches: ```json, ```, ```javascript, etc. (case-insensitive, dot matches newline)
var codeBlockRE = regexp.MustCompile(`(?si)\x60\x60\x60(json|jsonl|jsonlines|json5|jsonc|js|javascript)?\s*\n(.+?)\s*\x60\x60\x60`)

// Parse parses JSON from LLM output, with optional schema validation.
//
// The function attempts to extract JSON from markdown code blocks, or if no blocks
// are found, parses the entire text if it appears to be JSON. For each candidate,
// it first tries standard JSON parsing, then falls back to jsonrepair if needed.
// If a schema is provided, the parsed data is validated against it.
//
// Parameters:
//   - text: The LLM output text to parse
//   - jsonSchema: Optional JSON Schema as a map[string]interface{} for validation.
//     Set to nil to skip validation.
//
// Returns:
//   - interface{}: The parsed JSON data (typically map[string]interface{} or []interface{})
//   - error: An error if parsing fails, nil on success
//
// Example:
//
//	data, err := llmparser.Parse(`{"name": "John", "age": 25}`, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Result: %+v\n", data)
func Parse(text string, jsonSchema map[string]interface{}) (interface{}, error) {
	// Extract code blocks from text
	blocks := extractMarkdownCodeBlocks(text)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no JSON-like content found")
	}

	// Compile schema if provided
	var schema *gojsonschema.Schema
	var err error
	if jsonSchema != nil {
		schemaLoader := gojsonschema.NewGoLoader(jsonSchema)
		schema, err = gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON schema: %w", err)
		}
	}

	// Try parsing blocks in reverse order (last block is most likely the intended output)
	var errors []error
	for i := len(blocks) - 1; i >= 0; i-- {
		result, err := parseBlock(blocks[i], schema)
		if err == nil {
			return result, nil
		}
		errors = append(errors, err)
	}

	// All attempts failed, return the last error
	if len(errors) > 0 {
		return nil, fmt.Errorf("all parse attempts failed: %w", errors[len(errors)-1])
	}

	return nil, fmt.Errorf("no valid JSON found")
}

// ParseWithSchema parses JSON using a pre-compiled schema.
//
// This is a performance-optimized version for when you need to parse multiple
// texts with the same schema. Use this instead of Parse when doing repeated
// validations.
//
// Parameters:
//   - text: The LLM output text to parse
//   - schema: A pre-compiled gojsonschema.Schema
//
// Returns:
//   - interface{}: The parsed JSON data
//   - error: An error if parsing fails, nil on success
//
// Example:
//
//	schemaLoader := gojsonschema.NewStringLoader(`{"type": "object"}`)
//	schema, _ := gojsonschema.NewSchema(schemaLoader)
//
//	for _, text := range textList {
//	    data, err := llmparser.ParseWithSchema(text, schema)
//	    // ...
//	}
func ParseWithSchema(text string, schema *gojsonschema.Schema) (interface{}, error) {
	blocks := extractMarkdownCodeBlocks(text)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no JSON-like content found")
	}

	var errors []error
	for i := len(blocks) - 1; i >= 0; i-- {
		result, err := parseBlock(blocks[i], schema)
		if err == nil {
			return result, nil
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("all parse attempts failed: %w", errors[len(errors)-1])
	}

	return nil, fmt.Errorf("no valid JSON found")
}

// extractMarkdownCodeBlocks extracts JSON content from markdown code blocks.
// If no code blocks are found but the text looks like JSON, returns the text as a single block.
// Returns nil if no JSON-like content is found.
func extractMarkdownCodeBlocks(text string) []string {
	matches := codeBlockRE.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		if isLikelyJSON(text) {
			return []string{text}
		}
		return nil
	}

	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		// match[2] contains the code block content (without the ``` markers)
		if len(match) > 2 {
			blocks = append(blocks, strings.TrimSpace(match[2]))
		}
	}
	return blocks
}

// isLikelyJSON determines if text appears to be JSON.
// Checks for common JSON patterns: objects, arrays, strings, booleans, null, numbers.
func isLikelyJSON(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	// Check start characters for object, array, or string
	firstChar := text[0]
	if firstChar == '{' || firstChar == '[' || firstChar == '"' {
		return true
	}

	// Check end characters for object, array, or string
	lastChar := text[len(text)-1]
	if lastChar == '}' || lastChar == ']' || lastChar == '"' {
		return true
	}

	// Check for boolean and null literals
	switch text {
	case "true", "false", "null":
		return true
	}

	// Check for numbers (start with digit or minus sign)
	if len(text) > 0 {
		c := text[0]
		if c == '-' || (c >= '0' && c <= '9') {
			return true
		}
	}

	return false
}

// parseBlock attempts to parse a single block of text as JSON.
// First tries standard parsing, then falls back to jsonrepair.
// Optionally validates against a schema if provided.
func parseBlock(block string, schema *gojsonschema.Schema) (interface{}, error) {
	// Trim whitespace from block
	block = strings.TrimSpace(block)
	if block == "" {
		return nil, fmt.Errorf("empty block")
	}

	// Try standard JSON parsing first
	var result interface{}
	err := json.Unmarshal([]byte(block), &result)

	// If standard parsing fails, try jsonrepair
	if err != nil {
		repaired, repairErr := jsonrepair.JSONRepair(block)
		if repairErr != nil {
			return nil, fmt.Errorf("JSON repair failed: %w, original error: %w", repairErr, err)
		}
		err = json.Unmarshal([]byte(repaired), &result)
		if err != nil {
			return nil, fmt.Errorf("repaired JSON parsing failed: %w", err)
		}
	}

	// Validate against schema if provided
	if schema != nil {
		documentLoader := gojsonschema.NewGoLoader(result)
		validateResult, validateErr := schema.Validate(documentLoader)
		if validateErr != nil {
			return nil, fmt.Errorf("schema validation error: %w", validateErr)
		}
		if !validateResult.Valid() {
			var errMsg strings.Builder
			errMsg.WriteString("schema validation failed:")
			for _, desc := range validateResult.Errors() {
				errMsg.WriteString(fmt.Sprintf("\n  - %s", desc))
			}
			return nil, fmt.Errorf("%s", errMsg.String())
		}
	}

	return result, nil
}
