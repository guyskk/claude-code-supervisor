# Design: integrate-claude-sdk

## Context

Supervisor Mode 需要调用 Claude 来审查当前 session 的工作完成情况。当前实现使用自定义的 `internal/claude_agent_sdk/` 包，但该包功能有限。

## Goals / Non-Goals

### Goals
- 使用成熟的 Go SDK 替代自定义实现
- 简化 Supervisor Hook 的代码
- 获得更完整的会话管理功能

### Non-Goals
- 不修改普通模式的 claude 启动方式（继续使用 syscall.Exec）
- 不在其他地方使用 SDK（仅用于 Supervisor Hook）

## Decisions

### 决策 1: 只在 Supervisor Hook 中使用 SDK

**选择**: 仅在 `internal/cli/hook.go` 中使用 Go SDK

**原因**:
- 普通模式只需要启动 claude 进程并替换当前进程，`syscall.Exec` 更合适
- Supervisor Mode 需要 fork session 并获取结构化输出，SDK 提供了完整支持
- 减少对外部依赖的引入范围

### 决策 2: 使用 schlunsen/claude-agent-sdk-go

**选择**: 使用 `github.com/schlunsen/claude-agent-sdk-go`

**原因**:
- 零外部依赖（只使用 Go 标准库）
- 与 Claude Code CLI 完全兼容
- 支持完整的会话管理（包括 fork session）
- 活跃维护，功能完备

**核心 API**:
```go
// 创建客户端
options := sdk.NewClaudeAgentOptions().
    WithOutputFormat("stream-json").
    WithJSONSchema(jsonSchema).
    WithForkSession(true).
    WithSessionID(sessionID)

client, _ := sdk.NewClient(options)
client.Connect(ctx)
client.Query(ctx, prompt)

// 接收响应
for msg := range client.ReceiveResponse(ctx) {
    // 处理消息
}
```

## Migration Plan

### 步骤
1. 添加 Go SDK 依赖到 `go.mod`
2. 重写 `internal/cli/hook.go` 中的 Supervisor 执行逻辑
3. 删除 `internal/claude_agent_sdk/` 包
4. 更新相关测试

### Rollback
- Git 提供完整历史
