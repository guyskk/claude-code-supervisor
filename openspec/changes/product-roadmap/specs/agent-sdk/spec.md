# agent-sdk Specification

## Purpose

定义 Agent SDK 的接口规范。Agent SDK 将 Claude Code 封装为可复用的 Go SDK，支持进程管理、会话管理和实时通信。

## ADDED Requirements

### Requirement: Agent 接口

系统 SHALL 定义统一的 Agent 接口。

#### Scenario: Agent 接口定义
- **GIVEN** 定义 Agent 接口
- **THEN** 应当包含以下方法：
  - Start() error - 启动 Agent
  - Stop() error - 停止 Agent
  - SendMessage(ctx, content) (Response, error) - 发送消息
  - Stream(ctx, content) (<-chan Event, error) - 流式通信
  - Status() Status - 获取状态

#### Scenario: Claude Code Agent 实现
- **GIVEN** ClaudeCodeAgent 实现 Agent 接口
- **WHEN** 调用 Start()
- **THEN** 应当启动 claude 进程
- **AND** 应当返回 nil error

### Requirement: 会话管理

系统 SHALL 支持会话的创建、恢复和删除。

#### Scenario: 创建新会话
- **GIVEN** Agent 已启动
- **WHEN** 调用 NewSession(options)
- **THEN** 应当返回 Session 对象
- **AND** Session 应当包含唯一 ID

#### Scenario: 恢复会话
- **GIVEN** 已有会话 ID
- **WHEN** 调用 ResumeSession(sessionID)
- **THEN** 应当恢复会话状态
- **AND** 应当返回 Session 对象

### Requirement: 事件系统

系统 SHALL 提供事件系统用于监听 Agent 状态变化。

#### Scenario: 订阅事件
- **GIVEN** Agent 已启动
- **WHEN** 调用 Subscribe(events)
- **THEN** 应当返回事件通道
- **AND** 应当发送状态变化事件

#### Scenario: 事件类型
- **GIVEN** 事件系统正在运行
- **THEN** 应当支持以下事件类型：
  - EventTypeStarted - Agent 启动
  - EventTypeStopped - Agent 停止
  - EventTypeMessage - 收到消息
  - EventTypeError - 发生错误
