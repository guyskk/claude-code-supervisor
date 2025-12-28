# provider-management 规范变更

## ADDED Requirements

### Requirement: 提供商切换

系统 SHALL 能够切换到指定的提供商，包括配置合并和文件保存。

#### Scenario: 切换到存在的提供商
- **GIVEN** 配置中存在提供商 "kimi"
- **WHEN** 调用 `Switch(config, "kimi")`
- **THEN** 应当合并 settings 和 kimi 提供商配置
- **AND** 应当保存到 `settings-kimi.json`
- **AND** 应当更新 current_provider 为 "kimi"
- **AND** 应当返回合并后的配置

#### Scenario: 切换到不存在的提供商
- **GIVEN** 配置中不存在提供商 "unknown"
- **WHEN** 调用 `Switch(config, "unknown")`
- **THEN** 应当返回错误
- **AND** 错误信息应当包含 "provider 'unknown' not found"

### Requirement: 配置深度合并

系统 SHALL 能够深度合并基础配置和提供商配置，提供商配置优先。

#### Scenario: 合并 env 字段
- **GIVEN** settings.env 为 `{"API_TIMEOUT": "30000"}`
- **AND** provider.env 为 `{"BASE_URL": "https://api.kimi.com", "API_TIMEOUT": "60000"}`
- **WHEN** 执行深度合并
- **THEN** 结果 env 应当为 `{"BASE_URL": "https://api.kimi.com", "API_TIMEOUT": "60000"}`

#### Scenario: 合并非 env 字段
- **GIVEN** settings.permissions 为 `{"allow": ["Edit"]}`
- **AND** provider.permissions 为 `{"allow": ["WebSearch"]}`
- **WHEN** 执行深度合并
- **THEN** 结果 permissions 应当被提供商配置覆盖

#### Scenario: 提供商无覆盖
- **GIVEN** settings.alwaysThinkingEnabled 为 true
- **AND** provider 配置为空
- **WHEN** 执行深度合并
- **THEN** 结果应当保留 settings 的所有字段

### Requirement: 认证令牌提取

系统 SHALL 能够从合并后的配置中提取 ANTHROPIC_AUTH_TOKEN。

#### Scenario: 提取有效的令牌
- **GIVEN** 合并配置的 env.ANTHROPIC_AUTH_TOKEN 为 "sk-xxx"
- **WHEN** 调用 `GetAuthToken(config)`
- **THEN** 应当返回 "sk-xxx"

#### Scenario: 令牌未设置
- **GIVEN** 合并配置中没有 ANTHROPIC_AUTH_TOKEN
- **WHEN** 调用 `GetAuthToken(config)`
- **THEN** 应当返回占位符 "PLEASE_SET_ANTHROPIC_AUTH_TOKEN"

### Requirement: 配置文件生成

系统 SHALL 能够为特定提供商生成 Claude 配置文件。

#### Scenario: 生成提供商配置文件
- **GIVEN** 提供商名称为 "kimi"
- **AND** 合并后的配置
- **WHEN** 保存提供商配置
- **THEN** 应当创建 `~/.claude/settings-kimi.json`
- **AND** 文件内容应当是格式化的 JSON

#### Scenario: 无提供商名称
- **GIVEN** 提供商名称为空字符串
- **WHEN** 获取配置文件路径
- **THEN** 应当返回 `~/.claude/settings.json`
