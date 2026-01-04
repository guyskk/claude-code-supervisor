# Design: add-supervisor-hooks-mode

## Technical Approach

### 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│ 用户执行: CCC_SUPERVISOR=1 ccc kimi                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. provider.SwitchWithHook(providerName)                        │
│    - 生成 settings.json (包含 Stop hook)                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. syscall.Exec("claude") (无 --settings 参数)                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Claude Code 工作循环                                             │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ 用户输入 → Agent 执行 → 完成 → 触发 Stop hook               │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                              │                                   │
│                              ▼                                   │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ ccc supervisor-hook                                         │ │
│ │ - 检查 CCC_SUPERVISOR_HOOK=1? 否，继续                       │ │
│ │ - 读取 stdin JSON (session_id, stop_hook_active)            │ │
│ │ - 检查迭代次数 (>=10 则返回空)                               │ │
│ │ - 调用 claude --print --resume <session_id>                 │ │
│ │   (设置 CCC_SUPERVISOR_HOOK=1 环境变量)                      │ │
│ │ - 解析结构化输出                                             │ │
│ │ - 输出 JSON: {"decision":"block","reason":"反馈"}           │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                              │                                   │
│                    ┌─────────┴─────────┐                        │
│                    ▼                   ▼                        │
│              completed=true       completed=false               │
│                    │                   │                        │
│                    ▼                   ▼                        │
│              返回空 (停止)      返回 {"decision":"block"}       │
│                                      │                          │
│                                      └──→ Agent 继续工作        │
└─────────────────────────────────────────────────────────────────┘
```

### Settings 文件结构

**settings.json** (唯一的配置文件)：
```json
{
  "permissions": {...},
  "env": {...},
  "disableAllHooks": false,
  "allowManagedHooksOnly": false,
  "hooks": {
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "/abs/path/ccc supervisor-hook --state-dir .claude/ccc"
      }]
    }]
  }
}
```

### 防止 Hook 死循环

使用环境变量 `CCC_SUPERVISOR_HOOK=1` 来区分 Agent 和 Supervisor 的 hook 调用：

```
Agent claude (无 CCC_SUPERVISOR_HOOK)
    └─> stop 触发 hook
        └─> ccc supervisor-hook (检测：无 CCC_SUPERVISOR_HOOK，继续)
            └─> 启动 Supervisor claude (设置 CCC_SUPERVISOR_HOOK=1)
                └─> stop 触发 hook
                    └─> ccc supervisor-hook (检测：有 CCC_SUPERVISOR_HOOK=1，跳过)
                        └─> 直接返回，允许 stop
```

### supervisor-hook 子命令

**参数**：
- `--state-dir`: 状态文件目录（默认 `.claude/ccc`）

**输入 (stdin)**：
```json
{
  "session_id": "abc123",
  "stop_hook_active": true,
  ...
}
```

**输出 (stdout)**：
- 任务完成：空（什么都不输出）
- 需要继续：`{"decision":"block","reason":"反馈内容"}`

### 状态文件结构

**.claude/ccc/supervisor-{session_id}.json**：
```json
{
  "session_id": "abc123",
  "count": 3,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

### 输出文件结构

**.claude/ccc/supervisor-{session_id}-output.jsonl**：
```jsonl
{"type":"text","content":"..."}
{"type":"text","content":"..."}
{"type":"result","result":{"completed":false,"feedback":"需要补充..."}}
```

## Claude Command 构建

### Supervisor 调用命令

```bash
claude \
  --print \
  --resume <session_id> \
  --verbose \
  --output-format stream-json \
  --json-schema '{"type":"object","properties":{"completed":{"type":"boolean"},"feedback":{"type":"string"}},"required":["completed","feedback"]}' \
  --system-prompt "$(cat ~/.claude/SUPERVISOR.md)"
```

**环境变量**：
```bash
CCC_SUPERVISOR_HOOK=1
```

**关键参数**：
- `--print`: 非交互模式，获取输出后退出
- `--resume <session_id>`: 恢复指定的 session
- `--output-format stream-json`: 输出流式 JSON（便于解析）
- `--json-schema`: 强制结构化输出
- `--system-prompt`: 设置 Supervisor 提示词

## Stream-JSON 处理

```go
// 解析流式输出
for _, line := range lines {
    msg := ParseStreamJSONLine(line)
    // 原始内容输出到 stderr
    if msg.Type == "text" {
        fmt.Fprintf(os.Stderr, "%s\n", msg.Content)
    }
    // 保存到输出文件
    writeLineToFile(outputFile, line)
    // 提取 result 中的结构化数据
    if msg.Type == "result" {
        var result struct {
            Completed bool   `json:"completed"`
            Feedback  string `json:"feedback"`
        }
        json.Unmarshal([]byte(msg.Result), &result)
    }
}
```

## 防止无限循环

### 环境变量检查

```go
// hook.go 开头
if os.Getenv("CCC_SUPERVISOR_HOOK") == "1" {
    fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] Skipping (CCC_SUPERVISOR_HOOK=1), allowing stop\n")
    return nil
}
```

### 迭代次数限制

```go
// 检查迭代次数
state := loadState(sessionID)
if state.Count >= 10 {
    // 返回空，允许停止
    return nil
}
state.Count++
saveState(sessionID, state)
```

## Supervisor Prompt

从 `~/.claude/SUPERVISOR.md` 读取，包含：

```markdown
## 输出格式要求

你必须严格按照以下 JSON Schema 返回结果：

```json
{
  "type": "object",
  "properties": {
    "completed": {
      "type": "boolean",
      "description": "任务是否已完成"
    },
    "feedback": {
      "type": "string",
      "description": "当 completed 为 false 时，提供具体的反馈和改进建议"
    }
  },
  "required": ["completed", "feedback"]
}
```

- 如果任务完成，设置 `"completed": true`，`feedback` 可以为空
- 如果任务未完成，设置 `"completed": false`，`feedback` 必须包含具体的反馈
```

## Trade-offs

### 优点
1. **简化实现**：不需要自己管理 Agent 循环
2. **更好的集成**：利用 Claude 原生 Hook 机制
3. **用户体验更好**：直接与 Claude 交互，无中转
4. **单一配置文件**：只需要 settings.json，简化管理
5. **环境变量控制**：简洁的死循环防护机制

### 缺点
1. **依赖 Claude Hooks**：如果 Hooks 机制变化，需要适配
2. **路径问题**：需要确保 hook 命令中的绝对路径正确

### 决策
优点明显大于缺点。Hooks 是 Claude 官方机制，相对稳定。
环境变量控制比双配置文件更简洁。

## Testing Considerations

1. **单元测试**：
   - `supervisor-hook` 子命令的输入输出处理
   - 环境变量检测和跳过逻辑
   - 状态文件的读写
   - stream-json 的解析

2. **集成测试**：
   - 完整的 supervisor mode 流程
   - 迭代次数限制
   - 环境变量传递

3. **Mock 策略**：
   - claude 命令执行可以 mock
   - 文件系统操作可以 mock
   - 环境变量可以在测试中设置
