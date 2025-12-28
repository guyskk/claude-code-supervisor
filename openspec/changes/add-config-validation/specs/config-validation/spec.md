# config-validation Specification

## Purpose

配置验证功能允许用户验证 ccc 配置文件和提供商配置的有效性，包括配置格式、环境变量完整性和 API 连通性。

## ADDED Requirements

### Requirement: 单个提供商验证

系统 SHALL 能够验证指定提供商的配置是否有效。

#### Scenario: 验证有效的提供商配置
- **GIVEN** ccc.json 中配置了 `kimi` 提供商
- **AND** 配置包含完整的 `ANTHROPIC_BASE_URL` 和 `ANTHROPIC_AUTH_TOKEN`
- **AND** API 连接测试成功
- **WHEN** 用户执行 `ccc validate kimi`
- **THEN** 应当输出 "Valid: kimi"
- **AND** 显示配置摘要（base URL、模型名称等）
- **AND** 返回退出码 0

#### Scenario: 验证缺失环境变量的提供商
- **GIVEN** ccc.json 中配置了 `glm` 提供商
- **AND** 配置缺少 `ANTHROPIC_AUTH_TOKEN`
- **WHEN** 用户执行 `ccc validate glm`
- **THEN** 应当输出 "Invalid: glm"
- **AND** 显示错误 "Missing required environment variable: ANTHROPIC_AUTH_TOKEN"
- **AND** 返回非零退出码

#### Scenario: 验证无效的 Base URL
- **GIVEN** ccc.json 中配置了 `test` 提供商
- **AND** `ANTHROPIC_BASE_URL` 格式无效（如 "not-a-url"）
- **WHEN** 用户执行 `ccc validate test`
- **THEN** 应当输出 "Invalid: test"
- **AND** 显示错误 "Invalid Base URL format"
- **AND** 返回非零退出码

#### Scenario: 验证不存在的提供商
- **GIVEN** ccc.json 中不存在 `unknown` 提供商
- **WHEN** 用户执行 `ccc validate unknown`
- **THEN** 应当输出错误 "Provider 'unknown' not found in configuration"
- **AND** 返回非零退出码

#### Scenario: API 连通性测试失败
- **GIVEN** ccc.json 中配置了 `offline` 提供商
- **AND** 配置格式正确但 API 服务器无法访问
- **WHEN** 用户执行 `ccc validate offline`
- **THEN** 应当输出 "Warning: offline - API connection: failed"
- **AND** 显示连接错误详情
- **AND** 返回零退出码（配置有效但 API 不可用）

### Requirement: 所有提供商验证

系统 SHALL 能够一次性验证所有提供商的配置。

#### Scenario: 验证所有提供商 - 全部有效
- **GIVEN** ccc.json 中配置了 `kimi`、`glm`、`m2` 三个提供商
- **AND** 所有提供商配置都有效
- **WHEN** 用户执行 `ccc validate --all`
- **THEN** 应当输出汇总信息 "Validating 3 provider(s)..."
- **AND** 显示每个提供商的验证结果
- **AND** 最后输出 "All providers valid"
- **AND** 返回退出码 0

#### Scenario: 验证所有提供商 - 部分无效
- **GIVEN** ccc.json 中配置了 `kimi`、`glm`、`broken` 三个提供商
- **AND** `broken` 配置无效
- **WHEN** 用户执行 `ccc validate --all`
- **THEN** 应当显示每个提供商的验证结果
- **AND** 为 `broken` 显示具体错误
- **AND** 最后输出 "1/3 providers invalid"
- **AND** 返回非零退出码

#### Scenario: 验证所有提供商 - 无配置
- **GIVEN** ccc.json 中 `providers` 字段为空
- **WHEN** 用户执行 `ccc validate --all`
- **THEN** 应当输出 "No providers configured"
- **AND** 返回退出码 0

### Requirement: 当前提供商验证

系统 SHALL 支持验证当前激活的提供商。

#### Scenario: 验证当前提供商
- **GIVEN** ccc.json 中 `current_provider` 设置为 `kimi`
- **AND** `kimi` 配置有效
- **WHEN** 用户执行 `ccc validate`（不带参数）
- **THEN** 应当验证当前提供商 `kimi`
- **AND** 显示验证结果

