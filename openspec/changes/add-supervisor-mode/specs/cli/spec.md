# cli Specification Delta

## ADDED Requirements

### Requirement: --supervisor 参数

系统 SHALL 支持 `--supervisor` 参数，启用 Supervisor 模式，实现 Agent 执行与 Supervisor 检查的自动循环。

#### Scenario: 启用 supervisor 模式
- **GIVEN** 配置中存在提供商 "kimi"
- **WHEN** 用户执行 `ccc --supervisor`
- **THEN** 应当显示 "Supervisor mode enabled"
- **AND** 应当进入 Agent-Supervisor 循环
- **AND** 用户视角仍然是直接与 Agent 交互

#### Scenario: supervisor 模式切换提供商
- **WHEN** 用户执行 `ccc --supervisor kimi`
- **THEN** 应当切换到 kimi 提供商
- **AND** 应当启用 Supervisor 模式

#### Scenario: supervisor 模式传递参数
- **WHEN** 用户执行 `ccc --supervisor kimi /path/to/project`
- **THEN** 应当切换到 kimi 提供商
- **AND** 应当启用 Supervisor 模式
- **AND** 应当将 "/path/to/project" 传递给 claude

#### Scenario: supervisor 模式帮助信息
- **WHEN** 用户执行 `ccc --help`
- **THEN** 帮助信息应当包含 `--supervisor` 参数说明
- **AND** 应当说明 Supervisor 模式的功能
