# Change: improve-supervisor-logging

## Why

当前 Supervisor Mode 的验证非常困难：
- Supervisor 作为独立会话运行，其执行过程在原会话界面上完全不可见
- 用户无法直观地看到 Supervisor 的判断过程和结果
- 只能通过手动查看分散的日志文件来追踪运行状态

## What Changes

- 在启动 Supervisor Mode 时显示 log 文件路径提示
- 增强 log 文件内容，使其更易读和结构化
- 在 hook 执行时输出更清晰的进度信息到 stderr（可通过 ctrl+o 查看）

## Impact

- Affected specs: `supervisor-hooks`
- Affected code:
  - `internal/cli/cli.go` - 启动时的提示信息
  - `internal/cli/hook.go` - hook 执行时的日志输出
