## MODIFIED Requirements

### Requirement: 提供商切换

系统 SHALL 能够切换到指定的提供商，包括配置合并和文件保存。切换时应当清空 settings.json 中的 env 字段以防止配置污染。

#### Scenario: 切换到存在的提供商
- **GIVEN** 配置中存在提供商 "kimi"
- **AND** `~/.claude/settings.json` 存在且包含 `env` 字段
- **WHEN** 调用 `Switch(config, "kimi")`
- **THEN** 应当合并 settings 和 kimi 提供商配置
- **AND** 应当保存到 `settings-kimi.json`
- **AND** 应当将 settings.json 中的 env 字段清空为 `{}`
- **AND** 应当输出提示信息 "Cleared env field in settings.json to prevent configuration pollution"
- **AND** 应当更新 current_provider 为 "kimi"
- **AND** 应当返回合并后的配置

#### Scenario: 切换到存在的提供商（settings.json 无 env）
- **GIVEN** 配置中存在提供商 "kimi"
- **AND** `~/.claude/settings.json` 不存在或没有 `env` 字段
- **WHEN** 调用 `Switch(config, "kimi")`
- **THEN** 应当合并 settings 和 kimi 提供商配置
- **AND** 应当保存到 `settings-kimi.json`
- **AND** 不应输出任何关于清空 env 的提示
- **AND** 应当更新 current_provider 为 "kimi"
- **AND** 应当返回合并后的配置

#### Scenario: 切换到不存在的提供商
- **GIVEN** 配置中不存在提供商 "unknown"
- **WHEN** 调用 `Switch(config, "unknown")`
- **THEN** 应当返回错误
- **AND** 错误信息应当包含 "provider 'unknown' not found"

## ADDED Requirements

### Requirement: 清空 settings.json 中的 env 字段

系统 SHALL 能够清空 `~/.claude/settings.json` 中的 `env` 字段，以防止它污染提供商特定的配置。

#### Scenario: settings.json 存在且包含 env 字段
- **GIVEN** `~/.claude/settings.json` 存在
- **AND** 文件包含 `{"env": {"ANTHROPIC_AUTH_TOKEN": "sk-old"}}`
- **WHEN** 调用 `ClearEnvInSettings()`
- **THEN** 应当将 env 字段设置为空对象 `{}`
- **AND** 应当保留其他字段（如 permissions、alwaysThinkingEnabled）
- **AND** 应当返回 true 表示已清空

#### Scenario: settings.json 不存在
- **GIVEN** `~/.claude/settings.json` 不存在
- **WHEN** 调用 `ClearEnvInSettings()`
- **THEN** 不应创建文件
- **AND** 应当返回 false 表示未清空

#### Scenario: settings.json 存在但没有 env 字段
- **GIVEN** `~/.claude/settings.json` 存在
- **AND** 文件不包含 `env` 字段
- **WHEN** 调用 `ClearEnvInSettings()`
- **THEN** 文件内容不应改变
- **AND** 应当返回 false 表示未清空
