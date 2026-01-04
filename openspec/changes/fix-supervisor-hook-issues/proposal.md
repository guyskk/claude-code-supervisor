# Proposal: fix-supervisor-hook-issues

## 概述

修复 `add-supervisor-hooks-mode` 实现中的多个问题，包括代码重复、参数错误、输出格式问题等。

## 动机

当前实现存在以下问题需要修复：

1. **ccc 二进制文件被提交** - 构建产物不应该提交到版本控制
2. **runSupervisor 不必要的 providerName 参数** - provider 信息已通过 SwitchWithHook 保存，参数冗余
3. **runClaude 和 runSupervisor 代码重复** - 两个函数有大量重复逻辑
4. **CCC_SUPERVISOR_HOOK=1 时的输出格式错误** - 应该返回固定 JSON 而不是空输出
5. **--state-dir 参数不需要** - 应该直接使用环境变量或默认路径
6. **supervisor hook 命令参数错误** - 应该使用 --fork-session 而不是 --print，不应使用 --system-prompt
7. **流式输出处理需要完善** - 确保原始内容记录和结果解析正确

## 变更内容

- **删除 ccc 二进制文件** 并更新 .gitignore
- **重构 runSupervisor** - 移除 providerName 参数，从 cfg.CurrentProvider 获取
- **合并 runClaude 和 runSupervisor** - 提取公共逻辑到单一函数
- **修改 RunSupervisorHook** - 当 CCC_SUPERVISOR_HOOK=1 时返回固定 JSON `{"decision":"","":""}`
- **移除 --state-dir 参数** - 使用 CCC_WORK_DIR 环境变量或默认 ~/.claude/ccc/
- **修改 supervisor hook 调用 claude 的命令**：
  - 使用 `--fork-session` 而不是 `--print`
  - 移除 `--system-prompt`（supervisor prompt 应作为 agent 的默认 prompt）
  - user prompt 为 supervisor prompt + 具体指令
- **完善流式输出处理** - 按行读取并解析 JSON，记录原始内容到文件

## 影响范围

- **受影响的 specs**:
  - `supervisor-hooks` - 修改多个 requirements
- **受影响的代码**:
  - `.gitignore` - 添加忽略规则
  - `internal/cli/cli.go` - 重构 runClaude，移除重复逻辑
  - `internal/cli/exec.go` - 重构 runSupervisor，移除 providerName 参数
  - `internal/cli/hook.go` - 修改 RunSupervisorHook 实现
  - `internal/provider/provider.go` - 更新 hook command（移除 --state-dir）
  - `internal/supervisor/state.go` - 使用 CCC_WORK_DIR 环境变量
