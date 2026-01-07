## MODIFIED Requirements

### Requirement: Provider Configuration
系统 SHALL 支持通过配置文件定义提供商，提供商配置包括环境变量。

#### Scenario: 切换提供商时不将 env 写入 settings.json
- **WHEN** 用户执行 `ccc <provider>` 切换提供商
- **THEN** settings.json 中不包含 `env` 字段
- **AND** 环境变量通过进程环境传递给 claude 子进程

#### Scenario: 环境变量合并
- **WHEN** 切换提供商时，settings 中有 `env` 且 provider 中也有 `env`
- **THEN** provider 的 env 覆盖 settings 中的同名变量
- **AND** 最终环境变量传递给 claude 子进程

## ADDED Requirements

### Requirement: 环境变量传递
系统 SHALL 通过环境变量将提供商配置传递给 claude 子进程。

#### Scenario: 启动 claude 时传递环境变量
- **WHEN** 执行 `ccc <provider>` 启动 claude
- **THEN** 合并后的环境变量（settings.env + provider.env）传递给子进程
- **AND** claude 子进程继承父进程的环境变量

#### Scenario: 支持环境变量展开
- **WHEN** provider 配置中的 env 值包含 `${VAR}` 格式
- **THEN** 自动展开为实际的环境变量值
