## MODIFIED Requirements

### Requirement: Supervisor 配置

Supervisor 配置 SHALL 只包含迭代次数和超时设置，不包含启用状态。

#### Scenario: SupervisorConfig 结构（更新）
- **GIVEN** 系统定义 `SupervisorConfig` 结构体
- **THEN** 应当包含以下字段：
  - `MaxIterations int`: 最大迭代次数（默认 20）
  - `TimeoutSeconds int`: 每次 supervisor 调用的超时秒数（默认 600）
- **AND** 不应当包含 `Enabled` 字段

#### Scenario: 加载 Supervisor 配置（更新）
- **GIVEN** `ccc.json` 包含 `supervisor` 配置
- **AND** 配置为 `{"max_iterations": 15, "timeout_seconds": 300}`
- **WHEN** 系统加载配置
- **THEN** `MaxIterations` 应当为 15
- **AND** `TimeoutSeconds` 应当为 300
- **AND** 不应当读取 `enabled` 字段

#### Scenario: 默认 Supervisor 配置（更新）
- **GIVEN** `ccc.json` 不包含 `supervisor` 配置
- **WHEN** 系统加载默认配置
- **THEN** `MaxIterations` 应当为 20
- **AND** `TimeoutSeconds` 应当为 600
- **AND** 不应当包含 `Enabled` 字段

#### Scenario: 不再读取 CCC_SUPERVISOR 环境变量
- **GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置
- **WHEN** 系统加载 Supervisor 配置
- **THEN** 不应当读取该环境变量
- **AND** 配置的 `Enabled` 字段应当不存在

## REMOVED Requirements

### Requirement: Supervisor 配置（包含 Enabled 字段）

**Reason**: Supervisor 启用状态改用 state 文件的 `Enabled` 字段控制，不再需要在配置文件中设置。

**Migration**: 用户改用 `/supervisor` slash command 启用 Supervisor Mode，无需修改配置文件。

- ~~**GIVEN** `SupervisorConfig` 结构体~~
- ~~**THEN** 应当包含 `Enabled bool` 字段~~

- ~~**GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置~~
- ~~**WHEN** 系统加载配置~~
- ~~**THEN** `Enabled` 应当被覆盖为 `true`~~

### Requirement: 配置文件结构（包含 Supervisor）

**Reason**: `Config` 结构体的 `Supervisor` 字段被移除或简化，不再包含 `Enabled`。

**Migration**: 用户无需迁移，`ccc.json` 中的 `supervisor.enabled` 字段将被忽略。

- ~~**GIVEN** `ccc.json` 包含配置~~
- ~~**WHEN** 系统解析配置~~
- ~~**THEN** `Supervisor.Enabled` 应当被正确解析~~
