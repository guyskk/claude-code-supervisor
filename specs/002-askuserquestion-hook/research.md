# 技术研究：Supervisor Hook 支持 AskUserQuestion 工具调用审查

**功能**: 002-askuserquestion-hook
**创建日期**: 2026-01-20
**状态**: 已完成

## 研究目标

本研究为"Supervisor Hook 支持 AskUserQuestion 工具调用审查"功能提供技术决策依据，主要研究：
1. Claude Code PreToolUse hook 的输入输出格式
2. 如何区分不同的 hook 事件类型
3. Go 代码中如何扩展 hook 输入解析和输出格式

## 研究发现

### 1. Claude Code PreToolUse Hook 格式

根据 Claude Code hooks 文档 (`docs/claude-code-hooks.md`)：

**PreToolUse 输入格式**:
```json
{
  "session_id": "abc123",
  "transcript_path": "/path/to/transcript",
  "cwd": "/current/directory",
  "permission_mode": "default",
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {
    "questions": [...]
  },
  "tool_use_id": "toolu_01ABC123..."
}
```

**PreToolUse 输出格式**（用于决策控制）:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",  // 或 "deny", "ask"
    "permissionDecisionReason": "决策原因说明"
  }
}
```

**决策说明**:
- `allow`: 允许工具调用执行（跳过权限确认）
- `deny`: 阻止工具调用执行
- `ask`: 要求用户确认（正常权限流程）

### 2. Stop Hook 现有格式

**Stop 输入格式**（当前实现）:
```json
{
  "session_id": "abc123",
  "stop_hook_active": false
}
```

**Stop 输出格式**（当前实现）:
```json
{
  "decision": "block",  // 阻止停止，继续工作
  "reason": "反馈内容"
}
// 或（允许停止，省略 decision 字段）
{
  "reason": "工作已完成"
}
```

### 3. 事件类型识别策略

**决策**: 通过检测输入 JSON 中的 `hook_event_name` 字段来识别事件类型

| hook_event_name | 输入结构 | 输出结构 |
|-----------------|----------|----------|
| `Stop` | session_id, stop_hook_active | decision, reason |
| `PreToolUse` | session_id, tool_name, hook_event_name, tool_input | hookSpecificOutput.permissionDecision, hookSpecificOutput.permissionDecisionReason |

**理由**:
- `hook_event_name` 是 Claude Code 提供的标准字段
- 不需要维护额外的状态来区分事件类型
- 向后兼容：如果字段不存在，默认为 Stop 事件

### 4. 数据结构扩展

**当前代码结构** (`internal/cli/hook.go`):

```go
// StopHookInput 当前只支持 Stop 事件
type StopHookInput struct {
    SessionID      string `json:"session_id"`
    StopHookActive bool   `json:"stop_hook_active"`
}

// SupervisorResult 当前只返回 Stop 格式
type SupervisorResult struct {
    AllowStop bool   `json:"allow_stop"`
    Feedback  string `json:"feedback"`
}
```

**扩展后的结构**:

```go
// HookInput 支持所有 hook 事件类型
type HookInput struct {
    SessionID      string          `json:"session_id"`
    StopHookActive bool            `json:"stop_hook_active,omitempty"`
    HookEventName  string          `json:"hook_event_name,omitempty"`  // "Stop", "PreToolUse", etc.
    ToolName       string          `json:"tool_name,omitempty"`       // PreToolUse 特有
    ToolInput      json.RawMessage `json:"tool_input,omitempty"`      // PreToolUse 特有
    ToolUseID      string          `json:"tool_use_id,omitempty"`     // PreToolUse 特有
    // 其他通用字段...
    TranscriptPath string `json:"transcript_path,omitempty"`
    CWD            string `json:"cwd,omitempty"`
    PermissionMode string `json:"permission_mode,omitempty"`
}

