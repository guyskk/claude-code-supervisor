# web-service Specification

## Purpose

定义 Web 服务功能的行为规范。Web 服务提供 HTTP API 用于管理 Claude Code 实例，包括启动、停止、发送消息和实时通信。

## ADDED Requirements

### Requirement: HTTP API 服务

系统 SHALL 提供 HTTP API 服务用于管理 Claude Code。

#### Scenario: 启动 Claude 实例
- **GIVEN** Web 服务正在运行
- **WHEN** POST /api/instances with {"provider": "kimi", "project": "/path/to/project"}
- **THEN** 应当返回 201 状态码
- **AND** 响应体包含实例 ID
- **AND** Claude Code 进程应当在后台启动

#### Scenario: 发送消息
- **GIVEN** 实例正在运行
- **WHEN** POST /api/instances/{id}/messages with {"content": "hello"}
- **THEN** 应当返回 200 状态码
- **AND** 消息应当发送到 Claude Code
- **AND** 响应体包含 Claude 的回复

#### Scenario: 停止实例
- **GIVEN** 实例正在运行
- **WHEN** DELETE /api/instances/{id}
- **THEN** 应当返回 204 状态码
- **AND** Claude Code 进程应当停止

### Requirement: WebSocket 支持

系统 SHALL 支持 WebSocket 连接用于实时通信。

#### Scenario: 建立 WebSocket 连接
- **GIVEN** Web 服务正在运行
- **WHEN** WebSocket 连接到 /api/instances/{id}/stream
- **THEN** 应当建立连接
- **AND** 应当接收 Claude Code 的实时输出

#### Scenario: 发送消息通过 WebSocket
- **GIVEN** WebSocket 连接已建立
- **WHEN** 发送 JSON 消息 {"type": "user_message", "content": "test"}
- **THEN** 应当转发到 Claude Code
- **AND** 应当接收流式响应

### Requirement: 认证和授权

系统 SHALL 支持基本的 API 认证。

#### Scenario: API Key 认证
- **GIVEN** Web 服务配置了 API Key
- **WHEN** 请求不包含 X-API-Key 头
- **THEN** 应当返回 401 状态码

#### Scenario: 有效 API Key
- **GIVEN** 请求包含有效的 X-API-Key
- **WHEN** 访问受保护的端点
- **THEN** 应当正常处理请求
