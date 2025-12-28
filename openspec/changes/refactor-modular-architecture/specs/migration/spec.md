# migration 规范变更

本规范迁移现有的 `config-migration` 功能到独立的 `migration` 包。

## MODIFIED Requirements

### Requirement: 旧配置检测

系统 SHALL 能够检测 `~/.claude/settings.json` 文件是否存在，以便决定是否提供配置迁移选项。

#### Scenario: settings.json 存在
- **WHEN** 用户运行 `ccc` 或 `ccc <provider>` 命令
- **AND** `~/.claude/ccc.json` 文件不存在
- **AND** `~/.claude/settings.json` 文件存在
- **THEN** `CheckExisting()` 函数应当返回 true

#### Scenario: settings.json 不存在
- **WHEN** 用户运行 `ccc` 或 `ccc <provider>` 命令
- **AND** `~/.claude/ccc.json` 文件不存在
- **AND** `~/.claude/settings.json` 文件不存在
- **THEN** `CheckExisting()` 函数应当返回 false

### Requirement: 用户交互式迁移确认

当检测到旧配置存在时，系统 SHALL 提示用户并等待确认，而不是自动执行迁移。

#### Scenario: 用户接受迁移
- **WHEN** `PromptUser()` 函数被调用
- **AND** 用户输入 "y" 或 "yes"（不区分大小写）
- **THEN** 应当返回 true，表示用户同意迁移

#### Scenario: 用户拒绝迁移
- **WHEN** `PromptUser()` 函数被调用
- **AND** 用户输入 "n" 或 "no" 或其他任意字符
- **THEN** 应当返回 false，表示用户拒绝迁移

#### Scenario: 输入读取失败
- **WHEN** `PromptUser()` 函数尝试读取用户输入
- **AND** 读取过程中发生错误（如 stdin 关闭）
- **THEN** 应当返回 false，默认拒绝迁移

### Requirement: 配置迁移执行

系统 SHALL 能够从 `settings.json` 迁移配置到 `ccc.json`，正确拆分 `env` 字段和其他配置。

#### Scenario: 标准迁移 - 包含 env 字段
- **GIVEN** settings.json 内容包含 env 字段
- **WHEN** 调用 `MigrateFromSettings()` 函数
- **THEN** 应当创建 ccc.json
- **AND** env 字段应当移到 providers.default.env
- **AND** 其他字段保留在 settings 中
- **AND** settings.json 应当保持不变

#### Scenario: 迁移 - 不包含 env 字段
- **GIVEN** settings.json 内容不包含 env 字段
- **WHEN** 调用 `MigrateFromSettings()` 函数
- **THEN** 应当创建 ccc.json
- **AND** providers.default 应当为空对象

#### Scenario: settings.json 读取失败
- **GIVEN** settings.json 文件不存在
- **WHEN** 调用 `MigrateFromSettings()` 函数
- **THEN** 应当返回错误
- **AND** 错误信息应当包含 "failed to read settings file"

#### Scenario: settings.json 格式错误
- **GIVEN** settings.json 内容不是有效的 JSON
- **WHEN** 调用 `MigrateFromSettings()` 函数
- **THEN** 应当返回错误
- **AND** 错误信息应当包含 "failed to parse settings file"
