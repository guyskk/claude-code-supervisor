# Hook 输入输出契约

**功能**: 002-askuserquestion-hook
**版本**: 1.0.0
**创建日期**: 2026-01-20

## 概述

本文档定义了 `ccc supervisor-hook` 命令的输入输出契约，支持 Stop 和 PreToolUse 两种 hook 事件类型。

## 输入契约

### 通用输入格式

所有 hook 事件通过 stdin 接收 JSON 输入：

```
stdin ← JSON(HookInput)
```

### Stop 事件输入

```json
{
  "session_id": "string (必需)",
  "stop_hook_active": "boolean"
}
```

**字段说明**:
| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| session_id | string | 是 | Claude Code 会话 ID |
| stop_hook_active | boolean | 否 | 是否已由其他 stop hook 触发继续 |

### PreToolUse 事件输入（AskUserQuestion）

```json
{
  "session_id": "string (必需)",
  "transcript_path": "string",
  "cwd": "string",
  "permission_mode": "string",
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {
    "questions": [...]
  },
  "tool_use_id": "string"
}
```

**字段说明**:
| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| session_id | string | 是 | Claude Code 会话 ID |
| hook_event_name | string | 是 | 事件类型，固定为 "PreToolUse" |
| tool_name | string | 是 | 工具名称，固定为 "AskUserQuestion" |
| tool_input | object | 是 | 工具特定输入参数 |
| tool_use_id | string | 是 | 工具调用 ID |
| transcript_path | string | 否 | 会话记录文件路径 |
| cwd | string | 否 | 当前工作目录 |
| permission_mode | string | 否 | 权限模式 |

## 输出契约

### 通用输出格式

所有 hook 事件通过 stdout 返回 JSON 输出：

```
stdout → JSON(HookOutput)
stderr → 日志信息（仅在 verbose/debug 模式）
exit code → 0 (成功) 或 2 (阻塞错误)
```

### Stop 事件输出

#### 允许停止（allow_stop = true）

```json
{
  "reason": "工作已完成"
}
```

**行为**: Claude Code 停止执行

#### 阻止停止（allow_stop = false）

```json
{
  "decision": "block",
  "reason": "需要继续完善测试用例"
}
```

**行为**: Claude Code 继续工作，`reason` 字段作为反馈传入

### PreToolUse 事件输出（AskUserQuestion）

#### 允许工具调用（allow_stop = true）

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "permissionDecisionReason": "问题合理，可以向用户提问"
  }
}
```

**行为**: Claude Code 执行 AskUserQuestion 工具调用

#### 阻止工具调用（allow_stop = false）

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "应该先添加更多代码注释"
  }
}
```

**行为**: Claude Code 取消 AskUserQuestion 工具调用，`permissionDecisionReason` 作为反馈传入

## 错误处理

### 输入解析错误

```
exit code: 2
stderr: "failed to parse hook input: ..."
```

**行为**: hook 执行失败，Claude Code 记录错误

### SDK 调用失败

```
exit code: 1
stderr: "supervisor SDK failed: ..."
```

**行为**: hook 执行失败，Claude Code 记录错误

### 超时

```
exit code: 124 (或特定超时退出码)
stderr: "hook execution timeout"
```

**行为**: hook 超时（600 秒），Claude Code 根据配置决定是否继续

## 命令行参数

### --session-id 参数

```bash
ccc supervisor-hook --session-id <session_id>
```

**用途**: 直接指定 session ID，跳过 stdin 解析

**优先级**: 命令行参数 > stdin 输入

## 环境变量

### CCC_SUPERVISOR_ID

```bash
CCC_SUPERVISOR_ID=<session_id> ccc supervisor-hook
```

**用途**: 传递 supervisor 会话 ID，必需

### CCC_SUPERVISOR_HOOK

```bash
CCC_SUPERVISOR_HOOK=1
```

**用途**: 防止递归调用，当设置为 "1" 时，hook 直接返回允许决策

## 实现要求

### 向后兼容

1. 必须支持旧的 `StopHookInput` 结构（只有 `session_id` 和 `stop_hook_active` 字段）
2. 当 `hook_event_name` 字段不存在时，默认为 Stop 事件
3. Stop 事件的输出格式必须保持不变

### 事件类型识别

1. 首先检查 `hook_event_name` 字段
2. 如果值为 "PreToolUse"，使用 PreToolUse 输出格式
3. 否则，使用 Stop 输出格式（默认）

### 迭代计数

1. 无论事件类型，都必须增加迭代计数
2. 达到最大迭代次数时，返回允许决策

## 测试用例

### TC-001: Stop 事件 - 允许停止

**输入**:
```json
{"session_id": "test-001", "stop_hook_active": false}
```

**预期输出**:
```json
{"reason": "..."}
```

**exit code**: 0

### TC-002: Stop 事件 - 阻止停止

**输入**:
```json
{"session_id": "test-002", "stop_hook_active": false}
```

**预期输出**:
```json
{"decision": "block", "reason": "..."}
```

**exit code**: 0

### TC-003: PreToolUse 事件 - 允许调用

**输入**:
```json
{
  "session_id": "test-003",
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {"questions": [...]},
  "tool_use_id": "toolu_001"
}
```

**预期输出**:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "permissionDecisionReason": "..."
  }
}
```

**exit code**: 0

### TC-004: PreToolUse 事件 - 阻止调用

**输入**:
```json
{
  "session_id": "test-004",
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {"questions": [...]},
  "tool_use_id": "toolu_002"
}
```

**预期输出**:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "..."
  }
}
```

**exit code**: 0

### TC-005: 递归调用防护

**环境变量**: `CCC_SUPERVISOR_HOOK=1`

**预期输出**: 直接返回允许决策，不调用 SDK

**exit code**: 0

### TC-006: 迭代计数限制

**前置条件**: 迭代计数已达上限（默认 20）

**输入**: 任意

**预期输出**: 返回允许决策

**exit code**: 0

### TC-007: 向后兼容 - 无 hook_event_name

**输入**:
```json
{"session_id": "test-007"}
```

**预期行为**: 默认为 Stop 事件，使用 Stop 输出格式

**exit code**: 0
