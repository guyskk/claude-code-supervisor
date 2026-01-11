# LLM JSON Parser Go实现方案

## 一、项目背景与目标

### 1.1 背景

在项目中，我们需要容错解析大语言模型（LLM）输出的JSON数据。Python版本的 `llm_json_parser` 已经证明有效，现在需要在Go项目中实现相同的功能。

**原始Python实现**: `/home/ubuntu/dev/recurve-server1/recurve/utils/llm_json_parser.py`

### 1.2 核心目标

1. **容错解析**: 能够解析LLM输出的各种格式的JSON，包括：
   - 标准JSON
   - Markdown代码块包裹的JSON
   - 带注释的JSON
   - 带尾随逗号的JSON
   - 使用单引号的JSON
   - 截断的JSON
   - 嵌套在其他文本中的JSON

2. **数据校验**: 支持通过JSON Schema校验解析后的数据结构

3. **高性能**: 利用Go的性能优势，实现高效的解析和校验

---

## 二、依赖库分析

### 2.1 jsonrepair 库

**仓库**: [kaptinlin/jsonrepair](https://github.com/kaptinlin/jsonrepair)

**核心API**:
```go
func JSONRepair(text string) (string, error)
```

**功能特性**:

| 功能 | 描述 |
|------|------|
| 添加缺失引号 | 自动为未加引号的键添加双引号 |
| 添加缺失转义字符 | 自动添加必要的转义字符 |
| 添加缺失逗号 | 在元素之间插入缺失的逗号 |
| 添加缺失闭合括号 | 补全未闭合的对象/数组 |
| 修复截断JSON | 完成被截断的JSON数据 |
| 替换单引号 | 将单引号转换为双引号 |
| 替换特殊引号 | 将 `""` 等转换为标准双引号 |
| 移除注释 | 移除 `/* ... */` 和 `// ...` 注释 |
| 移除代码块标记 | 移除 markdown ` ```json` 标记 |
| 转换Python常量 | `None` → `null`, `True` → `true`, `False` → `false` |
| 移除尾随逗号 | 清理数组/对象中的尾随逗号 |
| 移除省略号 | 移除数组中的 `[1, 2, ...]` 省略号 |
| 移除JSONP | 移除JSONP回调包装 |
| 转换换行符分隔JSON | 将NDJSON包裹为数组 |

**优势**:
- 专门为LLM输出设计，针对性强
- 高性能Go实现
- 与JavaScript版本保持一致的逻辑

**示例**:
```go
package main

import (
    "fmt"
    "github.com/kaptinlin/jsonrepair"
)

func main() {
    // 无效JSON：缺失引号、单引号
    invalid := `{name: 'John', age: 25,}`
    repaired, err := jsonrepair.JSONRepair(invalid)
    if err != nil {
        panic(err)
    }
    fmt.Println(repaired)
    // 输出: {"name": "John", "age": 25}
}
```

### 2.2 gojsonschema 库

**仓库**: [xeipuuv/gojsonschema](https://github.com/xeipuuv/gojsonschema)

**支持的版本**: JSON Schema draft-04, draft-06, draft-07

**核心API**:

```go
// 方式1: 直接验证
result, err := gojsonschema.Validate(schemaLoader, documentLoader)

// 方式2: 预编译schema（推荐用于重复验证）
schema, err := gojsonschema.NewSchema(schemaLoader)
result, err := schema.Validate(documentLoader)
```

**Loader类型**:

| Loader | 用途 | 示例 |
|--------|------|------|
| `NewReferenceLoader` | 从文件或HTTP加载 | `file:///path/to/schema.json` |
| `NewStringLoader` | 从JSON字符串加载 | `{"type": "string"}` |
| `NewGoLoader` | 从Go类型加载 | `map[string]interface{}` |

**验证结果**:
```go
type Result interface {
    Valid() bool
    Errors() []ResultError
}

type ResultError interface {
    Type() string
    Field() string
    Value() interface{}
    Context() JsonContext
    Description() string
    Details() ErrorDetails
}
```

**支持的格式**:

| 格式 | 描述 |
|------|------|
| `date` | 日期格式 |
| `time` | 时间格式 |
| `date-time` | 日期时间格式 |
| `email` | 邮箱格式 |
| `uri` | URI格式 |
| `uuid` | UUID格式 |
| `hostname` | 主机名格式 |
| `ipv4` | IPv4地址 |
| `ipv6` | IPv6地址 |
| `regex` | 正则表达式 |

**示例**:
```go
package main

import (
    "fmt"
    "github.com/xeipuuv/gojsonschema"
)

func main() {
    // 定义schema
    schemaString := `{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "name": {"type": "string"},
            "age": {"type": "integer", "minimum": 0}
        },
        "required": ["name", "age"]
    }`

    // 要验证的JSON
    documentString := `{"name": "John", "age": 25}`

    schemaLoader := gojsonschema.NewStringLoader(schemaString)
    documentLoader := gojsonschema.NewStringLoader(documentString)

    result, err := gojsonschema.Validate(schemaLoader, documentLoader)
    if err != nil {
        panic(err)
    }

    if result.Valid() {
        fmt.Println("文档有效")
    } else {
        fmt.Println("验证失败:")
        for _, desc := range result.Errors() {
            fmt.Printf("- %s\n", desc)
        }
    }
}
```

