# Proposal: integrate-claude-sdk

## 概述

集成 `schlunsen/claude-agent-sdk-go` 作为 Supervisor Mode 的 Claude SDK 基础，替代当前的自定义实现。

## Why

当前 `internal/claude_agent_sdk/` 包是一个自定义的 claude 命令行封装，功能有限：
- 缺少完整的会话管理（如 fork session）
- 缺少对 stream-json 模式的完整支持
- 需要自己维护进程管理逻辑

`schlunsen/claude-agent-sdk-go` 是一个成熟的 Go SDK：
- 完整的会话管理功能（包括 fork session）
- 完善的 stream-json 消息解析
- 零外部依赖（纯标准库实现）
- 与 Claude Code CLI 完全兼容（文件系统、session 互通）

## What Changes

- **仅用于 Supervisor Mode**：只在 `internal/cli/hook.go` 中使用 SDK
- **普通模式不变**：普通模式继续使用 `syscall.Exec` 直接调用 claude 命令
- **删除自定义封装**：删除 `internal/claude_agent_sdk/` 包
- **添加外部依赖**：在 `go.mod` 中添加 `github.com/schlunsen/claude-agent-sdk-go`

## Impact

- Affected specs: `supervisor-hooks`
- Affected code:
  - `internal/cli/hook.go` - 使用 Go SDK 替代自定义 claude_agent_sdk
  - `go.mod` - 添加外部依赖
  - `internal/claude_agent_sdk/` - 删除整个包

## SDK 使用方式

```go
import (
    "context"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
)

// 在 Supervisor Hook 中
func runSupervisor(sessionID, prompt string) (*SupervisorResult, error) {
    options := sdk.NewClaudeAgentOptions().
        WithOutputFormat("stream-json").
        WithJSONSchema(supervisorJSONSchema).
        WithForkSession(true).
        WithSessionID(sessionID)

    client, err := sdk.NewClient(options)
    if err != nil {
        return nil, err
    }
    defer client.Close(context.Background())

    if err := client.Connect(context.Background()); err != nil {
        return nil, err
    }

    if err := client.Query(context.Background(), prompt); err != nil {
        return nil, err
    }

    // 接收响应并提取结构化输出
    for msg := range client.ReceiveResponse(context.Background()) {
        if resultMsg, ok := msg.(*sdk.ResultMessage); ok {
            return parseSupervisorResult(resultMsg.StructuredOutput)
        }
    }

    return nil, fmt.Errorf("no result from supervisor")
}
```

## 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 外部依赖引入 | 低 | SDK 零外部依赖，只使用 Go 标准库 |
| API 兼容性 | 低 | SDK 与 Claude Code CLI 完全兼容 |
| 维护依赖 | 中 | SDK 是活跃维护的开源项目 |
