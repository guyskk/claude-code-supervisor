# core-config 规范变更

## ADDED Requirements

### Requirement: ccc 配置结构

系统 SHALL 使用动态的 `map[string]interface{}` 来表示配置，以支持 Claude settings 的任意字段扩展。

#### Scenario: Config 结构
- **GIVEN** Config 包含 settings（动态 map）、current_provider（字符串）、providers（动态 map）
- **WHEN** 序列化/反序列化 JSON
- **THEN** 应当正确处理任意字段

#### Scenario: 动态 settings 字段
- **GIVEN** settings 可能包含 permissions、alwaysThinkingEnabled、env 等字段
- **AND** Claude 未来可能添加新字段
- **WHEN** 处理 settings
- **THEN** 系统应当能够处理未知字段而不出错

#### Scenario: 动态 provider 配置
- **GIVEN** provider 配置可能包含 env 或其他任意字段
- **WHEN** 反序列化提供商配置
- **THEN** 应当保留所有字段

### Requirement: 配置加载

系统 SHALL 提供从文件加载配置的功能。

#### Scenario: 加载有效配置
- **GIVEN** 存在有效的 ccc.json 文件
- **WHEN** 调用 `Load()` 函数
- **THEN** 应当返回 Config 对象
- **AND** error 应当为 nil

#### Scenario: 配置文件不存在
- **GIVEN** ccc.json 文件不存在
- **WHEN** 调用 `Load()` 函数
- **THEN** 应当返回错误
- **AND** 错误信息应当包含 "failed to read config file"

#### Scenario: 配置格式错误
- **GIVEN** ccc.json 包含无效的 JSON
- **WHEN** 调用 `Load()` 函数
- **THEN** 应当返回错误
- **AND** 错误信息应当包含 "failed to parse config file"

### Requirement: 配置保存

系统 SHALL 提供将配置保存到文件的功能。

#### Scenario: 保存配置
- **GIVEN** 一个有效的 Config 对象
- **WHEN** 调用 `Save()` 函数
- **THEN** 应当创建或更新 ccc.json 文件
- **AND** 文件内容应当是格式化的 JSON（缩进 2 个空格）
- **AND** 文件权限应当为 0644

#### Scenario: 目录不存在
- **GIVEN** 配置目录不存在
- **WHEN** 调用 `Save()` 函数
- **THEN** 应当自动创建目录
- **AND** 目录权限应当为 0755

### Requirement: 深度合并

系统 SHALL 能够深度合并基础 settings 和 provider 配置，provider 配置优先。

#### Scenario: 深度合并 env 字段
- **GIVEN** base.settings.env 为 `{"API_TIMEOUT": "30000"}`
- **AND** provider.env 为 `{"BASE_URL": "https://api.kimi.com", "API_TIMEOUT": "60000"}`
- **WHEN** 执行深度合并
- **THEN** 结果 env 应当为 `{"BASE_URL": "https://api.kimi.com", "API_TIMEOUT": "60000"}`

#### Scenario: 深度合并嵌套对象
- **GIVEN** base.settings.permissions 为 `{"allow": ["Edit"]}`
- **AND** provider.permissions 为 `{"allow": ["WebSearch"]}`
- **WHEN** 执行深度合并
- **THEN** 结果 permissions 应当被 provider 配置覆盖

#### Scenario: 保留未知字段
- **GIVEN** base.settings 包含未知字段 `{"newFeature": true}`
- **AND** provider 不包含此字段
- **WHEN** 执行深度合并
- **THEN** 结果应当保留 `{"newFeature": true}`

### Requirement: 配置路径解析

系统 SHALL 提供获取配置文件路径的函数。

#### Scenario: 默认配置路径
- **WHEN** 未设置 CCC_CONFIG_DIR 环境变量
- **THEN** 配置路径应当为 `~/.claude/ccc.json`

#### Scenario: 自定义配置路径
- **GIVEN** CCC_CONFIG_DIR 设置为 `/custom/path`
- **WHEN** 获取配置路径
- **THEN** 配置路径应当为 `/custom/path/ccc.json`

### Requirement: Settings 文件保存

系统 SHALL 能够保存合并后的 settings 到 provider 专属文件。

#### Scenario: 保存 provider settings
- **GIVEN** providerName 为 "kimi"
- **AND** 合并后的 settings
- **WHEN** 调用 `SaveSettings(settings, providerName)`
- **THEN** 应当创建 `~/.claude/settings-kimi.json`
- **AND** 文件内容应当是完整的 Claude settings 格式

### Requirement: 值提取辅助函数

系统 SHALL 提供从动态 settings 中提取特定值的辅助函数。

#### Scenario: 提取 ANTHROPIC_AUTH_TOKEN
- **GIVEN** settings.env 包含 ANTHROPIC_AUTH_TOKEN
- **WHEN** 调用 `GetAuthToken(settings)`
- **THEN** 应当返回 token 值
- **AND** 如果不存在，返回占位符

#### Scenario: 提取嵌套字段
- **GIVEN** settings 包含嵌套结构
- **WHEN** 调用提取函数
- **THEN** 应当能够安全地遍历嵌套 map
- **AND** 如果路径不存在，返回默认值
