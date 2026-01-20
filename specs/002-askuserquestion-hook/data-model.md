# 数据模型：Supervisor Hook 支持 AskUserQuestion 工具调用审查

**功能**: 002-askuserquestion-hook
**创建日期**: 2026-01-20
**状态**: 完成

## 核心数据结构

### 1. HookInput - Hook 输入结构

统一的 hook 输入结构，支持所有 Claude Code hook 事件类型。

```go
// HookInput 表示从 Claude Code hook 接收的输入
type HookInput struct {
    // 通用字段（所有事件类型共有）
    SessionID      string `json:"session_id"`       // 会话 ID，必需
    TranscriptPath string `json:"transcript_path,omitempty"`
    CWD            string `json:"cwd,omitempty"`
    PermissionMode string `json:"permission_mode,omitempty"`
    HookEventName  string `json:"hook_event_name,omitempty"` // "Stop", "PreToolUse", etc.

    // Stop 事件字段
    StopHookActive bool `json:"stop_hook_active,omitempty"`

    // PreToolUse 事件字段
    ToolName  string          `json:"tool_name,omitempty"`   // 例如: "AskUserQuestion"
    ToolInput json.RawMessage `json:"tool_input,omitempty"`  // 工具特定输入
    ToolUseID string          `json:"tool_use_id,omitempty"` // 工具调用 ID
}
```

**验证规则**:
- `SessionID` 为必需字段
- `HookEventName` 用于区分事件类型，不存在时默认为 "Stop"
- `ToolName` 仅在 `HookEventName == "PreToolUse"` 时使用

### 2. HookOutput - Hook 输出结构

统一的 hook 输出结构，根据事件类型返回不同格式。

```go
// HookOutput 表示返回给 Claude Code hook 的输出
type HookOutput struct {
    // Stop 事件使用
    Decision *string `json:"decision,omitempty"` // "block" 或省略（省略表示允许停止）
    Reason   string  `json:"reason,omitempty"`  // 反馈信息

    // PreToolUse 事件使用
    HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}
```

**输出格式规则**:

| 事件类型 | AllowStop=true | AllowStop=false |
|----------|----------------|-----------------|
| **Stop** | `{"reason": "..."}` | `{"decision": "block", "reason": "..."}` |
| **PreToolUse** | `{"hookSpecificOutput": {...}}` with `permissionDecision: "allow"` | `{"hookSpecificOutput": {...}}` with `permissionDecision: "deny"` |

### 3. HookSpecificOutput - PreToolUse 特定输出

```go
// HookSpecificOutput 表示 PreToolUse hook 的特定输出
type HookSpecificOutput struct {
    HookEventName            string `json:"hookEventName"`                       // "PreToolUse"
    PermissionDecision       string `json:"permissionDecision"`                  // "allow", "deny", "ask"
    PermissionDecisionReason string `json:"permissionDecisionReason"`            // 决策原因
}
```

**决策值说明**:
- `allow`: 允许工具调用执行（跳过权限确认）
- `deny`: 阻止工具调用执行
- `ask`: 要求用户确认（正常权限流程）

### 4. SupervisorResult - Supervisor 审查结果（内部使用）

```go
// SupervisorResult 表示从 Supervisor SDK 解析出的审查结果
type SupervisorResult struct {
    AllowStop bool   `json:"allow_stop"` // 是否允许操作（true=允许，false=阻止）
    Feedback  string `json:"feedback"`   // 反馈信息
}
```

**转换逻辑**:

```go
// SupervisorResultToHookOutput 将内部审查结果转换为 hook 输出
func SupervisorResultToHookOutput(result *SupervisorResult, eventType string) *HookOutput {
    if eventType == "PreToolUse" {
        decision := "allow"
        if !result.AllowStop {
            decision = "deny"
        }
        return &HookOutput{
            HookSpecificOutput: &HookSpecificOutput{
                HookEventName:            "PreToolUse",
                PermissionDecision:       decision,
                PermissionDecisionReason: result.Feedback,
            },
        }
    }

    // Stop 事件（默认）
    if !result.AllowStop {
        decision := "block"
        return &HookOutput{
            Decision: &decision,
            Reason:   result.Feedback,
        }
    }

    // 允许停止
    return &HookOutput{
        Reason: result.Feedback,
    }
}
```

### 5. 向后兼容类型

