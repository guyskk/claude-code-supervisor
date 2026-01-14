## MODIFIED Requirements

### Requirement: Supervisor Mode 启用判断

系统 SHALL 根据 state 文件中的 `Enabled` 字段判断是否执行 Supervisor review。

#### Scenario: State 文件 Enabled 为 true 时执行 Supervisor
- **GIVEN** State 文件中 `Enabled` 字段为 `true`
- **WHEN** Stop hook 被触发
- **THEN** 应当执行 Supervisor review
- **AND** 应当根据结果决定是否允许停止

#### Scenario: State 文件 Enabled 为 false 时跳过 Supervisor
- **GIVEN** State 文件中 `Enabled` 字段为 `false`
- **WHEN** Stop hook 被触发
- **THEN** 应当输出空 decision（允许停止）
- **AND** 应当立即返回（不执行 Supervisor）

#### Scenario: State 文件不存在时默认跳过
- **GIVEN** State 文件不存在
- **WHEN** Stop hook 被触发
- **THEN** 应当输出空 decision（允许停止）
- **AND** 应当立即返回（不执行 Supervisor）

### Requirement: supervisor-hook 子命令

系统 SHALL 提供 `supervisor-hook` 子命令处理 Stop hook 事件，根据 state 文件的 `Enabled` 字段决定是否执行 Supervisor review。

#### Scenario: 正常 Hook 调用（使用 Claude Agent SDK）
- **GIVEN** State 文件中 `Enabled` 字段为 `true`
- **AND** stdin 包含有效的 StopHookInput JSON
- **WHEN** 执行 `ccc supervisor-hook`
- **THEN** 应当使用 `schlunsen/claude-agent-sdk-go` 创建客户端
- **AND** 使用 fork session 模式恢复当前 session（`WithForkSession(true)` 和 `WithSessionID(id)`）
- **AND** Claude 能够访问原 session 的完整上下文
- **AND** 应当根据 Supervisor 结果输出 JSON 到 stdout

#### Scenario: State 文件 Enabled 为 false 时跳过
- **GIVEN** State 文件中 `Enabled` 字段为 `false`
- **WHEN** 执行 `ccc supervisor-hook`
- **THEN** 应当输出 `{"decision":"","":""}` 到 stdout
- **AND** 应当立即返回（不调用 Supervisor）

## REMOVED Requirements

### Requirement: Supervisor Mode 启动（环境变量方式）

**Reason**: 不再通过 `CCC_SUPERVISOR=1` 环境变量判断是否启动 Supervisor Mode，改用 state 文件的 `Enabled` 字段。

**Migration**: 用户改用 `/supervisor` slash command 启用 Supervisor Mode，会自动设置 state 文件的 `Enabled` 字段。

- ~~**GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置~~
- ~~**WHEN** 用户执行 `ccc <provider>`~~
- ~~**THEN** 应当生成带 Stop hook 的 `settings.json`~~

### Requirement: 防止死循环 - 环境变量

**Reason**: 不再需要通过 `CCC_SUPERVISOR_HOOK` 环境变量防止死循环，新的实现通过 state 文件的 `Enabled` 字段控制。

**Migration**: 无需迁移，此检查不再需要。

- ~~**GIVEN** 环境变量 `CCC_SUPERVISOR_HOOK=1` 已设置~~
- ~~**WHEN** 执行 `ccc supervisor-hook`~~
- ~~**THEN** 应当输出 `{"decision":"","":""}` 到 stdout~~

- ~~**GIVEN** hook 需要调用 Supervisor claude~~
- ~~**WHEN** 构建 Supervisor claude 命令~~
- ~~**THEN** 应当设置 `CCC_SUPERVISOR_HOOK=1` 环境变量~~

### Requirement: Settings 文件生成（条件生成）

**Reason**: 不再需要区分 Supervisor Mode 和普通模式，总是生成带 Stop Hook 的 settings。

**Migration**: 无需迁移，新实现总是生成带 Stop Hook 的 settings。

- ~~**GIVEN** Supervisor Mode 启用~~
- ~~**WHEN** 系统生成配置~~
- ~~**THEN** 应当将配置写入 `~/.claude/settings.json`~~
- ~~**AND** settings 中应当包含 `hooks.Stop` 配置~~

### Requirement: Fork Session 使用（环境变量隔离）

**Reason**: 不再需要防止 hook 递归调用的逻辑，新的实现通过 state 文件的 `Enabled` 字段控制。

**Migration**: 无需迁移，新实现不需要 supervisor 专用 settings。

- ~~**GIVEN** 主 settings 包含 Stop hook~~
- ~~**WHEN** 调用 Supervisor~~
- ~~**THEN** 应当使用 supervisor 专用 settings~~
- ~~**AND** supervisor settings 不应包含任何 hooks~~

## ADDED Requirements

### Requirement: State 文件 Enabled 字段

State 文件 SHALL 包含 `Enabled` 字段用于控制 Supervisor Mode 是否执行。

#### Scenario: State 文件结构（更新）
- **GIVEN** session_id 为 "abc123"
- **WHEN** 系统保存状态
- **THEN** 状态文件应当包含：
  - `session_id`: "abc123"
  - `enabled`: 是否启用 Supervisor Mode（默认 false）
  - `count`: 迭代次数
  - `created_at`: 创建时间（ISO 8601）
  - `updated_at`: 更新时间（ISO 8601）

#### Scenario: 加载旧 State 文件兼容性
- **GIVEN** State 文件存在但不包含 `enabled` 字段
- **WHEN** 系统加载状态
- **THEN** 应当将 `enabled` 默认设为 `false`
- **AND** 应当正常加载其他字段

### Requirement: Slash Command 集成

系统 SHALL 通过创建 slash command 文件实现 `/supervisor` 命令。

#### Scenario: supervisor.md 内容格式
- **GIVEN** ccc 执行 `SwitchWithHook()`
- **WHEN** 系统创建 `~/.claude/commands/supervisor.md`
- **THEN** 文件内容应当为：
```markdown
---
description: Enable supervisor mode
---
$ARGUMENTS!`ccc supervisor-mode on`
```

#### Scenario: supervisoroff.md 内容格式
- **GIVEN** ccc 执行 `SwitchWithHook()`
- **WHEN** 系统创建 `~/.claude/commands/supervisoroff.md`
- **THEN** 文件内容应当为：
```markdown
---
description: Disable supervisor mode
---
$ARGUMENTS!`ccc supervisor-mode off`
```

#### Scenario: 用户使用 /supervisor 命令
- **GIVEN** 用户在 Claude Code 中输入 `/supervisor 好，开始执行`
- **WHEN** Claude Code 解析 slash command
- **THEN** 应当执行 `ccc supervisor-mode on 好，开始执行`
- **AND** supervisor-mode 子命令应当忽略额外参数
- **AND** state 文件的 `enabled` 字段应当被设为 `true`
- **AND** 后续 Stop hook 将执行 Supervisor review

#### Scenario: 用户使用 /supervisoroff 命令
- **GIVEN** 用户在 Claude Code 中输入 `/supervisoroff`
- **WHEN** Claude Code 解析 slash command
- **THEN** 应当执行 `ccc supervisor-mode off`
- **AND** state 文件的 `enabled` 字段应当被设为 `false`
- **AND** 后续 Stop hook 将跳过 Supervisor review
