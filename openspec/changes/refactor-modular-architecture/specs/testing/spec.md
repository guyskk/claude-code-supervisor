# testing 规范变更

## ADDED Requirements

### Requirement: 单元测试覆盖

系统的所有核心功能 SHALL 有对应的单元测试。

#### Scenario: config 包测试
- **GIVEN** config 包中的所有导出函数和类型
- **WHEN** 运行 `go test ./config/...`
- **THEN** 应当测试 Load、Save、路径解析等所有功能
- **AND** 代码覆盖率应当 ≥80%

#### Scenario: provider 包测试
- **GIVEN** provider 包中的所有导出函数
- **WHEN** 运行 `go test ./provider/...`
- **THEN** 应当测试 Switch、Merge、GetAuthToken 等所有功能
- **AND** 代码覆盖率应当 ≥80%

#### Scenario: migration 包测试
- **GIVEN** migration 包中的所有导出函数
- **WHEN** 运行 `go test ./migration/...`
- **THEN** 应当测试 CheckExisting、PromptUser、MigrateFromSettings 等所有功能
- **AND** 代码覆盖率应当 ≥90%（已有基础）

#### Scenario: cli 包测试
- **GIVEN** cli 包中的命令解析逻辑
- **WHEN** 运行 `go test ./cli/...`
- **THEN** 应当测试各种命令行参数组合
- **AND** 代码覆盖率应当 ≥80%

### Requirement: 集成测试

系统 SHALL 有集成测试覆盖完整的用户使用流程。

#### Scenario: 首次运行迁移流程
- **GIVEN** 一个临时目录作为配置目录
- **AND** 目录中有 settings.json
- **WHEN** 模拟用户运行 ccc 并接受迁移
- **THEN** 应当创建 ccc.json
- **AND** 应当能够切换提供商
- **AND** 应当能够生成配置文件

#### Scenario: 提供商切换流程
- **GIVEN** 一个有效的 ccc.json 配置
- **WHEN** 执行 `ccc kimi` 命令
- **THEN** 应当创建 settings-kimi.json
- **AND** 配置应当正确合并

### Requirement: 测试隔离

所有测试 SHALL 使用独立的临时目录，不影响用户的真实配置。

#### Scenario: 使用临时目录
- **GIVEN** 任意测试用例
- **WHEN** 测试执行
- **THEN** 应当使用 `t.TempDir()` 或类似机制
- **AND** 不应当访问 `~/.claude/`

#### Scenario: 测试间独立
- **GIVEN** 多个测试用例
- **WHEN** 并发运行测试
- **THEN** 测试之间应当不共享状态
- **AND** 测试顺序不应当影响结果

### Requirement: 竞态条件检测

系统 SHALL 通过竞态检测器验证，不包含数据竞争。

#### Scenario: 通过竞态检测
- **WHEN** 运行 `go test -race ./...`
- **THEN** 不应当报告任何数据竞争
- **AND** 所有测试应当通过

### Requirement: 测试辅助工具

系统 SHALL 提供测试辅助函数，简化测试编写。

#### Scenario: 临时配置目录
- **WHEN** 测试需要配置目录
- **THEN** 应当提供 `testutil.TempConfigDir(t)` 辅助函数
- **AND** 返回的目录应当在测试结束后自动清理

#### Scenario: 配置文件读写
- **WHEN** 测试需要创建或读取配置
- **THEN** 应当提供 `testutil.WriteConfig(t, dir, cfg)` 函数
- **AND** 应当提供 `testutil.ReadConfig(t, path)` 函数