---

## 三、原始Python代码的容错策略分析

### 3.1 正则表达式提取代码块

```python
_RE_MODE = re.MULTILINE | re.ASCII | re.IGNORECASE | re.DOTALL
RE_CODE_BLOCK = re.compile(r"```(json|jsonl|jsonlines|json5|jsonc|js|javascript)?\s*\n(.+?)\s*```", _RE_MODE)
```

**特点**:
- 使用非贪婪匹配 `.+?`
- 支持多种语言标记（可选）
- 使用 `(?i)` 忽略大小写

**Go对应实现**:
```go
var codeBlockRE = regexp.MustCompile(`(?i)\x60\x60\x60(json|jsonl|jsonlines|json5|jsonc|js|javascript)?\s*\n(.+?)\s*\x60\x60\x60`)
```

### 3.2 JSON似然判断

```python
def _is_likely_json(self, text: str):
    text = text.strip()
    if not text:
        return False
    if text.startswith(("{", "[", '"')):
        return True
    if text.endswith(("}", "]", '"')):
        return True
    if text in ("true", "false", "null"):
        return True
    return re.match(r"^-?\d+", text)
```

**判断逻辑**:
1. 检查开头是否为对象、数组或字符串起始符
2. 检查结尾是否为对象、数组或字符串结束符
3. 检查是否为布尔值或null
4. 检查是否为数字

### 3.3 多层级容错解析流程

```
┌─────────────────────────────────────────────────────────┐
│  1. 提取Markdown代码块                                   │
│     - 正则匹配 ```json...```                             │
│     - 如果无代码块，检查全文是否像JSON                     │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│  2. 反向遍历代码块（最后一个最可能是期望输出）              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│  3. 尝试标准JSON解析                                      │
│     json.loads(text)                                     │
│          ↓ 失败                                          │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│  4. JSON修复                                             │
│     repair_json(text, return_objects=True)              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│  5. 数据模型验证（如果提供了data_model）                    │
│     data_model.model_validate(result)                    │
└─────────────────────────────────────────────────────────┘
                          ↓
                    返回结果
```

> **注意**: Go版本不包含Python原版中的"提取嵌套的Result字段"步骤，该步骤已被移除。

### 3.4 错误收集策略

```python
code_block_s = list(self._extract_markdown_code_block(text))
error_s = []
for text in reversed(code_block_s):
    # ... 尝试解析
    if data_model is not None:
        try:
            result = data_model.model_validate(result)
        except ValidationError as ex:
            error_s.append(ex)
            continue
    return None, result
if error_s:
    return error_s[0], None
return None, None
```

**策略**:
- 收集所有解析/验证错误
- 返回第一个错误
- 如果所有代码块都失败，返回 `None, None`

---

## 四、Go版本设计

### 4.1 设计原则

采用**无状态包级函数**设计：
- 无需实例化，直接调用包级函数
- 正则表达式在包初始化时编译一次
- 无缓存逻辑，保持简洁

### 4.2 API设计

