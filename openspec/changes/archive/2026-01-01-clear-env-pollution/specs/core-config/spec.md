## ADDED Requirements

### Requirement: settings.json 路径获取

系统 SHALL 提供获取 `~/.claude/settings.json` 路径的函数。

#### Scenario: 默认路径
- **WHEN** 未设置 CCC_CONFIG_DIR 环境变量
- **THEN** 应当返回 `~/.claude/settings.json`

#### Scenario: 自定义路径
- **GIVEN** CCC_CONFIG_DIR 设置为 `/custom/path`
- **WHEN** 调用 `GetSettingsJSONPath()`
- **THEN** 应当返回 `/custom/path/settings.json`

### Requirement: 清空 settings.json 中的 env 字段

系统 SHALL 能够清空 `~/.claude/settings.json` 中的 `env` 字段，保留其他配置不变。

#### Scenario: 成功清空 env 字段
- **GIVEN** `~/.claude/settings.json` 包含 `{"permissions": {...}, "env": {"API_KEY": "xxx"}}`
- **WHEN** 调用 `ClearEnvInSettings()`
- **THEN** 文件内容应当变为 `{"permissions": {...}, "env": {}}`
- **AND** 应当返回 true
- **AND** error 应当为 nil

#### Scenario: settings.json 不存在
- **GIVEN** `~/.claude/settings.json` 不存在
- **WHEN** 调用 `ClearEnvInSettings()`
- **THEN** 不应创建文件
- **AND** 应当返回 false
- **AND** error 应当为 nil

#### Scenario: settings.json 格式错误
- **GIVEN** `~/.claude/settings.json` 包含无效的 JSON
- **WHEN** 调用 `ClearEnvInSettings()`
- **THEN** 应当返回错误
- **AND** 错误信息应当包含 "failed to parse settings.json"

#### Scenario: 无 env 字段
- **GIVEN** `~/.claude/settings.json` 包含 `{"permissions": {...}}`（无 env 字段）
- **WHEN** 调用 `ClearEnvInSettings()`
- **THEN** 文件内容不应改变
- **AND** 应当返回 false
- **AND** error 应当为 nil
