# central-server Specification

## Purpose

定义中心服务器的规范。中心服务器管理所有 ccc 实例，提供用户认证、设备管理和 AI 员工管理功能。

## ADDED Requirements

### Requirement: 设备注册

系统 SHALL 支持 ccc 实例注册到中心服务器。

#### Scenario: 设备注册
- **GIVEN** ccc 启动时配置了中心服务器地址
- **WHEN** 连接到中心服务器
- **THEN** 应当发送注册请求
- **AND** 请求包含设备信息（ID、名称、平台）
- **AND** 服务器应当返回设备 Token

#### Scenario: 心跳保活
- **GIVEN** 设备已注册
- **WHEN** 每隔 30 秒
- **THEN** 应当发送心跳
- **AND** 服务器应当更新设备状态

### Requirement: 用户认证

系统 SHALL 支持用户认证和授权。

#### Scenario: 用户注册
- **GIVEN** 新用户访问 Web 界面
- **WHEN** 提交注册信息
- **THEN** 应当创建用户账户
- **AND** 应当返回认证 Token

#### Scenario: 用户登录
- **GIVEN** 已注册用户
- **WHEN** 提交登录信息
- **THEN** 应当验证凭证
- **AND** 应当返回访问 Token

### Requirement: AI 员工管理

系统 SHALL 支持创建和管理 AI 员工。

#### Scenario: 创建员工
- **GIVEN** 用户已登录
- **WHEN** POST /api/employees with {"name": "AI助手", "agent_type": "claude"}
- **THEN** 应当返回 201 状态码
- **AND** 应当创建员工记录
- **AND** 应当分配到用户账户

#### Scenario: 批量创建员工
- **GIVEN** 用户已登录
- **WHEN** POST /api/employees/batch with count=5, template_id="xxx"
- **THEN** 应当创建 5 个员工
- **AND** 所有员工应当继承模板配置

### Requirement: 实时通信

系统 SHALL 支持 WebSocket 与 ccc 实例通信。

#### Scenario: 建立 Agent 连接
- **GIVEN** 用户选择一个 AI 员工
- **WHEN** WebSocket 连接到 /api/agents/{id}/ws
- **THEN** 应当建立到对应 ccc 实例的连接
- **AND** 消息应当双向转发

#### Scenario: 多人协作
- **GIVEN** 多个用户加入同一会话
- **WHEN** 任一用户发送消息
- **THEN** 所有用户应当收到消息
- **AND** Agent 应当响应一次
