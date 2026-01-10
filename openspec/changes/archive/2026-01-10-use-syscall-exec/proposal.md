# Proposal: use-syscall-exec

## Summary

使用 `syscall.Exec` 替换 `exec.Command().Run()` 来运行 claude 命令，实现真正的 Unix exec 语义。

## Motivation

当前实现使用 `exec.Command().Run()` 创建子进程运行 claude：

```go
claudeCmd := exec.Command("claude", cmdArgs...)
claudeCmd.Run()
```

这种方式的问题：
1. **进程树冗余**：进程链为 `shell → ccc → claude`，ccc 只是在等待 claude 结束
2. **信号传递间接**：Ctrl+C 等信号需要 ccc 转发给 claude
3. **退出码处理复杂**：需要从子进程错误中提取退出码

Unix `exec` 系统调用可以让当前进程直接"变成"目标程序，更符合 ccc 作为"配置切换器"的定位——配置好环境后就应该让位给 claude。

## Proposed Solution

使用条件编译实现跨平台支持：
- **Unix 系统** (Linux/macOS)：使用 `syscall.Exec` 直接替换进程
- **Windows**：保持现有的 `exec.Command().Run()` 方式（Windows 没有 exec 语义）

## Impact

- **进程树**：从 `shell → ccc → claude` 变为 `shell → claude`
- **信号处理**：信号直接发送给 claude，无需 ccc 转发
- **退出码**：直接继承 claude 的退出码
- **资源占用**：减少一个进程

## Affected Specs

- `cli`：新增 Claude 进程执行相关的 requirements

## Files Changed

- `internal/cli/exec_unix.go`（新增）：Unix 平台的 exec 实现
- `internal/cli/exec_windows.go`（新增）：Windows 平台的 exec 实现
- `internal/cli/cli.go`：重构 runClaude 函数，调用平台特定实现
