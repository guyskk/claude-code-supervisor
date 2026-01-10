## MODIFIED Requirements

### Requirement: supervisor-hook 子命令
系统 SHALL 提供 `supervisor-hook` 子命令处理 Stop hook 事件。

#### Scenario: 正常 Hook 调用（使用 Claude Agent SDK）
- **GIVEN** 环境变量 `CCC_SUPERVISOR_HOOK` 未设置
- **AND** stdin 包含有效的 StopHookInput JSON
- **WHEN** 执行 `ccc supervisor-hook`
- **THEN** 应当使用 `schlunsen/claude-agent-sdk-go` 创建客户端
- **AND** 使用 fork session 模式恢复当前 session（`WithForkSession(true)` 和 `WithSessionID(id)`）
- **AND** Claude 能够访问原 session 的完整上下文
- **AND** 应当根据 Supervisor 结果输出 JSON 到 stdout

#### Scenario: 解析 Supervisor 结果
- **WHEN** SDK 返回 ResultMessage
- **THEN** 应当从 Result 字段提取并解析 JSON
- **AND** 转换为 `{allow_stop: bool, feedback: string}` 格式
- **AND** 根据结果决定是否允许停止
