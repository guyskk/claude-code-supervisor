package llmparser

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

// TestExtractMarkdownCodeBlocks tests the extraction of markdown code blocks.
func TestExtractMarkdownCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "标准json代码块",
			input:    "```json\n{\"name\": \"John\"}\n```",
			expected: []string{`{"name": "John"}`},
		},
		{
			name:     "无标记代码块",
			input:    "```\n{\"name\": \"John\"}\n```",
			expected: []string{`{"name": "John"}`},
		},
		{
			name:     "javascript代码块",
			input:    "```javascript\n{\"name\": \"John\"}\n```",
			expected: []string{`{"name": "John"}`},
		},
		{
			name:     "js代码块",
			input:    "```js\n{\"name\": \"John\"}\n```",
			expected: []string{`{"name": "John"}`},
		},
		{
			name:     "json5代码块",
			input:    "```json5\n{\"name\": \"John\"}\n```",
			expected: []string{`{"name": "John"}`},
		},
		{
			name:     "jsonc代码块",
			input:    "```jsonc\n{\"name\": \"John\"}\n```",
			expected: []string{`{"name": "John"}`},
		},
		{
			name:     "大小写不敏感",
			input:    "```JSON\n{\"name\": \"John\"}\n```",
			expected: []string{`{"name": "John"}`},
		},
		{
			name:  "多代码块",
			input: "```json\n{\"first\": 1}\n```\n```json\n{\"second\": 2}\n```",
			expected: []string{
				`{"first": 1}`,
				`{"second": 2}`,
			},
		},
		{
			name:     "无代码块但像JSON - 对象",
			input:    `{"name": "John"}`,
			expected: []string{`{"name": "John"}`},
		},
		{
			name:     "无代码块但像JSON - 数组",
			input:    `[1, 2, 3]`,
			expected: []string{`[1, 2, 3]`},
		},
		{
			name:     "无代码块但像JSON - 字符串",
			input:    `"hello"`,
			expected: []string{`"hello"`},
		},
		{
			name:     "无代码块但像JSON - 布尔值",
			input:    `true`,
			expected: []string{`true`},
		},
		{
			name:     "无代码块但像JSON - null",
			input:    `null`,
			expected: []string{`null`},
		},
		{
			name:     "无代码块但像JSON - 数字",
			input:    `42`,
			expected: []string{`42`},
		},
		{
			name:     "无代码块但像JSON - 负数",
			input:    `-42`,
			expected: []string{`-42`},
		},
		{
			name:     "无代码块但像JSON - 以{开头",
			input:    `{"key": "value"`,
			expected: []string{`{"key": "value"`},
		},
		{
			name:     "无代码块但像JSON - 以}结尾",
			input:    `"key": "value"}`,
			expected: []string{`"key": "value"}`},
		},
		{
			name:     "无代码块且不像JSON",
			input:    "这是一段普通文本",
			expected: nil,
		},
		{
			name:     "空字符串",
			input:    "",
			expected: nil,
		},
		{
			name:     "只有空白字符",
			input:    "   \n\t  ",
			expected: nil,
		},
		{
			name:     "带额外文本的代码块",
			input:    "分析过程如下：\n```json\n{\"result\": \"success\"}\n```\n分析完毕",
			expected: []string{`{"result": "success"}`},
		},
		{
			name:     "代码块内有多余空格",
			input:    "```json  \n  {\"name\": \"John\"}  \n  ```",
			expected: []string{`{"name": "John"}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMarkdownCodeBlocks(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsLikelyJSON tests the JSON likelihood detection.
func TestIsLikelyJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"对象开头", `{"key": "value"}`, true},
		{"数组开头", `[1, 2, 3]`, true},
		{"字符串开头", `"hello"`, true},
		{"对象结尾", `something}`, true},
		{"数组结尾", `something]`, true},
		{"字符串结尾", `something"`, true},
		{"布尔值true", `true`, true},
		{"布尔值false", `false`, true},
		{"null值", `null`, true},
		{"正数", `42`, true},
		{"负数", `-42`, true},
		{"小数", `3.14`, true},
		{"普通文本", `hello world`, false},
		{"空字符串", ``, false},
		{"只有空白", `   `, false},
		{"以字母开头但不匹配关键字", `abc`, false},
		{"带前缀空格的对象", `  {"key": "value"}`, true},
		{"带后缀空格的数组", `[1, 2, 3]  `, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseValidJSON tests parsing valid JSON.
func TestParseValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "简单对象",
			input: `{"name": "John", "age": 25}`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "带布尔值",
			input: `{"active": true, "deleted": false}`,
			expected: map[string]interface{}{
				"active":  true,
				"deleted": false,
			},
		},
		{
			name:  "带null值",
			input: `{"value": null}`,
			expected: map[string]interface{}{
				"value": nil,
			},
		},
		{
			name:  "嵌套对象",
			input: `{"user": {"name": "John", "age": 25}}`,
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
					"age":  float64(25),
				},
			},
		},
		{
			name:  "数组",
			input: `{"items": [1, 2, 3]}`,
			expected: map[string]interface{}{
				"items": []interface{}{float64(1), float64(2), float64(3)},
			},
		},
		{
			name:  "字符串数组",
			input: `{"tags": ["a", "b", "c"]}`,
			expected: map[string]interface{}{
				"tags": []interface{}{"a", "b", "c"},
			},
		},
		{
			name:  "复杂嵌套",
			input: `{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`,
			expected: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"id": float64(1), "name": "Alice"},
					map[string]interface{}{"id": float64(2), "name": "Bob"},
				},
			},
		},
		{
			name:     "纯数组",
			input:    `[1, 2, 3]`,
			expected: nil, // Will be []interface{}, not map
		},
		{
			name:     "纯字符串",
			input:    `"hello"`,
			expected: nil, // Will be string, not map
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Parse(tt.input, nil)
			assert.Nil(t, err)
			assert.NotNil(t, data)

			// Only check map type results
			if tt.expected != nil {
				result, ok := data.(map[string]interface{})
				assert.True(t, ok, "Result should be a map")
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestParseInvalidJSONWithRepair tests parsing invalid JSON that can be repaired.
func TestParseInvalidJSONWithRepair(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "缺失引号",
			input: `{name: 'John', age: 25}`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "尾随逗号",
			input: `{"name": "John", "age": 25,}`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "单引号",
			input: `{'name': 'John', 'age': 25}`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "带注释 - 单行",
			input: `{"name": "John", /* comment */ "age": 25}`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "带注释 - 多行",
			input: `{"name": "John", /* multi-line\ncomment */ "age": 25}`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "带注释 - 双斜杠（行内）",
			input: `{"name": "John", // comment\n"age": 25}`,
			// Note: jsonrepair removes content after // comment, resulting in just {"name": "John"}
			expected: map[string]interface{}{
				"name": "John",
			},
		},
		{
			name:  "Python常量 - True",
			input: `{"name": "John", "active": True}`,
			expected: map[string]interface{}{
				"name":   "John",
				"active": true,
			},
		},
		{
			name:  "Python常量 - False",
			input: `{"name": "John", "deleted": False}`,
			expected: map[string]interface{}{
				"name":    "John",
				"deleted": false,
			},
		},
		{
			name:  "Python常量 - None",
			input: `{"name": "John", "value": None}`,
			expected: map[string]interface{}{
				"name":  "John",
				"value": nil,
			},
		},
		{
			name:  "Python常量 - 混合",
			input: `{"active": True, "deleted": False, "value": None}`,
			expected: map[string]interface{}{
				"active":  true,
				"deleted": false,
				"value":   nil,
			},
		},
		{
			name:  "数组尾随逗号",
			input: `{"items": [1, 2, 3,]}`,
			expected: map[string]interface{}{
				"items": []interface{}{float64(1), float64(2), float64(3)},
			},
		},
		{
			name:  "省略号",
			input: `{"items": [1, 2, ...]}`,
			expected: map[string]interface{}{
				"items": []interface{}{float64(1), float64(2)},
			},
		},
		{
			name:  "截断的对象",
			input: `{"name": "John", "age": 25`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "截断的数组",
			input: `{"items": [1, 2, 3`,
			expected: map[string]interface{}{
				"items": []interface{}{float64(1), float64(2), float64(3)},
			},
		},
		{
			name:  "无引号的键",
			input: `{name: "John", age: 25}`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "混合单双引号",
			input: `{"name": 'John', "age": 25}`,
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Parse(tt.input, nil)
			assert.Nil(t, err, "Should parse without error")
			assert.NotNil(t, data, "Should return data")

			result, ok := data.(map[string]interface{})
			assert.True(t, ok, "Result should be a map")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseWithSchemaValidation tests JSON Schema validation.
func TestParseWithSchemaValidation(t *testing.T) {
	// Define schema
	var schema map[string]interface{}
	err := json.Unmarshal([]byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name", "age"]
	}`), &schema)
	assert.Nil(t, err)

	t.Run("有效数据", func(t *testing.T) {
		data, err := Parse(`{"name": "John", "age": 25}`, schema)
		assert.Nil(t, err)
		assert.NotNil(t, data)

		result, ok := data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "John", result["name"])
		assert.Equal(t, float64(25), result["age"])
	})

	t.Run("缺少必需字段 - name", func(t *testing.T) {
		data, err := Parse(`{"age": 25}`, schema)
		assert.NotNil(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("缺少必需字段 - age", func(t *testing.T) {
		data, err := Parse(`{"name": "John"}`, schema)
		assert.NotNil(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "age")
	})

	t.Run("类型错误 - age不是整数", func(t *testing.T) {
		data, err := Parse(`{"name": "John", "age": "twenty-five"}`, schema)
		assert.NotNil(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "integer")
	})

	t.Run("类型错误 - name不是字符串", func(t *testing.T) {
		data, err := Parse(`{"name": 123, "age": 25}`, schema)
		assert.NotNil(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "string")
	})

	t.Run("值超出范围 - 负数年龄", func(t *testing.T) {
		data, err := Parse(`{"name": "John", "age": -5}`, schema)
		assert.NotNil(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "greater than or equal to")
	})

	t.Run("额外字段允许", func(t *testing.T) {
		data, err := Parse(`{"name": "John", "age": 25, "city": "NYC"}`, schema)
		assert.Nil(t, err)
		assert.NotNil(t, data)
	})

	t.Run("修复后的JSON通过校验", func(t *testing.T) {
		// Input with single quotes that gets repaired
		data, err := Parse(`{'name': 'John', 'age': 25}`, schema)
		assert.Nil(t, err)
		assert.NotNil(t, data)
	})
}

// TestParseFromMarkdownCodeBlock tests parsing JSON from markdown code blocks.
func TestParseFromMarkdownCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "简单代码块",
			input: "这是一个分析过程...\n\n```json\n{\"name\": \"John\", \"age\": 25}\n```\n\n分析完毕。",
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "无语言标记",
			input: "```\n{\"name\": \"John\"}\n```",
			expected: map[string]interface{}{
				"name": "John",
			},
		},
		{
			name:  "javascript标记",
			input: "```javascript\n{\"name\": \"John\"}\n```",
			expected: map[string]interface{}{
				"name": "John",
			},
		},
		{
			name:  "js标记",
			input: "```js\n{\"name\": \"John\"}\n```",
			expected: map[string]interface{}{
				"name": "John",
			},
		},
		{
			name:  "大小写不敏感",
			input: "```JSON\n{\"name\": \"John\"}\n```",
			expected: map[string]interface{}{
				"name": "John",
			},
		},
		{
			name:  "带格式的代码块",
			input: "```json\n{\n  \"name\": \"John\",\n  \"age\": 25\n}\n```",
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "代码块前后有文本",
			input: "前导文本...\n```json\n{\"result\": \"success\"}\n```\n后缀文本...",
			expected: map[string]interface{}{
				"result": "success",
			},
		},
		{
			name:  "代码块内带注释",
			input: "```json\n{\"name\": \"John\", /* comment */ \"age\": 25}\n```",
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
		{
			name:  "代码块内Python常量",
			input: "```json\n{\"name\": \"John\", \"active\": True}\n```",
			expected: map[string]interface{}{
				"name":   "John",
				"active": true,
			},
		},
		{
			name:  "代码块内单引号",
			input: "```json\n{'name': 'John', 'age': 25}\n```",
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Parse(tt.input, nil)
			assert.Nil(t, err, "Should parse without error")
			assert.NotNil(t, data, "Should return data")

			result, ok := data.(map[string]interface{})
			assert.True(t, ok, "Result should be a map")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseMultipleCodeBlocks tests parsing when multiple code blocks are present.
func TestParseMultipleCodeBlocks(t *testing.T) {
	input := "第一个代码块：\n```json\n{\"step\": \"1\", \"status\": \"pending\"}\n```\n\n第二个代码块：\n```json\n{\"step\": \"2\", \"status\": \"completed\"}\n```\n\n最终结果：\n```json\n{\"step\": \"3\", \"status\": \"success\", \"value\": 42}\n```\n"

	data, err := Parse(input, nil)
	assert.Nil(t, err)
	assert.NotNil(t, data)

	// Should return the last code block's content
	result, ok := data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "3", result["step"])
	assert.Equal(t, "success", result["status"])
	assert.Equal(t, float64(42), result["value"])
}

// TestParseMultipleCodeBlocksLastValid tests when last block is valid but earlier ones are invalid.
func TestParseMultipleCodeBlocksLastValid(t *testing.T) {
	input := "第一个无效代码块：\n```json\n{invalid json}\n```\n\n第二个也无效：\n```json\n{also invalid}\n```\n\n最终有效的：\n```json\n{\"valid\": true, \"value\": 42}\n```\n"

	data, err := Parse(input, nil)
	assert.Nil(t, err)
	assert.NotNil(t, data)

	result, ok := data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, result["valid"])
	assert.Equal(t, float64(42), result["value"])
}

// TestParseWithNoValidContent tests parsing with no valid JSON content.
func TestParseWithNoValidContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "纯文本",
			input: "这是一段普通的文本，不包含任何JSON内容。",
		},
		{
			name:  "空字符串",
			input: "",
		},
		{
			name:  "只有空白字符",
			input: "   \n\t  ",
		},
		{
			name:  "HTML代码",
			input: "<div>Some HTML</div>",
		},
		{
			name:  "纯英文段落",
			input: "This is a paragraph of English text without any JSON.",
		},
		{
			name:  "代码块内无有效JSON",
			input: "```\nSome text that's not JSON\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Parse(tt.input, nil)
			assert.NotNil(t, err, "Should return an error")
			assert.Nil(t, data, "Should not return data")
		})
	}
}

