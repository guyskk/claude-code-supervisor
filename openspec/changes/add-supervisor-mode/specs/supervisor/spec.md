# supervisor Specification

## Purpose

定义 Supervisor 模式的核心循环逻辑，实现 Agent 执行与 Supervisor 检查的自动循环，提升 Agent 任务完成质量。

## ADDED Requirements

### Requirement: Supervisor 主循环

系统 SHALL 实现 Agent Phase 与 Supervisor Phase 的自动循环，直到 Supervisor 确认任务完成。

#### Scenario: supervisor 循环启动
- **GIVEN** 用户执行 `ccc --supervisor`
- **WHEN** ccc 启动 Supervisor 模式
- **THEN** 应当显示 "Supervisor mode enabled"
- **AND** 应当进入 Agent Phase

#### Scenario: Agent Phase → Supervisor Phase
- **GIVEN** Agent 正在运行
- **WHEN** Agent 停止（等待用户输入）
- **THEN** 应当捕获 session_id 和用户输入
- **AND** 应当进入 Supervisor Phase

#### Scenario: Supervisor Phase → Agent Phase（未完成）
- **GIVEN** Supervisor 检查未完成
- **WHEN** Supervisor 输出反馈
- **THEN** 应当将反馈作为新的用户消息
- **AND** 应当回到 Agent Phase（resume session）

#### Scenario: Supervisor Phase → 结束（已完成）
- **GIVEN** Supervisor 检查已完成
- **WHEN** Supervisor 输出 `[TASK_COMPLETED]` 标记
- **THEN** 应当退出循环
- **AND** 应当 resume 原始 session
- **AND** 应当等待用户继续输入

### Requirement: Agent pty 交互

系统 SHALL 使用 pty 启动 Agent，实时解析 stream-json 输出，捕获 session_id 和用户输入。

#### Scenario: 启动 Agent pty
- **GIVEN** Supervisor 模式已启动
- **WHEN** 进入 Agent Phase
- **THEN** 应当使用 pty 启动 claude
- **AND** 参数应当包含 `--print --output-format stream-json`

#### Scenario: 解析 stream-json
- **GIVEN** Agent 正在输出 stream-json
- **WHEN** 接收到 stream 消息
- **THEN** 应当解析消息获取 session_id
- **AND** 应当将内容输出到用户终端

#### Scenario: 捕获用户输入
- **GIVEN** Agent 正在运行
- **WHEN** 用户输入内容
- **THEN** 应当通过 pty 传递给 Agent
- **AND** 应当记录用户输入（用于 Supervisor 检查）

#### Scenario: 检测 Agent 停止
- **GIVEN** Agent 正在运行
- **WHEN** Agent 等待用户输入（stream 结束或特定状态）
- **THEN** 应当检测到停止状态
- **AND** 应当进入 Supervisor Phase

### Requirement: Supervisor 调用

系统 SHALL 使用 fork-session 调用 Supervisor，传入 SUPERVISOR.md 和用户输入上下文。

#### Scenario: Fork session 调用 Supervisor
- **GIVEN** Agent 已停止，session_id 已获取
- **WHEN** 进入 Supervisor Phase
- **THEN** 应当使用 `--fork-session --resume <session_id>`
- **AND** 应当传入 `--system-prompt "$(cat SUPERVISOR.md)"`
- **AND** 应当传入用户输入作为上下文
- **AND** 应当使用 `--print --output-format stream-json`

#### Scenario: 检测完成标记
- **GIVEN** Supervisor 正在输出
- **WHEN** Supervisor 输出包含 `[TASK_COMPLETED]` 标记
- **THEN** 应当检测到完成标记
- **AND** 应当返回完成状态

#### Scenario: 捕获 Supervisor 反馈
- **GIVEN** Supervisor 检查未完成
- **WHEN** Supervisor 输出反馈
- **THEN** 应当捕获反馈内容
- **AND** 应当返回反馈内容

### Requirement: SUPERVISOR.md 提示词

系统 SHALL 支持 Supervisor 提示词配置，优先使用项目级配置，fallback 到全局配置。

#### Scenario: 读取项目级 SUPERVISOR.md
- **GIVEN** 项目根目录存在 `SUPERVISOR.md`
- **WHEN** Supervisor 需要提示词
- **THEN** 应当读取 `./SUPERVISOR.md`
- **AND** 应当作为 system-prompt 传给 Supervisor

#### Scenario: fallback 到全局 SUPERVISOR.md
- **GIVEN** 项目根目录不存在 `SUPERVISOR.md`
- **AND** `~/.claude/SUPERVISOR.md` 存在
- **WHEN** Supervisor 需要提示词
- **THEN** 应当读取 `~/.claude/SUPERVISOR.md`
- **AND** 应当作为 system-prompt 传给 Supervisor

#### Scenario: SUPERVISOR.md 不存在
- **GIVEN** 项目根目录和 `~/.claude/` 都不存在 `SUPERVISOR.md`
- **WHEN** Supervisor 需要提示词
- **THEN** 应当返回错误
- **AND** 应当提示用户创建 SUPERVISOR.md

### Requirement: stream-json 解析

系统 SHALL 解析 claude 的 stream-json 输出，提取 session_id 和消息类型。

#### Scenario: 解析 session_id
- **GIVEN** stream-json 包含 `sessionId` 字段
- **WHEN** 解析 stream 消息
- **THEN** 应当提取 session_id
- **AND** 应当保存用于后续 fork/resume

#### Scenario: 解析消息类型
- **GIVEN** stream-json 包含 `type` 字段
- **WHEN** 解析 stream 消息
- **THEN** 应当识别消息类型（text/result/error 等）
- **AND** 应当根据类型判断 Agent 状态

#### Scenario: 解析错误处理
- **GIVEN** stream-json 行格式错误
- **WHEN** 解析失败
- **THEN** 应当记录警告
- **AND** 应当继续处理后续消息
