## MODIFIED Requirements

### Requirement: Supervisor Hook Execution
系统 SHALL 在 Stop hook 中调用 Supervisor 审查工作完成情况。

#### Scenario: 使用 Claude Agent SDK 执行 Supervisor
- **WHEN** Stop hook 触发且 Supervisor 模式启用
- **THEN** 使用 `schlunsen/claude-agent-sdk-go` 创建客户端
- **AND** 使用 fork session 模式恢复当前 session
- **AND** 传递结构化输出 schema 获取 JSON 格式的审查结果

#### Scenario: 解析 Supervisor 结构化输出
- **WHEN** SDK 返回 ResultMessage
- **THEN** 从 `StructuredOutput` 字段提取 `{completed: bool, feedback: string}`
- **AND** 根据结果决定是否允许停止

## ADDED Requirements

### Requirement: Fork Session Resume
Supervisor Hook SHALL 能够 fork 当前 session 并让 Claude 查看已完成的工作。

#### Scenario: Fork Session 上下文继承
- **WHEN** 使用 `WithForkSession(true)` 和 `WithSessionID(id)` 创建 SDK 客户端
- **THEN** Claude 能够访问原 session 的完整上下文
- **AND** Claude 能够查看所有已执行的工具调用和结果
