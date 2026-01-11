# cli 规范变更

## ADDED Requirements

### Requirement: 命令行参数解析

系统 SHALL 能够解析命令行参数，识别命令、选项和参数。

#### Scenario: 解析 --help
- **WHEN** 用户执行 `ccc --help` 或 `ccc -h`
- **THEN** 应当显示帮助信息
- **AND** 退出码应当为 0

#### Scenario: 解析 --version
- **WHEN** 用户执行 `ccc --version` 或 `ccc -v`
- **THEN** 应当显示版本信息
- **AND** 格式应当为 "claude-code-supervisor version {version} (built at {time})"
- **AND** 退出码应当为 0

#### Scenario: 解析提供商名称
- **GIVEN** 配置中存在提供商 "kimi"
- **WHEN** 用户执行 `ccc kimi`
- **THEN** 应当识别 "kimi" 为提供商名称
- **AND** 应当切换到 kimi 提供商

#### Scenario: 解析附加参数
- **GIVEN** 用户执行 `ccc kimi /path/to/project`
- **THEN** 应当识别 "kimi" 为提供商名称
- **AND** 应当将 "/path/to/project" 作为参数传递给 claude

#### Scenario: 无参数使用当前提供商
- **GIVEN** current_provider 为 "glm"
- **WHEN** 用户执行 `ccc`
- **THEN** 应当使用 glm 提供商
- **AND** 不切换提供商

### Requirement: 帮助信息显示

系统 SHALL 提供清晰的帮助信息，包括用法、命令和可用提供商列表。

#### Scenario: 显示帮助
- **WHEN** 用户请求帮助
- **THEN** 应当显示用法说明
- **AND** 应当显示可用命令列表
- **AND** 应当显示环境变量说明
- **AND** 如果配置可用，应当显示提供商列表

#### Scenario: 配置加载失败时的帮助
- **GIVEN** 配置文件不存在或格式错误
- **WHEN** 用户请求帮助
- **THEN** 应当显示帮助信息
- **AND** 应当显示配置文件路径
- **AND** 应当显示错误信息（截断到 40 字符）

### Requirement: 版本信息显示

系统 SHALL 能够显示版本和构建时间信息。

#### Scenario: 显示开发版本
- **GIVEN** 版本未设置（开发构建）
- **WHEN** 用户请求版本信息
- **THEN** 应当显示 "claude-code-supervisor version dev (built at unknown)"

#### Scenario: 显示发布版本
- **GIVEN** 版本为 "v0.1.2"，构建时间为 "2024-01-15T10:30:00Z"
- **WHEN** 用户请求版本信息
- **THEN** 应当显示 "claude-code-supervisor version v0.1.2 (built at 2024-01-15T10:30:00Z)"
