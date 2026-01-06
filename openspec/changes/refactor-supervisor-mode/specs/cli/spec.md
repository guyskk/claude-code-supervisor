# cli Specification Delta

## ADDED Requirements

### Requirement: Supervisor 配置解析

CLI SHALL 支持从 `ccc.json` 读取 Supervisor 配置。

#### Scenario: 读取 Supervisor 配置
- **GIVEN** `ccc.json` 包含 `supervisor` 段
- **WHEN** 加载配置
- **THEN** 应当解析 `supervisor.enabled`, `supervisor.max_iterations` 等字段
- **AND** 应当应用默认值到未设置的字段

#### Scenario: 配置优先级
- **GIVEN** `ccc.json` 中 `supervisor.max_iterations` 为 15
- **AND** 环境变量 `CCC_SUPERVISOR_MAX_ITERATIONS` 为 25
- **WHEN** 启动 Supervisor Mode
- **THEN** 应当使用环境变量值 25

## MODIFIED Requirements

### Requirement: Claude 执行

CLI SHALL 使用 syscall.Exec 替换当前进程为 claude。

#### Scenario: Supervisor Mode 启动（更新）
- **GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置
- **AND** 配置中 `supervisor.max_iterations` 为 20
- **WHEN** 用户执行 `ccc <provider>`
- **THEN** 应当生成带 Stop hook 的 `settings.json`
- **AND** hook 命令应当为 `ccc supervisor-hook`（无额外参数）
- **AND** 应当设置 `CCC_SUPERVISOR_ID` 环境变量为新的 UUID
- **AND** 应当使用 syscall.Exec 启动 claude
- **AND** 应当在日志文件中显示 session 日志路径

#### Scenario: 显示配置信息（新增）
- **GIVEN** Supervisor Mode 启用
- **AND** 配置中 `supervisor.max_iterations` 为 20
- **WHEN** 启动 claude
- **THEN** 应当在 stderr 输出配置摘要
- **AND** 输出格式应当为：
  ```
  [Supervisor Mode]
  Max iterations: 20
  Timeout: 600s
  Log: ~/.claude/ccc/supervisor-{id}.log
  ```

### Requirement: 环境变量支持

CLI SHALL 支持通过环境变量控制行为。

#### Scenario: 环境变量列表（更新）
- **WHEN** 查询支持的环境变量
- **THEN** 应当支持：
  - `CCC_CONFIG_DIR`: 配置目录
  - `CCC_SUPERVISOR`: 启用 Supervisor Mode ("1"=启用)
  - `CCC_SUPERVISOR_MAX_ITERATIONS`: 覆盖最大迭代次数
  - `CCC_SUPERVISOR_TIMEOUT`: 覆盖超时秒数
  - `CCC_SUPERVISOR_LOG_LEVEL`: 覆盖日志级别
  - `CCC_SUPERVISOR_ID`: 内部使用（session ID）
  - `CCC_SUPERVISOR_HOOK`: 内部使用（防止死循环）

### Requirement: 帮助信息

CLI SHALL 显示帮助信息。

#### Scenario: 帮助包含 Supervisor 配置说明（新增）
- **WHEN** 用户执行 `ccc --help`
- **THEN** 帮助信息应当包含：
  - Supervisor 配置文件格式说明
  - 环境变量覆盖说明
  - 配置示例

## REMOVED Requirements

无移除的需求。
