# Design: refactor-supervisor-mode

## Context

当前 Supervisor Mode 实现虽然功能可用，但存在可维护性问题：
- 日志输出混杂，没有统一格式
- 错误处理不规范
- 配置项过多（prompt_path、log_level 不需要可配置）

Phase 1 已实现的基础设施：
- `internal/logger/` - 结构化日志系统
- `internal/errors/` - 统一错误处理
- `internal/claude_agent_sdk/` - Claude 命令行封装

## Goals / Non-Goals

### Goals
- 简化配置，移除不需要的可配置项
- 使用已实现的日志和错误处理系统
- 为集成 Claude Agent SDK 做好准备

### Non-Goals
- 不改变 Supervisor Mode 的核心工作流程
- 不修改已有的 spec 需求（只简化）
- 不添加新的用户可见功能

## Decisions

### 决策 1: 简化配置结构

**选择**: 移除 `prompt_path` 和 `log_level` 配置项

**原因**:
- Supervisor prompt 使用硬编码的默认值（已在 `hook.go` 中实现）
- Log level 固定为 `info`，不需要用户配置
- 减少配置复杂度

**新的配置结构**:
```json
{
  "supervisor": {
    "enabled": false,
    "max_iterations": 20,
    "timeout_seconds": 600
  }
}
```

**向后兼容**:
- 如果配置文件中存在 `prompt_path` 或 `log_level`，将被忽略
- 不影响已有用户的使用

### 决策 2: 使用现有的日志和错误处理系统

**选择**: 使用 Phase 1 已实现的 `logger.Logger` 和 `errors.AppError`

**原因**:
- 避免重复实现
- 统一代码风格
- 已经过测试验证

## Migration Plan

### 步骤
1. 修改 `internal/config/supervisor.go` - 移除 `PromptPath` 和 `LogLevel` 字段
2. 修改 `internal/cli/hook.go` - 使用硬编码的 prompt 和固定的 log level
3. 更新测试以适配新的配置结构

### Rollback
- Git 提供完整历史，可随时回滚