```go
// StopHookInput 保持向后兼容，是 HookInput 的别名
type StopHookInput = HookInput
```

## 配置数据结构

### Claude Code Hooks 配置

在 `settings.json` 中的 hooks 配置结构：

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/ccc supervisor-hook",
            "timeout": 600
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "AskUserQuestion",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/ccc supervisor-hook",
            "timeout": 600
          }
        ]
      }
    ]
  }
}
```

**配置说明**:
- `matcher`: 使用 "AskUserQuestion" 精确匹配该工具
- `timeout`: 600 秒超时，与 Stop hook 一致
- `command`: 复用相同的 `ccc supervisor-hook` 命令

## 状态数据结构

### Supervisor 状态（无变化）

现有的 supervisor 状态结构保持不变：

```go
// State 表示 supervisor 的持久化状态
type State struct {
    SessionID string    `json:"session_id"` // 会话 ID
    Enabled   bool      `json:"enabled"`    // 是否启用 supervisor 模式
    Count     int       `json:"count"`      // 迭代计数
    CreatedAt time.Time `json:"created_at"` // 创建时间
    UpdatedAt time.Time `json:"updated_at"` // 更新时间
}
```

**变更**: 迭代计数 `Count` 现在会在所有 hook 事件类型（Stop 和 PreToolUse）中递增。

## 数据流图

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Claude Code                                  │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              │ 准备调用 AskUserQuestion
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    PreToolUse Hook 触发                              │
│  输入: {session_id, hook_event_name: "PreToolUse",                   │
│         tool_name: "AskUserQuestion", tool_input: {...}}             │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              │ ccc supervisor-hook
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    RunSupervisorHook()                               │
│  1. 解析输入 → HookInput                                            │
│  2. 检查 CCC_SUPERVISOR_HOOK 防止递归                                │
│  3. 加载 State，检查是否启用                                         │
│  4. 检查迭代计数限制                                                 │
│  5. 增加迭代计数                                                     │
│  6. 调用 Supervisor SDK 审查                                         │
│  7. 解析结果 → SupervisorResult                                     │
│  8. 转换输出 → HookOutput                                           │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              │ 输出: {hookSpecificOutput: {permissionDecision: "deny", ...}}
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Claude Code                                  │
│  根据决策: deny → 取消 AskUserQuestion 调用                          │
│             allow → 正常执行 AskUserQuestion 调用                     │
└─────────────────────────────────────────────────────────────────────┘
```

## 错误处理

### 输入解析错误

```go
// 解析失败时的行为
if err := json.Unmarshal(stdinData, &input); err != nil {
    // 返回 exit code 2，stderr 包含错误信息
    return fmt.Errorf("failed to parse hook input: %w", err)
}
```

### 输出生成错误

```go
// 如果输出 JSON 生成失败，返回错误
outputJSON, err := json.Marshal(output)
if err != nil {
    return fmt.Errorf("failed to marshal hook output: %w", err)
}
```

### SDK 调用失败

```go
// 如果 Supervisor SDK 调用失败，使用 fallback 策略
if err := runSupervisorWithSDK(...); err != nil {
    // 记录错误日志
    log.Error("supervisor SDK failed", "error", err.Error())
    // 返回 deny 决策（安全失败）
    return &HookOutput{
        HookSpecificOutput: &HookSpecificOutput{
            HookEventName:            "PreToolUse",
            PermissionDecision:       "deny",
            PermissionDecisionReason: "Supervisor 审查失败，已阻止操作",
        },
    }
}
```

## 测试数据示例

### Stop 事件

**输入**:
```json
{
  "session_id": "abc123",
  "stop_hook_active": false
}
```

**输出（阻止停止）**:
```json
{
  "decision": "block",
  "reason": "工作尚未完成，需要继续测试"
}
```

**输出（允许停止）**:
```json
{
  "reason": "工作已完成"
}
```

### PreToolUse 事件（AskUserQuestion）

**输入**:
```json
{
  "session_id": "abc123",
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {
    "questions": [
      {
        "question": "请选择实现方案",
        "header": "方案选择",
        "options": [...]
      }
    ]
  },
  "tool_use_id": "toolu_01ABC123..."
}
```

**输出（允许）**:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "permissionDecisionReason": "问题合理，可以向用户提问"
  }
}
```

**输出（阻止）**:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "应该在代码中添加更多注释后再提问"
  }
}
```