// TestParseComplexJSON tests parsing complex JSON structures.
func TestParseComplexJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, data interface{})
	}{
		{
			name: "嵌套对象和数组",
			input: `{
				"users": [
					{"id": 1, "name": "Alice", "tags": ["admin", "active"]},
					{"id": 2, "name": "Bob", "tags": ["user"]}
				],
				"metadata": {
					"total": 2,
					"page": 1
				}
			}`,
			check: func(t *testing.T, data interface{}) {
				result, ok := data.(map[string]interface{})
				assert.True(t, ok)
				assert.Contains(t, result, "users")
				assert.Contains(t, result, "metadata")
			},
		},
		{
			name:  "深层嵌套",
			input: `{"a": {"b": {"c": {"d": "value"}}}}`,
			check: func(t *testing.T, data interface{}) {
				result, ok := data.(map[string]interface{})
				assert.True(t, ok)
				assert.Contains(t, result, "a")
			},
		},
		{
			name:  "空对象",
			input: `{}`,
			check: func(t *testing.T, data interface{}) {
				result, ok := data.(map[string]interface{})
				assert.True(t, ok)
				assert.Empty(t, result)
			},
		},
		{
			name:  "空数组",
			input: `[]`,
			check: func(t *testing.T, data interface{}) {
				result, ok := data.([]interface{})
				assert.True(t, ok)
				assert.Empty(t, result)
			},
		},
		{
			name:  "纯数字",
			input: `42`,
			check: func(t *testing.T, data interface{}) {
				result, ok := data.(float64)
				assert.True(t, ok)
				assert.Equal(t, float64(42), result)
			},
		},
		{
			name:  "纯字符串",
			input: `"hello world"`,
			check: func(t *testing.T, data interface{}) {
				result, ok := data.(string)
				assert.True(t, ok)
				assert.Equal(t, "hello world", result)
			},
		},
		{
			name:  "纯布尔值",
			input: `true`,
			check: func(t *testing.T, data interface{}) {
				result, ok := data.(bool)
				assert.True(t, ok)
				assert.True(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Parse(tt.input, nil)
			assert.Nil(t, err)
			assert.NotNil(t, data)
			tt.check(t, data)
		})
	}

	// Test null separately since it returns nil data
	t.Run("纯null", func(t *testing.T) {
		data, err := Parse(`null`, nil)
		assert.Nil(t, err)
		assert.Nil(t, data)
	})
}

