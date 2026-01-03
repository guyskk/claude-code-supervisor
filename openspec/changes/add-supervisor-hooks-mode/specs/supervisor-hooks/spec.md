# supervisor-hooks Specification Delta

## ADDED Requirements

### Purpose

定义 Supervisor Mode 的行为规范，使用 Claude Code Hooks 机制实现自动的 Agent-Supervisor 循环。

### Requirement: Supervisor Mode 工作流程

系统 SHALL 实现 Agent-Supervisor 自动循环，直到任务完成或达到迭代限制。

#### Scenario: 首次启动 Supervisor Mode
- **GIVEN** 用户执行 `ccc --supervisor`
- **AND** SUPERVISOR.md 文件存在
- **WHEN** Claude Code 首次触发 Stop hook
- **THEN** supervisor-hook 应被调用
- **AND** 迭代次数应初始化为 1
- **AND** Supervisor 应检查工作质量

#### Scenario: Supervisor 反馈继续工作
- **GIVEN** Supervisor 返回 `{"completed":false,"feedback":"需要添加错误处理"}`
- **WHEN** supervisor-hook 输出反馈
- **THEN** Claude 应收到 `{"decision":"block","reason":"需要添加错误处理"}`
- **AND** Claude 应继续工作
- **AND** 下次 Stop 时迭代次数应为 2

#### Scenario: Supervisor 确认完成
- **GIVEN** Supervisor 返回 `{"completed":true,"feedback":""}`
- **WHEN** supervisor-hook 处理结果
- **THEN** 应输出空到 stdout
- **AND** Claude 应停止（不再继续）

#### Scenario: 达到迭代限制
- **GIVEN** 迭代次数已达到 10 次
- **WHEN** supervisor-hook 被调用
- **THEN** 应输出空到 stdout
- **AND** 应允许 Claude 停止
- **AND** 不应再调用 Supervisor

### Requirement: JSON Schema 输出

系统 SHALL 使用 JSON Schema 强制 Supervisor 返回结构化输出。

#### Scenario: Supervisor 返回完成
- **GIVEN** Agent 已完成所有用户要求的任务
- **WHEN** Supervisor 被调用
- **THEN** 应返回 `{"completed":true,"feedback":"任务已成功完成"}`
- **AND** completed 字段应为 true

#### Scenario: Supervisor 返回未完成
- **GIVEN** Agent 未完成用户要求的任务
- **WHEN** Supervisor 被调用
- **THEN** 应返回 `{"completed":false,"feedback":"具体的问题和改进建议"}`
- **AND** completed 字段应为 false
- **AND** feedback 字段应包含具体的反馈

#### Scenario: JSON Schema 格式要求
- **WHEN** 调用 Supervisor
- **THEN** 应使用 `--json-schema` 参数指定 schema
- **AND** schema 应要求 completed 和 feedback 字段
- **AND** schema 应定义 completed 为 boolean 类型
- **AND** schema 应定义 feedback 为 string 类型

### Requirement: Supervisor Prompt

系统 SHALL 从 SUPERVISOR.md 读取 Supervisor 提示词。

#### Scenario: 读取 Supervisor Prompt
- **GIVEN** `~/.claude/SUPERVISOR.md` 文件存在
- **WHEN** 调用 Supervisor
- **THEN** 应使用 `--system-prompt` 参数传入 SUPERVISOR.md 内容
- **AND** SUPERVISOR.md 应包含 JSON Schema 输出格式说明

#### Scenario: 默认 Supervisor Prompt
- **GIVEN** `~/.claude/SUPERVISOR.md` 文件不存在
- **WHEN** 用户首次使用 Supervisor Mode
- **THEN** 应创建默认的 SUPERVISOR.md
- **AND** 默认内容应包含角色说明和输出格式要求

### Requirement: 状态文件管理

系统 SHALL 使用 JSON 文件管理每个 session 的迭代状态。

#### Scenario: 状态文件结构
- **GIVEN** session_id 为 "abc123"
- **WHEN** 创建状态文件
- **THEN** 文件路径应为 `.claude/ccc/supervisor-abc123.json`
- **AND** 内容应包含: session_id, count, created_at, updated_at
- **AND** count 应为当前迭代次数

#### Scenario: 状态文件持久化
- **GIVEN** 状态文件已存在
- **WHEN** supervisor-hook 被调用
- **THEN** 应读取现有状态
- **AND** 应增加 count
- **AND** 应更新 updated_at 时间戳
- **AND** 应保存回文件

#### Scenario: 状态文件并发处理
- **GIVEN** 多个 hook 可能同时执行（理论上不应发生）
- **WHEN** 读写状态文件
- **THEN** 应使用文件锁或原子操作避免竞态条件

### Requirement: 输出文件管理

系统 SHALL 保存 Supervisor 的原始输出到 JSONL 文件。

#### Scenario: 输出文件创建
- **GIVEN** session_id 为 "abc123"
- **WHEN** supervisor-hook 首次被调用
- **THEN** 应创建 `.claude/ccc/supervisor-abc123-output.jsonl`
- **AND** 文件应以 append 模式写入

#### Scenario: 输出文件内容
- **GIVEN** Supervisor 输出 stream-json
- **WHEN** 处理输出
- **THEN** 每行应作为 JSON 对象写入文件
- **AND** 应保持原始格式（包括 whitespace）
- **AND** 文件应为有效的 JSONL 格式

#### Scenario: 输出文件用途
- **GIVEN** 输出文件存在
- **WHEN** 用户需要调试
- **THEN** 文件可用于查看 Supervisor 的完整输出
- **AND** 文件可用于分析 Supervisor 的决策过程

### Requirement: 错误处理

系统 SHALL 正确处理 supervisor-hook 执行中的错误。

#### Scenario: Supervisor 调用失败
- **GIVEN** claude 命令执行失败
- **WHEN** supervisor-hook 调用 Supervisor
- **THEN** 应输出错误信息到 stderr
- **AND** 应返回空到 stdout（允许停止）
- **AND** 退出码应为 0（不影响 Claude）

#### Scenario: 状态文件读写失败
- **GIVEN** 状态文件读写权限不足
- **WHEN** supervisor-hook 尝试读写状态
- **THEN** 应输出错误信息到 stderr
- **AND** 应继续执行（使用默认值或跳过状态管理）

#### Scenario: JSON 解析失败
- **GIVEN** Supervisor 返回无效的 JSON
- **WHEN** supervisor-hook 解析结果
- **THEN** 应输出错误信息到 stderr
- **AND** 应返回空到 stdout（允许停止）

### Requirement: Fork Session 使用

系统 SHALL 使用 --fork-session 避免污染主 session。

#### Scenario: Supervisor 使用 Fork Session
- **WHEN** supervisor-hook 调用 Supervisor
- **THEN** 应使用 `--fork-session` 参数
- **AND** 应使用 `--resume <session_id>` 恢复上下文
- **AND** Supervisor 的输出不应影响主 session

#### Scenario: Supervisor Settings 隔离
- **GIVEN** 主 settings 包含 Stop hook
- **WHEN** 调用 Supervisor
- **THEN** 应使用 supervisor 专用 settings
- **AND** supervisor settings 不应包含任何 hooks
- **AND** 避免 hook 递归调用
