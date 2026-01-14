# Change: 重构 Supervisor Mode 启用方式为 Slash Command

## Why

当前 Supervisor Mode 通过 `CCC_SUPERVISOR=1` 环境变量或 `ccc.json` 中的 `supervisor.enabled` 配置来启用，这种方式不够灵活，用户需要在启动 ccc 前设置环境变量或修改配置文件。

新的方式通过 Claude Code 的 Slash Command 机制（`/supervisor`）来动态启用 Supervisor Mode，用户可以在与 Agent 确认需求后再决定是否启用 Supervisor，使用体验更加流畅。

## What Changes

- **去掉** `CCC_SUPERVISOR` 环境变量的支持
- **去掉** `SupervisorConfig.Enabled` 字段（保留 `MaxIterations` 和 `TimeoutSeconds`）
- **新增** `supervisor-mode` 子命令（`ccc supervisor-mode on/off`）用于控制 Supervisor 启用状态
- **State 文件** 增加 `Enabled` 字段（默认 false），用于存储当前 session 的 Supervisor 启用状态
- **ccc 启动时** 无论什么模式都设置 `CCC_SUPERVISOR_ID`（如果环境变量没有就新生成，否则复用）
- **总是** 使用 `SwitchWithHook()` 生成带 Stop Hook 的 settings（不再需要普通模式的 `Switch()`）
- **Hook 改为** 从 state 文件读取 `Enabled` 字段判断是否执行 Supervisor review
- **每次 ccc 启动时** 都覆盖创建 `~/.claude/commands/supervisor.md` 和 `supervisoroff.md`
- **为新增的 `supervisor-mode` 子命令编写单元测试**

## Impact

- **Affected specs**: `cli`, `supervisor-hooks`, `core-config`
- **Affected code**:
  - `internal/cli/cli.go` - 新增 `supervisor-mode` 子命令
  - `internal/cli/exec.go` - 总是设置 `CCC_SUPERVISOR_ID`，去掉 `supervisorMode` 参数
  - `internal/cli/hook.go` - 从 state 文件读取 `Enabled` 判断
  - `internal/provider/provider.go` - 删除 `Switch()`，`SwitchWithHook()` 增加创建 commands 文件
  - `internal/config/supervisor.go` - 去掉 `Enabled` 字段
  - `internal/supervisor/state.go` - 增加 `Enabled` 字段
  - 测试文件更新
- **Documentation**: README.md, README-CN.md

## 用户使用流程（新方式）

1. 用户启动 ccc（不需要设置任何环境变量）
   ```bash
   ccc glm
   ```

2. 用户和 Agent 沟通，确认需求和方案

3. 用户输入 `/supervisor 好，开始执行`
   - 触发 `supervisor.md` slash command
   - 执行 `ccc supervisor-mode on`
   - state 文件 `enabled` 字段设为 `true`

4. Agent 执行任务，完成后触发 Stop Hook
   - hook 读取 state 文件，发现 `enabled=true`
   - 执行 supervisor review

5. Agent 继续执行或完成任务