// TestParseWithSchema tests ParseWithSchema with pre-compiled schema.
func TestParseWithSchema(t *testing.T) {
	// Create a pre-compiled schema
	schemaString := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name", "age"]
	}`

	schemaLoader := gojsonschema.NewStringLoader(schemaString)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	assert.Nil(t, err)

	t.Run("有效数据", func(t *testing.T) {
		data, err := ParseWithSchema(`{"name": "John", "age": 25}`, schema)
		assert.Nil(t, err)
		assert.NotNil(t, data)

		result, ok := data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "John", result["name"])
	})

	t.Run("无效数据", func(t *testing.T) {
		data, err := ParseWithSchema(`{"name": "John"}`, schema)
		assert.NotNil(t, err)
		assert.Nil(t, data)
	})

	t.Run("从代码块解析", func(t *testing.T) {
		input := "```json\n{\"name\": \"Alice\", \"age\": 30}\n```"
		data, err := ParseWithSchema(input, schema)
		assert.Nil(t, err)
		assert.NotNil(t, data)
	})

	t.Run("多次解析复用schema", func(t *testing.T) {
		texts := []string{
			`{"name": "Alice", "age": 30}`,
			`{"name": "Bob", "age": 25}`,
			`{"name": "Charlie", "age": 35}`,
		}

		for _, text := range texts {
			data, err := ParseWithSchema(text, schema)
			assert.Nil(t, err)
			assert.NotNil(t, data)
		}
	})
}