// HookOutput 根据事件类型返回不同格式
type HookOutput struct {
    // Stop 事件使用
    Decision *string `json:"decision,omitempty"`  // "block" 或省略
    Reason   string  `json:"reason,omitempty"`

    // PreToolUse 事件使用
    HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type HookSpecificOutput struct {
    HookEventName             string `json:"hookEventName"`
    PermissionDecision        string `json:"permissionDecision"`        // "allow", "deny", "ask"
    PermissionDecisionReason  string `json:"permissionDecisionReason"`
}
```

### 5. 向后兼容性策略

**决策**: 保持现有 Stop hook 完全兼容

**实现方式**:
1. 保留 `StopHookInput` 类型作为 `HookInput` 的别名
2. 保留 `SupervisorResult` 类型，添加转换逻辑
3. 输入解析时先尝试新结构，失败后回退到旧结构
4. 输出时根据事件类型选择对应格式

**代码示例**:
```go
// 兼容性：保持旧类型作为别名
type StopHookInput = HookInput

// 输出转换函数
func supervisorResultToHookOutput(result *SupervisorResult, eventType string) *HookOutput {
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
    return &HookOutput{
        Reason: result.Feedback,
    }
}
```

### 6. 迭代计数一致性

**当前实现**: 迭代计数只在 Stop hook 时增加

**变更**: 在所有 hook 事件类型中增加迭代计数

**理由**:
- 防止因审查点增多导致无限循环
- 保持审查逻辑一致性
- 迭代限制是保护机制，不应因事件类型而异

### 7. 配置生成策略

**当前代码** (`internal/provider/provider.go`):

```go
hooks := map[string]interface{}{
    "Stop": []map[string]interface{}{
        {
            "hooks": []map[string]interface{}{
                {
                    "type":    "command",
                    "command": hookCommand,
                    "timeout": 600,
                },
            },
        },
    },
}
```

**扩展后**:

```go
hooks := map[string]interface{}{
    "Stop": []map[string]interface{}{
        {
            "hooks": []map[string]interface{}{
                {
                    "type":    "command",
                    "command": hookCommand,
                    "timeout": 600,
                },
            },
        },
    },
    "PreToolUse": []map[string]interface{}{
        {
            "matcher": "AskUserQuestion",  // 只匹配 AskUserQuestion 工具
            "hooks": []map[string]interface{}{
                {
                    "type":    "command",
                    "command": hookCommand,  // 复用相同的命令
                    "timeout": 600,
                },
            },
        },
    },
}
```

**决策**: 使用 `matcher: "AskUserQuestion"` 只匹配该工具

**理由**:
- 不是所有工具调用都需要 supervisor 审查
- AskUserQuestion 是关键交互点，符合审查目标
- 可扩展：如果将来需要审查其他工具，可以添加更多 matcher

## 技术决策总结

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 事件类型识别 | 通过 `hook_event_name` 字段 | Claude Code 标准字段，向后兼容 |
| 输入结构 | 扩展 `HookInput` 支持所有字段 | 统一解析，减少代码重复 |
| 输出结构 | 根据 `hook_event_name` 返回不同格式 | 符合 Claude Code hook 规范 |
| 向后兼容 | 保留旧类型，添加转换函数 | 不破坏现有功能 |
| PreToolUse 匹配 | 使用 `matcher: "AskUserQuestion"` | 只审查关键工具调用 |
| 迭代计数 | 所有事件类型都增加计数 | 防止无限循环 |

## 未考虑的方案及原因

| 方案 | 被拒绝的原因 |
|------|-------------|
| 为每个 hook 事件类型创建独立命令 | 代码重复，维护成本高 |
| 使用环境变量传递事件类型 | 不符合 Claude Code hook 规范 |
| 只审查 Stop 事件 | 无法满足用户需求，审查不全面 |
| 审查所有 PreToolUse 事件 | 性能影响大，大多数工具调用不需要审查 |

## 参考资料

- Claude Code Hooks 文档: `docs/claude-code-hooks.md`
- 现有 hook 实现: `internal/cli/hook.go`
- Provider 配置生成: `internal/provider/provider.go`
- 项目宪章: `.specify/memory/constitution.md`