```go
package llmparser

import (
    "regexp"
    "github.com/xeipuuv/gojsonschema"
)

// 包级变量：预编译正则表达式
var codeBlockRE = regexp.MustCompile(`(?i)\x60\x60\x60(json|jsonl|jsonlines|json5|jsonc|js|javascript)?\s*\n(.+?)\s*\x60\x60\x60`)

// Parse 解析LLM输出的JSON
//
// 参数:
//   - text: LLM输出的文本
//   - jsonSchema: JSON Schema（可选，为nil时不校验数据结构）
//
// 返回:
//   - interface{}: 解析后的数据
//   - error: 错误信息，nil表示成功
func Parse(text string, jsonSchema map[string]interface{}) (interface{}, error)

// ParseWithSchema 使用预编译的schema解析（性能优化版本）
//
// 参数:
//   - text: LLM输出的文本
//   - schema: 预编译的JSON Schema
//
// 返回:
//   - interface{}: 解析后的数据
//   - error: 错误信息，nil表示成功
func ParseWithSchema(text string, schema *gojsonschema.Schema) (interface{}, error)
```

**API使用示例**:

```go
package main

import (
    "encoding/json"
    "fmt"
    llmparser "yourmodule/internal/llmparser"
)

func main() {
    // 示例1: 简单解析（无schema校验）
    data, err := llmparser.Parse(`{"name": "John", "age": 25}`, nil)
    if err != nil {
        panic(err)
    }
    fmt.Printf("解析结果: %+v\n", data)

    // 示例2: 带schema校验
    var schema map[string]interface{}
    json.Unmarshal([]byte(`{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "name": {"type": "string"},
            "age": {"type": "integer", "minimum": 0}
        },
        "required": ["name", "age"]
    }`), &schema)

    data, err = llmparser.Parse(`{"name": "John", "age": 25}`, schema)
    if err != nil {
        panic(err)
    }
    fmt.Printf("校验通过: %+v\n", data)

    // 示例3: 从markdown代码块中解析
    input := `这是分析过程...

\`\`\`json
{"result": "success", "value": 42}
\`\`\`

分析完毕。`
    data, err = llmparser.Parse(input, nil)
    if err != nil {
        panic(err)
    }
    fmt.Printf("从代码块解析: %+v\n", data)

    // 示例4: 使用预编译schema（高性能场景）
    schemaLoader := gojsonschema.NewStringLoader(`{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "name": {"type": "string"}
        },
        "required": ["name"]
    }`)
    compiledSchema, _ := gojsonschema.NewSchema(schemaLoader)

    // 多次解析，复用schema
    for _, text := range []string{`{"name": "Alice"}`, `{"name": "Bob"}`} {
        data, err := llmparser.ParseWithSchema(text, compiledSchema)
        // ...
    }
}
```

### 4.3 完整解析流程

```
                    输入: text, jsonSchema?
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  1. 提取Markdown代码块                                       │
│     - 使用正则表达式匹配 ```json...```                       │
│     - 支持多种语言标记（可选）                                │
│     - 如果无代码块，使用全文                                  │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  2. JSON似然判断（仅当无代码块时）                            │
│     - 检查开头/结尾是否为JSON起始/结束符                      │
│     - 检查是否为基本值                                        │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  3. 反向遍历代码块                                           │
│     - 最后一个代码块最可能是期望输出                           │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  4. 尝试标准JSON解析                                         │
│     encoding/json.Unmarshal()                               │
│          ↓ 失败                                              │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  5. JSON修复                                                 │
│     jsonrepair.JSONRepair()                                 │
│          ↓ 失败                                              │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  6. JSON Schema校验（如果提供了schema）                       │
│     gojsonschema.Validate()                                 │
└─────────────────────────────────────────────────────────────┘
                           ↓
              返回 (interface{}, error)
```

---

## 五、实现细节

### 5.1 代码块提取

```go
// extractMarkdownCodeBlocks 提取markdown代码块
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
        if len(match) > 2 {
            blocks = append(blocks, strings.TrimSpace(match[2]))
        }
    }
    return blocks
}
```

**正则表达式说明**:
```
(?i)```(json|jsonl|jsonlines|json5|jsonc|js|javascript)?\s*\n(.+?)\s*```
```
- `(?i)` - 忽略大小写
- ``` - 匹配三个反引号
- `(json|...)` - 语言标记（可选）
- `\s*\n` - 可选空白后跟换行
- `(.+?)` - 代码块内容（非贪婪）
- `\s*``` - 可选空白后跟三个反引号

### 5.2 JSON似然判断

```go
// isLikelyJSON 判断文本是否像JSON
func isLikelyJSON(text string) bool {
    text = strings.TrimSpace(text)
    if text == "" {
        return false
    }

    // 检查开头
    firstChar := text[0]
    if firstChar == '{' || firstChar == '[' || firstChar == '"' {
        return true
    }

    // 检查结尾
    lastChar := text[len(text)-1]
    if lastChar == '}' || lastChar == ']' || lastChar == '"' {
        return true
    }

    // 检查基本值
    switch text {
    case "true", "false", "null":
        return true
    }

    // 检查数字
    if len(text) > 0 {
        c := text[0]
        if c == '-' || (c >= '0' && c <= '9') {
            return true
        }
    }

    return false
}
```