// TestInvalidSchema tests error handling for invalid schemas.
func TestInvalidSchema(t *testing.T) {
	// Invalid schema - circular reference which causes schema compilation to fail
	invalidSchema := map[string]interface{}{
		"$ref": "#/invalidRef",
	}

	data, err := Parse(`{"name": "John"}`, invalidSchema)
	assert.NotNil(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "invalid JSON schema")
}

// TestParseBlockEdgeCases tests edge cases for parseBlock.
func TestParseBlockEdgeCases(t *testing.T) {
	t.Run("空代码块", func(t *testing.T) {
		_, err := parseBlock("", nil)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "empty block")
	})

	t.Run("只有空格的代码块", func(t *testing.T) {
		_, err := parseBlock("   ", nil)
		assert.NotNil(t, err)
	})

	t.Run("无法修复的JSON - jsonrepair仍会尝试修复", func(t *testing.T) {
		// jsonrepair will attempt to fix even completely broken JSON
		// It will add quotes and closing braces
		data, err := parseBlock("{completely broken json", nil)
		// jsonrepair repairs this to {"completely broken json":null}
		assert.Nil(t, err)
		assert.NotNil(t, data)
	})
}

// BenchmarkParse benchmarks the Parse function.
func BenchmarkParse(b *testing.B) {
	inputs := []struct {
		name string
		text string
	}{
		{
			name: "SimpleJSON",
			text: `{"name": "John", "age": 25}`,
		},
		{
			name: "JSONWithRepair",
			text: `{name: 'John', age: 25}`,
		},
		{
			name: "MarkdownCodeBlock",
			text: "```json\n{\"name\": \"John\", \"age\": 25}\n```",
		},
		{
			name: "ComplexJSON",
			text: `{
				"users": [
					{"id": 1, "name": "Alice", "tags": ["admin", "active"]},
					{"id": 2, "name": "Bob", "tags": ["user"]}
				],
				"metadata": {
					"total": 2,
					"page": 1
				}
			}`,
		},
	}

	for _, bb := range inputs {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				Parse(bb.text, nil)
			}
		})
	}
}

