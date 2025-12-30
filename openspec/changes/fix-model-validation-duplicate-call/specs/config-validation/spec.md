# config-validation Specification Delta

## Purpose

简化 API 验证逻辑，移除不必要的回退机制，避免重复调用 `/v1/models` 端点。

## MODIFIED Requirements

### Requirement: API 连通性测试

系统 SHALL 尝试连接到提供商的 API 端点以验证配置是否真正可用。

#### Scenario: 无模型配置时验证（MODIFIED）
- **GIVEN** 提供商没有配置 `ANTHROPIC_MODEL`
- **WHEN** 执行 API 连通性测试
- **THEN** 系统 SHALL：
  1. 调用 `/v1/models` 端点获取可用模型列表
  2. 如果成功，直接返回 "ok"（token 有效，配置正确）
  3. 如果失败，返回错误信息
- **AND** 系统 SHALL NOT 调用 `/v1/messages` 端点

#### Scenario: 有模型配置时验证（MODIFIED）
- **GIVEN** 提供商配置了 `ANTHROPIC_MODEL`
- **WHEN** 执行 API 连通性测试
- **THEN** 系统 SHALL：
  1. 用配置的模型调用 `/v1/messages` 端点
  2. 如果成功，返回 "ok"
  3. 如果失败，返回错误信息
- **AND** 系统 SHALL NOT 回退调用 `/v1/models` 端点

#### Scenario: API 连接失败 - 不回退（ADDED）
- **GIVEN** 提供商配置了 `ANTHROPIC_MODEL`
- **AND** 用配置的模型调用 `/v1/messages` 失败
- **WHEN** 执行 API 连通性测试
- **THEN** 系统 SHALL 直接返回错误信息
- **AND** 系统 SHALL NOT 尝试回退到其他端点

#### Scenario: API 连接失败 - 无模型配置（ADDED）
- **GIVEN** 提供商没有配置 `ANTHROPIC_MODEL`
- **AND** 调用 `/v1/models` 失败
- **WHEN** 执行 API 连通性测试
- **THEN** 系统 SHALL 返回错误信息
- **AND** 系统 SHALL NOT 尝试其他验证方式

#### Scenario: API 连接成功
- **GIVEN** 有效提供商配置
- **WHEN** 执行 API 连通性测试
- **THEN** 应当向 API 端点发送请求
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
- **AND** 将认证错误视为连接失败

