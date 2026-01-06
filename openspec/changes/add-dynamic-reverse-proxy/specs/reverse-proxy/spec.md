# reverse-proxy Specification

## Purpose

定义动态反向代理功能的行为规范。反向代理允许在本地运行一个 HTTP 代理服务器，Claude Code 通过该代理访问 API，支持动态切换提供商而无需重启。

## ADDED Requirements

### Requirement: 代理配置

系统 SHALL 支持在 ccc.json 中配置反向代理。

#### Scenario: 代理配置结构
- **GIVEN** ccc.json 包含 proxy 配置
- **WHEN** 加载配置
- **THEN** 应当解析以下字段：
  - `enabled`: bool - 是否启用代理
  - `listen`: string - 监听地址 (默认 "127.0.0.1:8080")
  - `default_provider`: string - 默认提供商名称
  - `api_key_header`: string - API Key 请求头名称 (可选)

#### Scenario: 禁用代理模式
- **GIVEN** proxy.enabled 为 false 或未配置
- **WHEN** 启动 ccc
- **THEN** 应当使用传统模式（不启动代理）

#### Scenario: 启用代理模式
- **GIVEN** proxy.enabled 为 true
- **WHEN** 启动 ccc
- **THEN** 应当启动反向代理服务器
- **AND** 应当在后台运行
- **AND** 应当输出代理地址到 stderr

### Requirement: 代理服务器启动

系统 SHALL 能够启动 HTTP 反向代理服务器。

#### Scenario: 成功启动代理
- **GIVEN** 配置的端口可用
- **WHEN** 启动代理服务器
- **THEN** 应当监听在配置的地址
- **AND** 应当输出 "Proxy listening on http://127.0.0.1:8080"
- **AND** error 应当为 nil

#### Scenario: 端口被占用
- **GIVEN** 配置的端口已被占用
- **WHEN** 启动代理服务器
- **THEN** 应当自动选择可用端口
- **AND** 应当输出实际端口地址
- **AND** error 应当为 nil

#### Scenario: IPv4 回环地址
- **GIVEN** proxy.listen 未配置
- **WHEN** 启动代理服务器
- **THEN** 应当监听在 127.0.0.1:8080

### Requirement: Settings 修改

系统 SHALL 自动修改 settings.json 使 Claude Code 通过代理访问 API。

#### Scenario: 修改 BASE_URL
- **GIVEN** 启用代理模式
- **AND** 代理监听在 127.0.0.1:8080
- **WHEN** 生成 settings.json
- **THEN** ANTHROPIC_BASE_URL 应当为 "http://127.0.0.1:8080"
- **AND** 其他配置保持不变

#### Scenario: 保留原始提供商信息
- **GIVEN** 启用代理模式
- **AND** 当前提供商为 kimi
- **WHEN** 启动代理
- **THEN** 代理内部应当记录原始提供商为 kimi
- **AND** API 请求应当转发到 kimi 的实际地址

### Requirement: API 请求转发

系统 SHALL 将 Claude Code 的 API 请求转发到当前提供商。

#### Scenario: 转发 POST /v1/messages
- **GIVEN** Claude Code 发送 POST 请求到 http://127.0.0.1:8080/v1/messages
- **AND** 当前提供商为 kimi
- **WHEN** 代理接收请求
- **THEN** 应当转发到 https://api.moonshot.cn/anthropic/v1/messages
- **AND** 应当添加提供商的 Authorization 头
- **AND** 应当返回提供商的响应

#### Scenario: 转发 GET /v1/models
- **GIVEN** Claude Code 发送 GET 请求到 http://127.0.0.1:8080/v1/models
- **AND** 当前提供商为 glm
- **WHEN** 代理接收请求
- **THEN** 应当转发到 https://open.bigmodel.cn/api/anthropic/v1/models

#### Scenario: 保留请求头
- **GIVEN** Claude Code 发送请求包含 anthropic-version 头
- **WHEN** 代理转发请求
- **THEN** 应当保留所有原始请求头
- **AND** 应当添加/覆盖 Authorization 头

#### Scenario: 保留响应头
- **GIVEN** 提供商返回响应包含特定的 CORS 头
- **WHEN** 代理返回响应
- **THEN** 应当保留所有原始响应头
- **AND** 应当保留响应状态码

### Requirement: 管理接口

系统 SHALL 提供 REST API 用于管理代理状态。

#### Scenario: 获取提供商列表
- **GIVEN** 代理服务器正在运行
- **WHEN** GET /api/providers
- **THEN** 应当返回 200 状态码
- **AND** 响应体包含所有可用提供商

#### Scenario: 获取当前提供商
- **GIVEN** 当前提供商为 kimi
- **WHEN** GET /api/provider/current
- **THEN** 应当返回 200 状态码
- **AND** 响应体包含 kimi 的信息

#### Scenario: 切换提供商
- **GIVEN** 当前提供商为 kimi
- **WHEN** PUT /api/provider/current with {"name": "glm"}
- **THEN** 应当返回 200 状态码
- **AND** 当前提供商应当变为 glm
- **AND** 后续 API 请求应当转发到 glm

#### Scenario: 切换到不存在的提供商
- **WHEN** PUT /api/provider/current with {"name": "nonexistent"}
- **THEN** 应当返回 400 状态码
- **AND** 响应体包含错误信息
- **AND** 当前提供商应当保持不变

#### Scenario: 健康检查
- **GIVEN** 代理服务器正在运行
- **WHEN** GET /api/health
- **THEN** 应当返回 200 状态码
- **AND** 响应体包含 "status": "ok"
- **AND** 响应体包含当前提供商名称

### Requirement: 错误处理

系统 SHALL 正确处理代理过程中的各种错误。

#### Scenario: 提供商不可达
- **GIVEN** 当前提供商 API 不可达
- **WHEN** Claude Code 发送请求
- **THEN** 应当返回 502 Bad Gateway
- **AND** 响应体包含错误详情

#### Scenario: API Key 无效
- **GIVEN** 提供商返回 401 Unauthorized
- **WHEN** 代理接收响应
- **THEN** 应当直接转发 401 响应
- **AND** 不修改响应体

#### Scenario: 代理内部错误
- **WHEN** 代理处理请求时发生内部错误
- **THEN** 应当返回 500 Internal Server Error
- **AND** 响应体包含错误信息

### Requirement: 并发安全

系统 SHALL 支持并发请求和并发切换提供商。

#### Scenario: 并发 API 请求
- **GIVEN** 多个 Claude Code 请求同时到达
- **WHEN** 代理处理请求
- **THEN** 应当正确处理所有请求
- **AND** 不应当有请求丢失

#### Scenario: 并发切换提供商
- **GIVEN** 当前有正在处理的请求
- **WHEN** 切换提供商
- **THEN** 新请求应当使用新提供商
- **AND** 正在进行的请求应当完成（不被中断）

### Requirement: 生命周期管理

系统 SHALL 正确管理代理服务器的生命周期。

#### Scenario: 随 ccc 启动
- **GIVEN** ccc 启动时启用代理模式
- **WHEN** ccc 执行
- **THEN** 代理应当在 Claude Code 启动前就绪
- **AND** ccc 退出时代理应当停止

#### Scenario: 优雅关闭
- **GIVEN** 代理服务器正在运行
- **AND** 有正在处理的请求
- **WHEN** 收到关闭信号
- **THEN** 应当等待当前请求完成
- **AND** 应当拒绝新请求