#### Scenario: 无当前提供商时验证
- **GIVEN** ccc.json 中 `current_provider` 为空
- **AND** 配置了多个提供商
- **WHEN** 用户执行 `ccc validate`（不带参数）
- **THEN** 应当输出 "No current provider set"
- **AND** 显示可用提供商列表
- **AND** 返回非零退出码

### Requirement: 跳过 API 测试

系统 SHALL 支持跳过 API 连通性测试，仅验证配置格式。

#### Scenario: 跳过 API 测试验证有效配置
- **GIVEN** ccc.json 中配置了 `kimi` 提供商
- **AND** 配置格式有效
- **WHEN** 用户执行 `ccc validate kimi --no-api-test`
- **THEN** 应当显示验证结果
- **AND** 不显示 "API connection" 信息
- **AND** 返回退出码 0

#### Scenario: 跳过 API 测试验证无效配置
- **GIVEN** ccc.json 中配置了 `broken` 提供商
- **AND** 配置缺少必需字段
- **WHEN** 用户执行 `ccc validate broken --no-api-test`
- **THEN** 应当显示配置错误
- **AND** 不执行 API 连通性测试
- **AND** 返回非零退出码

### Requirement: 配置格式验证

系统 SHALL 在验证提供商之前检查配置文件的格式和完整性。

#### Scenario: 配置文件格式错误
- **GIVEN** ccc.json 不是有效的 JSON 格式
- **WHEN** 用户执行 `ccc validate`
- **THEN** 应当输出配置加载错误
- **AND** 显示具体的 JSON 解析错误
- **AND** 返回非零退出码

### Requirement: 环境变量检查

系统 SHALL 检查提供商配置中的关键环境变量是否正确设置。

#### Scenario: 检查必需的环境变量
- **WHEN** 验证提供商配置
- **THEN** 应当检查以下变量是否存在：
  - `ANTHROPIC_BASE_URL`
  - `ANTHROPIC_AUTH_TOKEN`
- **AND** 如果缺失，应当报告错误

#### Scenario: 验证环境变量格式
- **WHEN** 验证提供商配置
- **THEN** 应当验证以下格式：
  - `ANTHROPIC_BASE_URL` 必须是有效的 HTTP(S) URL
  - `ANTHROPIC_BASE_URL` 必须包含 scheme (http/https)
  - `ANTHROPIC_BASE_URL` 必须包含 host
  - `ANTHROPIC_AUTH_TOKEN` 必须非空
- **AND** 如果格式错误，应当报告具体问题

### Requirement: API 连通性测试

系统 SHALL 尝试连接到提供商的 API 端点以验证配置是否真正可用。

#### Scenario: API 连接成功
- **GIVEN** 有效提供商配置
- **WHEN** 执行 API 连通性测试
- **THEN** 应当向 Base URL 发送简单请求
- **AND** 根据响应判断连接是否成功
- **AND** 连接成功时显示 "API connection: OK"

#### Scenario: API 连接失败 - 网络错误
- **GIVEN** 提供商 Base URL 无法访问
- **WHEN** 执行 API 连通性测试
- **THEN** 应当捕获网络错误
- **AND** 显示警告而不是错误（配置可能正确但网络不可用）
- **AND** 显示 "API connection: failed: {error}"

#### Scenario: API 连接失败 - 认证错误
- **GIVEN** 提供商配置了无效的 API Key
- **WHEN** 执行 API 连通性测试
- **THEN** 应当检测到 401/403 认证错误
- **AND** 将认证错误视为连接成功（网络可达，配置可能有问题）

### Requirement: 详细验证报告

系统 SHALL 提供详细的验证报告，帮助用户快速定位和修复配置问题。

#### Scenario: 显示配置摘要
- **GIVEN** 有效提供商配置
- **WHEN** 验证成功
- **THEN** 应当显示以下信息：
  - Base URL
  - 模型名称（如果配置了）
  - API 连接状态（如果测试了）

#### Scenario: 彩色输出状态
- **GIVEN** 终端支持彩色输出
- **WHEN** 显示验证结果
- **THEN** 有效配置使用绿色显示
- **AND** 无效配置使用红色显示
- **AND** 警告状态使用黄色显示