### 5.3 核心解析方法

```go
// Parse 解析LLM输出的JSON
func Parse(text string, jsonSchema map[string]interface{}) (interface{}, error) {
    // 1. 提取代码块
    blocks := extractMarkdownCodeBlocks(text)
    if len(blocks) == 0 {
        return nil, fmt.Errorf("no JSON-like content found")
    }

    // 2. 编译schema（如果提供）
    var schema *gojsonschema.Schema
    var err error
    if jsonSchema != nil {
        schemaLoader := gojsonschema.NewGoLoader(jsonSchema)
        schema, err = gojsonschema.NewSchema(schemaLoader)
        if err != nil {
            return nil, fmt.Errorf("invalid JSON schema: %w", err)
        }
    }

    // 3. 反向遍历代码块
    var errors []error
    for i := len(blocks) - 1; i >= 0; i-- {
        result, err := parseBlock(blocks[i], schema)
        if err == nil {
            return result, nil
        }
        errors = append(errors, err)
    }

    // 4. 所有尝试都失败，返回最后一个错误
    if len(errors) > 0 {
        return nil, fmt.Errorf("all parse attempts failed: %w", errors[len(errors)-1])
    }

    return nil, fmt.Errorf("no valid JSON found")
}

// parseBlock 解析单个代码块
func parseBlock(block string, schema *gojsonschema.Schema) (interface{}, error) {
    // 1. 尝试标准解析
    var result interface{}
    err := json.Unmarshal([]byte(block), &result)
    if err != nil {
        // 2. 标准解析失败，尝试修复
        repaired, repairErr := jsonrepair.JSONRepair(block)
        if repairErr != nil {
            return nil, fmt.Errorf("JSON repair failed: %w, original error: %w", repairErr, err)
        }
        err = json.Unmarshal([]byte(repaired), &result)
        if err != nil {
            return nil, fmt.Errorf("repaired JSON parsing failed: %w", err)
        }
    }

    // 3. Schema校验（如果提供了schema）
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
            return nil, fmt.Errorf(errMsg.String())
        }
    }

    return result, nil
}
```

### 5.4 性能优化：ParseWithSchema

```go
// ParseWithSchema 使用预编译的schema解析（性能优化版本）
func ParseWithSchema(text string, schema *gojsonschema.Schema) (interface{}, error) {
    blocks := extractMarkdownCodeBlocks(text)
    if len(blocks) == 0 {
        return nil, fmt.Errorf("no JSON-like content found")
    }

    for i := len(blocks) - 1; i >= 0; i-- {
        result, err := parseBlock(blocks[i], schema)
        if err == nil {
            return result, nil
        }
    }

    return nil, fmt.Errorf("no valid JSON found")
}
```

**使用场景**:
```go
// 场景: 需要多次解析相同schema的数据
schemaLoader := gojsonschema.NewStringLoader(mySchemaString)
schema, _ := gojsonschema.NewSchema(schemaLoader)

// 多次解析，复用schema
for _, text := range textList {
    data, err := llmparser.ParseWithSchema(text, schema)
    // ...
}
```

---

## 六、测试策略

### 6.1 测试覆盖范围

| 测试类别 | 测试内容 |
|---------|----------|
| **代码块提取** | 各种格式的markdown代码块、无代码块、多代码块 |
| **JSON似然判断** | 对象、数组、字符串、数字、布尔值、null、非JSON文本 |
| **标准JSON解析** | 标准格式的JSON |
| **JSON修复** | 缺失引号、单引号、尾随逗号、注释、截断JSON |
| **Schema校验** | 有效数据、无效数据、复杂schema |
| **边界情况** | 空字符串、纯文本、多个代码块、超大JSON |
| **性能测试** | 大量解析操作、schema预编译效果 |

### 6.2 测试用例示例

