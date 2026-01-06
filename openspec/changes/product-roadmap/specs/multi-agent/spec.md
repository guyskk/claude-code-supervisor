# multi-agent Specification

## Purpose

定义多 Agent 支持的规范。系统应当支持多种 AI CLI 工具，包括 Claude Code、Gemini CLI、OpenCode 等。

## ADDED Requirements

### Requirement: Agent 注册

系统 SHALL 支持注册和管理多个 Agent 类型。

#### Scenario: 注册 Agent 类型
- **GIVEN** 配置文件包含 agent_types
- **WHEN** 加载配置
- **THEN** 应当解析每个 agent_type
- **AND** 每个 agent_type 应当包含：
  - name: string - Agent 名称
  - command: string - 命令路径
  - args: []string - 默认参数
  - capabilities: []string - 能力列表

#### Scenario: 获取可用 Agent
- **GIVEN** 配置了多个 Agent
- **WHEN** 调用 ListAgents()
- **THEN** 应当返回所有可用 Agent
- **AND** 每个 Agent 应当显示其能力

### Requirement: Agent 适配器

系统 SHALL 使用适配器模式支持不同 Agent。

#### Scenario: Gemini CLI 适配器
- **GIVEN** GeminiAgent 实现 Agent 接口
- **WHEN** 调用 Start()
- **THEN** 应当启动 gemini-cli 进程
- **AND** 应当正确处理输入输出

#### Scenario: OpenCode 适配器
- **GIVEN** OpenCodeAgent 实现 Agent 接口
- **WHEN** 调用 Start()
- **THEN** 应当启动 opencode 进程
- **AND** 应当正确处理 TUI 输出

### Requirement: Agent 选择

系统 SHALL 根据任务自动选择合适的 Agent。

#### Scenario: 能力匹配
- **GIVEN** 用户请求需要 "code_review" 能力
- **WHEN** 调用 SelectAgent(capabilities)
- **THEN** 应当返回具备该能力的 Agent
- **AND** 如果有多个，返回第一个

#### Scenario: 手动选择 Agent
- **GIVEN** 用户指定使用 "gemini"
- **WHEN** 调用 GetAgent("gemini")
- **THEN** 应当返回 Gemini Agent
