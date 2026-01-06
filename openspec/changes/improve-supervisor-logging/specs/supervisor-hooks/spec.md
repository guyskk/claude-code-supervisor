## ADDED Requirements

### Requirement: Supervisor Mode 启动提示

当 Supervisor Mode 启动时，系统 SHALL 在 stderr 输出 log 文件路径信息。

#### Scenario: 显示 log 路径提示
- **GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置
- **WHEN** 用户执行 `ccc <provider>`
- **THEN** 应当在 stderr 输出 "[Supervisor Mode] 日志文件:" 提示
- **AND** 应当输出 state 目录路径
- **AND** 应当输出 hook 调用日志路径
- **AND** 应当输出 supervisor 输出日志路径

#### Scenario: State 目录路径计算
- **GIVEN** 环境变量 `CCC_WORK_DIR` 未设置
- **WHEN** 系统计算 state 目录路径
- **THEN** 路径应当为 `~/.claude/ccc`

#### Scenario: 自定义 State 目录
- **GIVEN** 环境变量 `CCC_WORK_DIR=/tmp/test` 已设置
- **WHEN** 系统计算 state 目录路径
- **THEN** 路径应当为 `/tmp/test/ccc`

### Requirement: Hook 执行日志输出

当 Stop hook 执行时，系统 SHALL 在 stderr 输出结构化的执行进度信息。

#### Scenario: Hook 调用开始
- **GIVEN** Stop hook 被触发
- **WHEN** `ccc supervisor-hook` 开始执行
- **THEN** 应当在 stderr 输出 "[SUPERVISOR HOOK] 开始执行" 分节符
- **AND** 应当输出 session_id 和当前迭代次数

#### Scenario: Supervisor 调用中
- **GIVEN** hook 准备调用 Supervisor
- **WHEN** Supervisor claude 启动
- **THEN** 应当在 stderr 输出 "[SUPERVISOR] 正在审查工作..."
- **AND** 应当输出 "请在新窗口查看日志文件了解详情"

#### Scenario: 审查结果输出
- **GIVEN** Supervisor 返回结果
- **WHEN** `completed` 为 `false`
- **THEN** 应当在 stderr 输出 "[SUPERVISOR] 任务未完成"
- **AND** 应当输出 feedback 内容
- **AND** 应当输出 "Agent 将根据反馈继续工作"

#### Scenario: 任务完成
- **GIVEN** Supervisor 返回 `completed: true`
- **WHEN** hook 处理结果
- **THEN** 应当在 stderr 输出 "[SUPERVISOR] 任务已完成"
- **AND** 应当输出 "允许停止"

### Requirement: 日志文件格式

系统 SHALL 使用易读的格式记录日志。

#### Scenario: hook-invocation.log 格式
- **GIVEN** hook 被调用
- **WHEN** 系统记录日志到 `hook-invocation.log`
- **THEN** 每条记录应当包含 ISO 8601 时间戳
- **AND** 应当包含事件类型（如 "supervisor-hook invoked"）
- **AND** 应当包含关键参数（如 session_id, iteration count）

#### Scenario: supervisor 输出日志格式
- **GIVEN** Supervisor 输出 stream-json
- **WHEN** 系统保存输出到 `supervisor-{session}-output.jsonl`
- **THEN** 应当保留原始 stream-json 行
- **AND** 应当同时在 hook-invocation.log 中记录摘要

## MODIFIED Requirements

### Requirement: 结构化输出处理

系统 SHALL 解析 Supervisor 的 stream-json 输出，并将关键信息输出到 stderr。

#### Scenario: 解析 stream-json
- **GIVEN** Supervisor 输出 stream-json 格式
- **WHEN** 系统处理输出
- **THEN** 应当将 `type: "text"` 的内容输出到 stderr
- **AND** 应当提取 `type: "result"` 中的结构化 JSON
- **AND** 应当将原始输出保存到 `{state_dir}/supervisor-{session_id}-output.jsonl`
- **AND** 应当在 stderr 输出审查结果摘要

#### Scenario: 结果 JSON Schema
- **GIVEN** Supervisor 被要求返回结构化结果
- **WHEN** Supervisor 返回结果
- **THEN** 结果应当符合以下 schema：
```json
{
  "type": "object",
  "properties": {
    "completed": {"type": "boolean"},
    "feedback": {"type": "string"}
  },
  "required": ["completed", "feedback"]
}
```