```go
package llmparser_test

import (
    "encoding/json"
    "testing"
    "github.com/stretchr/testify/assert"
    "yourmodule/internal/llmparser"
)

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
            name:     "多代码块",
            input:    "```json\n{\"first\": 1}\n```\n```json\n{\"second\": 2}\n```",
            expected: []string{`{"first": 1}`, `{"second": 2}`},
        },
        {
            name:     "无代码块但像JSON",
            input:    `{"name": "John"}`,
            expected: []string{`{"name": "John"}`},
        },
        {
            name:     "无代码块且不像JSON",
            input:    "这是一段普通文本",
            expected: nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := llmparser.TestExtractMarkdownCodeBlocks(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestParseValidJSON(t *testing.T) {
    data, err := llmparser.Parse(`{"name": "John", "age": 25}`, nil)
    assert.Nil(t, err)
    assert.NotNil(t, data)

    result, ok := data.(map[string]interface{})
    assert.True(t, ok)
    assert.Equal(t, "John", result["name"])
    assert.Equal(t, float64(25), result["age"])
}

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
            name:  "带注释",
            input: `{"name": "John", /* comment */ "age": 25}`,
            expected: map[string]interface{}{
                "name": "John",
                "age":  float64(25),
            },
        },
        {
            name:  "Python常量",
            input: `{"name": "John", "active": True, "value": None}`,
            expected: map[string]interface{}{
                "name":   "John",
                "active": true,
                "value":  nil,
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            data, err := llmparser.Parse(tt.input, nil)
            assert.Nil(t, err)
            assert.NotNil(t, data)

            result, ok := data.(map[string]interface{})
            assert.True(t, ok)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestParseWithSchemaValidation(t *testing.T) {
    var schema map[string]interface{}
    json.Unmarshal([]byte(`{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "name": {"type": "string"},
            "age": {"type": "integer", "minimum": 0}
        },
        "required": ["name", "age"]
    }`), &schema)

    t.Run("有效数据", func(t *testing.T) {
        data, err := llmparser.Parse(`{"name": "John", "age": 25}`, schema)
        assert.Nil(t, err)
        assert.NotNil(t, data)
    })

    t.Run("缺少必需字段", func(t *testing.T) {
        data, err := llmparser.Parse(`{"name": "John"}`, schema)
        assert.NotNil(t, err)
        assert.Nil(t, data)
        assert.Contains(t, err.Error(), "age")
    })

    t.Run("类型错误", func(t *testing.T) {
        data, err := llmparser.Parse(`{"name": "John", "age": "twenty-five"}`, schema)
        assert.NotNil(t, err)
        assert.Nil(t, data)
        assert.Contains(t, err.Error(), "integer")
    })

    t.Run("值超出范围", func(t *testing.T) {
        data, err := llmparser.Parse(`{"name": "John", "age": -5}`, schema)
        assert.NotNil(t, err)
        assert.Nil(t, data)
        assert.Contains(t, err.Error(), "minimum")
    })
}

func TestParseFromMarkdownCodeBlock(t *testing.T) {
    input := `这是一个分析过程...

```json
{"name": "John", "age": 25}
```

分析完毕。`

    data, err := llmparser.Parse(input, nil)
    assert.Nil(t, err)
    assert.NotNil(t, data)

    result, ok := data.(map[string]interface{})
    assert.True(t, ok)
    assert.Equal(t, "John", result["name"])
}

func TestParseMultipleCodeBlocks(t *testing.T) {
    input := `第一个代码块：
```json
{"step": "1", "status": "pending"}
```

第二个代码块：
```json
{"step": "2", "status": "completed"}
```

最终结果：
```json
{"step": "3", "status": "success", "value": 42}
```
`

    data, err := llmparser.Parse(input, nil)
    assert.Nil(t, err)

    // 应该返回最后一个代码块的内容
    result, ok := data.(map[string]interface{})
    assert.True(t, ok)
    assert.Equal(t, "3", result["step"])
    assert.Equal(t, "success", result["status"])
    assert.Equal(t, float64(42), result["value"])
}

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
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            data, err := llmparser.Parse(tt.input, nil)
            assert.NotNil(t, err)
            assert.Nil(t, data)
        })
    }
}

func TestParseComplexJSON(t *testing.T) {
    complexJSON := `{
        "users": [
            {"id": 1, "name": "Alice", "tags": ["admin", "active"]},
            {"id": 2, "name": "Bob", "tags": ["user"]}
        ],
        "metadata": {
            "total": 2,
            "page": 1
        }
    }`

    data, err := llmparser.Parse(complexJSON, nil)
    assert.Nil(t, err)
    assert.NotNil(t, data)
}

