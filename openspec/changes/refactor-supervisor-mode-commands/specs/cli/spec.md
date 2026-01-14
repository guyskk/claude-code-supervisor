## ADDED Requirements

### Requirement: supervisor-mode 子命令

系统 SHALL 提供 `supervisor-mode` 子命令用于动态控制 Supervisor Mode 的启用状态。

#### Scenario: 启用 Supervisor Mode
- **GIVEN** 环境变量 `CCC_SUPERVISOR_ID` 已设置
- **WHEN** 用户执行 `ccc supervisor-mode on`
- **THEN** 应当加载对应 session 的 state 文件
- **AND** 应当设置 `Enabled` 字段为 `true`
- **AND** 应当保存 state 文件
- **AND** 应当通过 supervisor logger 输出成功信息到 stderr
- **AND** 不应当输出任何内容到 stdout

#### Scenario: 禁用 Supervisor Mode
- **GIVEN** 环境变量 `CCC_SUPERVISOR_ID` 已设置
- **WHEN** 用户执行 `ccc supervisor-mode off`
- **THEN** 应当加载对应 session 的 state 文件
- **AND** 应当设置 `Enabled` 字段为 `false`
- **AND** 应当保存 state 文件
- **AND** 应当通过 supervisor logger 输出成功信息到 stderr
- **AND** 不应当输出任何内容到 stdout

#### Scenario: 默认参数为启用
- **GIVEN** 环境变量 `CCC_SUPERVISOR_ID` 已设置
- **WHEN** 用户执行 `ccc supervisor-mode`（不带参数）
- **THEN** 应当视为 `on` 参数
- **AND** 应当设置 `Enabled` 字段为 `true`

#### Scenario: 缺少 CCC_SUPERVISOR_ID
- **GIVEN** 环境变量 `CCC_SUPERVISOR_ID` 未设置
- **WHEN** 用户执行 `ccc supervisor-mode on`
- **THEN** 应当输出错误信息到 stderr
- **AND** 应当返回非零退出码

## MODIFIED Requirements

### Requirement: Supervisor Mode 命令行

系统 SHALL 总是生成带 Stop Hook 的 settings.json，通过 state 文件的 `Enabled` 字段控制是否执行 Supervisor review。

#### Scenario: 总是设置 CCC_SUPERVISOR_ID
- **WHEN** 用户执行 `ccc <provider>`
- **THEN** 应当检查 `CCC_SUPERVISOR_ID` 环境变量
- **AND** 如果未设置，应当生成新的 UUID 并设置到环境变量
- **AND** 如果已设置，应当复用现有值

#### Scenario: 总是生成带 Hook 的 Settings
- **GIVEN** 用户执行 `ccc kimi`（无环境变量）
- **WHEN** 系统生成配置
- **THEN** 应当调用 `SwitchWithHook()` 生成配置
- **AND** 应当将配置写入 `~/.claude/settings.json`
- **AND** settings 中应当包含 `hooks.Stop` 配置
- **AND** 应当创建 `~/.claude/commands/supervisor.md` 文件
- **AND** 应当创建 `~/.claude/commands/supervisoroff.md` 文件

#### Scenario: 创建 supervisor.md 命令文件
- **GIVEN** ccc 执行 `SwitchWithHook()`
- **WHEN** 系统创建命令文件
- **THEN** 应当创建 `~/.claude/commands/supervisor.md`
- **AND** 文件内容应当包含 frontmatter（description: Enable supervisor mode）
- **AND** 文件内容应当包含命令 `$ARGUMENTS!`ccc supervisor-mode on``

#### Scenario: 创建 supervisoroff.md 命令文件
- **GIVEN** ccc 执行 `SwitchWithHook()`
- **WHEN** 系统创建命令文件
- **THEN** 应当创建 `~/.claude/commands/supervisoroff.md`
- **AND** 文件内容应当包含 frontmatter（description: Disable supervisor mode）
- **AND** 文件内容应当包含命令 `$ARGUMENTS!`ccc supervisor-mode off``

### Requirement: 帮助信息显示

系统 SHALL 提供清晰的帮助信息，包括用法、命令和可用提供商列表。

#### Scenario: 显示帮助（更新）
- **WHEN** 用户请求帮助
- **THEN** 应当显示用法说明
- **AND** 应当显示可用命令列表（包括 `supervisor-mode`）
- **AND** 应当显示环境变量说明（不包括 `CCC_SUPERVISOR`）
- **AND** 如果配置可用，应当显示提供商列表

## REMOVED Requirements

### Requirement: Supervisor Mode 命令行（旧版本）

**Reason**: 不再通过 `CCC_SUPERVISOR=1` 环境变量启用 Supervisor Mode，改用 `supervisor-mode` 子命令和 state 文件控制。

**Migration**: 用户改用 `/supervisor` slash command 启用 Supervisor Mode。

- ~~**GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置~~
- ~~**WHEN** 用户执行 `ccc kimi`~~
- ~~**THEN** 应当启用 Supervisor Mode~~

### Requirement: supervisor-hook 子命令（环境变量检查）

**Reason**: 不再需要通过 `CCC_SUPERVISOR_HOOK` 环境变量防止死循环，新的实现通过 state 文件的 `Enabled` 字段控制。

**Migration**: 无需迁移，此检查不再需要。

- ~~**GIVEN** 环境变量 `CCC_SUPERVISOR_HOOK=1` 已设置~~
- ~~**WHEN** 执行 `ccc supervisor-hook`~~
- ~~**THEN** 应当跳过 hook 执行~~