// BenchmarkParseWithSchema benchmarks Parse with schema validation.
func BenchmarkParseWithSchema(b *testing.B) {
	input := `{"name": "John", "age": 25, "active": true}`

	schemaString := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"active": {"type": "boolean"}
		},
		"required": ["name", "age", "active"]
	}`

	var schema map[string]interface{}
	json.Unmarshal([]byte(schemaString), &schema)

	b.Run("WithSchema", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Parse(input, schema)
		}
	})

	b.Run("WithPrecompiledSchema", func(b *testing.B) {
		schemaLoader := gojsonschema.NewGoLoader(schema)
		compiledSchema, _ := gojsonschema.NewSchema(schemaLoader)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ParseWithSchema(input, compiledSchema)
		}
	})
}

// TestTableDrivenParsing is a comprehensive table-driven test.
func TestTableDrivenParsing(t *testing.T) {
	// Helper schema for validation tests
	var personSchema map[string]interface{}
	json.Unmarshal([]byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name", "age"]
	}`), &personSchema)

	tests := []struct {
		name        string
		input       string
		schema      map[string]interface{}
		wantErr     bool
		wantData    interface{}
		errContains string
	}{
		{
			name:     "标准JSON",
			input:    `{"name": "John", "age": 25}`,
			wantErr:  false,
			wantData: map[string]interface{}{"name": "John", "age": float64(25)},
		},
		{
			name:     "标准JSON通过schema校验",
			input:    `{"name": "John", "age": 25}`,
			schema:   personSchema,
			wantErr:  false,
			wantData: map[string]interface{}{"name": "John", "age": float64(25)},
		},
		{
			name:        "Schema校验失败-缺少字段",
			input:       `{"name": "John"}`,
			schema:      personSchema,
			wantErr:     true,
			errContains: "age",
		},
		{
			name:     "修复单引号",
			input:    `{'name': 'John', 'age': 25}`,
			wantErr:  false,
			wantData: map[string]interface{}{"name": "John", "age": float64(25)},
		},
		{
			name:     "修复缺失引号",
			input:    `{name: "John", age: 25}`,
			wantErr:  false,
			wantData: map[string]interface{}{"name": "John", "age": float64(25)},
		},
		{
			name:     "从代码块提取",
			input:    "```json\n{\"name\": \"John\"}\n```",
			wantErr:  false,
			wantData: map[string]interface{}{"name": "John"},
		},
		{
			name:        "无效JSON无法修复",
			input:       `{completely invalid}`,
			wantErr:     true,
			errContains: "parse attempts failed",
		},
		{
			name:        "无JSON内容",
			input:       "纯文本内容",
			wantErr:     true,
			errContains: "no JSON-like",
		},
		{
			name:     "Python常量转换",
			input:    `{"active": True, "value": None}`,
			wantErr:  false,
			wantData: map[string]interface{}{"active": true, "value": nil},
		},
		{
			name:     "带注释的JSON",
			input:    `{"name": "John", /* comment */ "age": 25}`,
			wantErr:  false,
			wantData: map[string]interface{}{"name": "John", "age": float64(25)},
		},
		{
			name:     "尾随逗号",
			input:    `{"name": "John", "age": 25,}`,
			wantErr:  false,
			wantData: map[string]interface{}{"name": "John", "age": float64(25)},
		},
		{
			name:     "截断JSON",
			input:    `{"name": "John", "age": 25`,
			wantErr:  false,
			wantData: map[string]interface{}{"name": "John", "age": float64(25)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Parse(tt.input, tt.schema)

			if tt.wantErr {
				assert.NotNil(t, err)
				if tt.errContains != "" {
					assert.True(t, strings.Contains(err.Error(), tt.errContains),
						"Error should contain %q, got: %v", tt.errContains, err)
				}
			} else {
				assert.Nil(t, err)
				if tt.wantData != nil {
					assert.Equal(t, tt.wantData, data)
				}
			}
		})
	}
}