// 基准测试
func BenchmarkParse(b *testing.B) {
    input := `{"name": "John", "age": 25, "active": true}`

    var schema map[string]interface{}
    json.Unmarshal([]byte(`{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "name": {"type": "string"},
            "age": {"type": "integer"},
            "active": {"type": "boolean"}
        },
        "required": ["name", "age", "active"]
    }`), &schema)

    b.Run("WithoutSchema", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            llmparser.Parse(input, nil)
        }
    })

    b.Run("WithSchema", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            llmparser.Parse(input, schema)
        }
    })

    b.Run("WithPrecompiledSchema", func(b *testing.B) {
        schemaLoader := gojsonschema.NewGoLoader(schema)
        compiledSchema, _ := gojsonschema.NewSchema(schemaLoader)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            llmparser.ParseWithSchema(input, compiledSchema)
        }
    })
}
```

### 6.3 表驱动测试（推荐用于扩展）

```go
func TestTableDrivenParsing(t *testing.T) {
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
            name:        "无效JSON",
            input:       `{invalid json}`,
            wantErr:     true,
            errContains: "parse attempts failed",
        },
        // ... 更多测试用例
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            data, err := llmparser.Parse(tt.input, tt.schema)

            if tt.wantErr {
                assert.NotNil(t, err)
                if tt.errContains != "" {
                    assert.Contains(t, err.Error(), tt.errContains)
                }
            } else {
                assert.Nil(t, err)
                assert.Equal(t, tt.wantData, data)
            }
        })
    }
}
```

---

## 七、文件结构

```
/home/ubuntu/dev/claude-code-supervisor1/
├── internal/
│   └── llmparser/
│       ├── parser.go       # 核心实现
│       └── parser_test.go  # 测试文件
```

**文件说明**:

| 文件 | 内容 |
|------|------|
| `parser.go` | 核心解析逻辑，包括包级变量和函数 |
| `parser_test.go` | 完整的测试套件 |

---

## 八、依赖管理

### 8.1 go.mod 添加依赖

```bash
go get github.com/kaptinlin/jsonrepair@latest
go get github.com/xeipuuv/gojsonschema@latest
go get github.com/stretchr/testify@latest  # 测试断言库
```

### 8.2 最终 go.mod 依赖

```go
module github.com/guyskk/claude-code-supervisor

go 1.21

require (
    github.com/kaptinlin/jsonrepair v0.x.x
    github.com/xeipuuv/gojsonschema v1.x.x
    github.com/stretchr/testify v1.x.x
)
```

---

## 九、实现检查清单

在开始实现前，确保以下要点都已确认：

- [ ] 已阅读并理解Python原始实现
- [ ] 已熟悉 jsonrepair 库的API和功能
- [ ] 已熟悉 gojsonschema 库的API和用法
- [ ] 已设计好完整的API（包级函数，无状态）
- [ ] 已规划好测试策略
- [ ] 已确定文件结构
- [ ] 已准备好测试用例

---

## 十、总结

本方案详细描述了如何使用Go语言实现一个容错的LLM JSON解析器，主要特点：

1. **容错性强**: 能够处理LLM输出的各种格式问题
2. **简洁设计**: 无状态包级函数，无需实例化
3. **测试完善**: 覆盖各种边界情况和错误场景
4. **Go惯例**: 遵循Go的错误处理习惯，返回 `(interface{}, error)`

**与Python原版的差异**:
- **移除**: 提取嵌套的 `Result` 字段逻辑
- **移除**: schema 缓存逻辑（保持简洁）
- **改进**: 使用Go惯用的 `(T, error)` 返回值和包级函数设计

**核心优势**:
- 使用 `jsonrepair` 自动修复LLM输出格式问题
- 使用 `gojsonschema` 进行灵活的数据校验
- 完整的测试覆盖确保代码质量
- 高性能实现适合生产环境使用
- 遵循Go语言惯用法，易于集成
- 无状态设计，并发安全

**下一步**:
1. 根据此方案实现代码
2. 编写完整的测试
3. 运行测试确保所有场景覆盖
4. 添加性能基准测试
5. 根据测试结果优化性能

---

**参考资料**:
- [kaptinlin/jsonrepair](https://github.com/kaptinlin/jsonrepair)
- [xeipuuv/gojsonschema](https://github.com/xeipuuv/gojsonschema)
- [JSON Schema官方文档](http://json-schema.org/)
- Python原始实现: `/home/ubuntu/dev/recurve-server1/recurve/utils/llm_json_parser.py`
