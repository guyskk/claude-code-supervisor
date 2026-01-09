## ADDED Requirements

### Requirement: Supervisor 结果解析 Fallback

当 Supervisor 返回的结果无法解析为符合 Schema 的 JSON 时，系统 SHALL 将原始内容作为 feedback，并设置 `allow_stop=false` 让 Agent 继续工作。

#### Scenario: 解析失败时使用原始内容作为 feedback
- **GIVEN** Supervisor 返回的 result 内容无法解析为有效 JSON
- **WHEN** 系统尝试解析 Supervisor 结果
- **THEN** 应当将原始 result 内容作为 feedback
- **AND** 应当设置 `allow_stop=false`
- **AND** Agent 应当继续工作

#### Scenario: 空结果时的默认反馈
- **GIVEN** Supervisor 返回的 result 为空字符串
- **WHEN** 系统尝试解析 Supervisor 结果
- **THEN** 应当使用默认 feedback "请继续完成任务"
- **AND** 应当设置 `allow_stop=false`
