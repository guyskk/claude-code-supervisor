package prettyjson

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshal(t *testing.T) {
	tests := []struct {
		name           string
		input          interface{}
		expected       string
		mustContain    []string // Optional: check that result contains these strings
		mustNotContain []string // Optional: check that result does not contain these
	}{
		{
			name:     "simple object",
			input:    map[string]string{"name": "test"},
			expected: "{\n    \"name\": \"test\"\n}",
		},
		{
			name:     "Chinese characters",
			input:    map[string]string{"message": "你好世界"},
			expected: "{\n    \"message\": \"你好世界\"\n}",
		},
		{
			name:     "nested object",
			input:    map[string]interface{}{"user": map[string]string{"name": "Alice", "role": "admin"}},
			expected: "{\n    \"user\": {\n        \"name\": \"Alice\",\n        \"role\": \"admin\"\n    }\n}",
		},
		{
			name:     "array",
			input:    []string{"a", "b", "c"},
			expected: "[\n    \"a\",\n    \"b\",\n    \"c\"\n]",
		},
		{
			name:     "null value",
			input:    map[string]interface{}{"value": nil},
			expected: "{\n    \"value\": null\n}",
		},
		{
			name:     "number",
			input:    map[string]interface{}{"count": 42, "price": 3.14},
			expected: "{\n    \"count\": 42,\n    \"price\": 3.14\n}",
		},
		{
			name:     "boolean",
			input:    map[string]bool{"active": true, "deleted": false},
			expected: "{\n    \"active\": true,\n    \"deleted\": false\n}",
		},
		{
			name:     "HTML characters should not be escaped",
			input:    map[string]string{"html": "<div>content</div>"},
			expected: "{\n    \"html\": \"<div>content</div>\"\n}",
		},
		{
			name:  "mixed Chinese and English",
			input: map[string]string{"title": "标题Title", "content": "内容Content"},
			// Map key order is not guaranteed in Go, so we check for content presence
			mustContain: []string{"\"title\": \"标题Title\"", "\"content\": \"内容Content\""},
		},
		{
			name:     "emoji",
			input:    map[string]string{"emoji": "😀🎉"},
			expected: "{\n    \"emoji\": \"😀🎉\"\n}",
		},
		{
			name:     "special characters",
			input:    map[string]string{"special": "\n\t\r"},
			expected: "{\n    \"special\": \"\\n\\t\\r\"\n}",
		},
		{
			name:     "empty object",
			input:    map[string]string{},
			expected: "{}",
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			// Check exact match if expected is provided
			if tt.expected != "" && string(result) != tt.expected {
				t.Errorf("Marshal() = %q, want %q", string(result), tt.expected)
			}

			// Check mustContain strings
			for _, s := range tt.mustContain {
				if !strings.Contains(string(result), s) {
					t.Errorf("Marshal() result should contain %q, but got: %q", s, string(result))
				}
			}

			// Check mustNotContain strings
			for _, s := range tt.mustNotContain {
				if strings.Contains(string(result), s) {
					t.Errorf("Marshal() result should NOT contain %q, but got: %q", s, string(result))
				}
			}
		})
	}
}

func TestMarshal_SetEscapeHTML(t *testing.T) {
	// Test that SetEscapeHTML(false) works correctly
	// by comparing with standard library encoder with SetEscapeHTML(false)
	input := map[string]string{
		"chinese": "你好",
		"emoji":   "😀",
		"html":    "<div>",
	}

	// Create standard encoder with SetEscapeHTML(false)
	var stdBuf strings.Builder
	stdEncoder := json.NewEncoder(&stdBuf)
	stdEncoder.SetEscapeHTML(false)
	stdEncoder.SetIndent("", "    ")
	if err := stdEncoder.Encode(input); err != nil {
		t.Fatalf("Standard encoder error = %v", err)
	}
	stdResult := strings.TrimRight(stdBuf.String(), "\n")

	// Our prettyjson should produce same result
	prettyResult, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if string(prettyResult) != stdResult {
		t.Errorf("Marshal() = %q, want %q (same as std lib with SetEscapeHTML(false))", string(prettyResult), stdResult)
	}

	// Verify actual characters are present, not escaped
	if !strings.Contains(string(prettyResult), "你好") {
		t.Error("Result should contain Chinese characters, not escaped unicode")
	}
	if !strings.Contains(string(prettyResult), "😀") {
		t.Error("Result should contain emoji, not escaped unicode")
	}
	if !strings.Contains(string(prettyResult), "<div>") {
		t.Error("Result should contain unescaped HTML")
	}

	// Should NOT contain escaped unicode sequences
	if strings.Contains(string(prettyResult), "\\u") {
		t.Error("Result should NOT contain escaped unicode sequences")
	}
}

func TestMarshal_ErrorCases(t *testing.T) {
	// Test with unmarshalable types
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "channel",
			input: make(chan int),
		},
		{
			name:  "function",
			input: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Marshal(tt.input)
			if err == nil {
				t.Error("Marshal() should return error for unmarshalable type")
			}
		})
	}
}

func TestMarshal_NoTrailingNewline(t *testing.T) {
	input := map[string]string{"key": "value"}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Check that result does not end with newline
	if len(result) > 0 && result[len(result)-1] == '\n' {
		t.Error("Marshal() should not add trailing newline")
	}
}

func TestMarshal_ComplexNestedStructure(t *testing.T) {
	input := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "张三",
			"age":  30,
			"tags": []string{"developer", "golang"},
			"address": map[string]string{
				"city":    "北京",
				"country": "中国",
			},
		},
		"active": true,
		"score":  95.5,
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify Chinese characters are not escaped
	if !strings.Contains(string(result), "张三") {
		t.Error("Chinese characters should not be escaped")
	}
	if !strings.Contains(string(result), "北京") {
		t.Error("Chinese characters should not be escaped")
	}
	if !strings.Contains(string(result), "中国") {
		t.Error("Chinese characters should not be escaped")
	}

	// Verify proper indentation (4 spaces)
	if !strings.Contains(string(result), "    ") {
		t.Error("Should have 4-space indentation")
	}

	// Should not have tabs
	if strings.Contains(string(result), "\t") {
		t.Error("Should not use tabs for indentation")
	}
}

func TestMarshal_UnicodeEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		checks []string // Strings that must be in result
	}{
		{
			name:   "arabic",
			input:  map[string]string{"text": "مرحبا"},
			checks: []string{"مرحبا"},
		},
		{
			name:   "japanese",
			input:  map[string]string{"text": "こんにちは"},
			checks: []string{"こんにちは"},
		},
		{
			name:   "korean",
			input:  map[string]string{"text": "안녕하세요"},
			checks: []string{"안녕하세요"},
		},
		{
			name:   "russian",
			input:  map[string]string{"text": "Привет"},
			checks: []string{"Привет"},
		},
		{
			name:   "mixed unicode",
			input:  map[string]string{"mixed": "Hello你好🙂مرحبا"},
			checks: []string{"Hello你好🙂مرحبا"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			for _, check := range tt.checks {
				if !strings.Contains(string(result), check) {
					t.Errorf("Result should contain %q, got: %q", check, string(result))
				}
			}

			// Should not have escaped unicode
			if strings.Contains(string(result), "\\u") {
				t.Errorf("Result should NOT contain escaped unicode, got: %q", string(result))
			}
		})
	}
}
