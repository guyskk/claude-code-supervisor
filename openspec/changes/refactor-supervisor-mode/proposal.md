# Proposal: refactor-supervisor-mode

## 概述

重构 Supervisor Mode 实现，简化配置并提高代码质量，为集成 Claude Agent SDK 做好准备。

## Why

当前实现存在以下问题：
1. 配置硬编码（最大迭代次数、超时时间）
2. 日志系统混乱（混杂 fmt.Fprintf 和不同级别输出）
3. 进程管理脆弱（缺少超时控制、清理机制）
4. 错误处理不规范（缺少统一的错误分类）

## What Changes

- 简化配置：移除 `prompt_path` 和 `log_level` 配置项，使用硬编码的默认值
- 统一日志：使用结构化日志系统（已在 Phase 1 实现）
- 统一错误处理：使用统一的错误类型和错误码（已在 Phase 1 实现）
- 保留现有的 supervisor 配置：`enabled`、`max_iterations`、`timeout_seconds`

## Impact

- Affected specs: `supervisor-hooks`
- Affected code:
  - `internal/config/supervisor.go` - 简化配置结构
  - `internal/cli/hook.go` - 使用固定 log level 和默认 prompt
  - `internal/supervisor/` - 使用新的日志和错误处理系统

## 向后兼容性

- 现有的 `supervisor` 配置段继续有效
- `prompt_path` 和 `log_level` 配置如果存在将被忽略
- 环境变量 `CCC_SUPERVISOR`、`CCC_SUPERVISOR_MAX_ITERATIONS` 继续有效
