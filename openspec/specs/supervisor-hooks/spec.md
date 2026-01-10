# supervisor-hooks Specification

## Purpose

定义 Supervisor Mode 使用 Claude Code Hooks 机制的行为规范。Supervisor Mode 通过 Stop hook 在每次 Agent 停止时自动进行 Supervisor 检查，根据反馈决定是否继续工作，形成自动迭代循环直到任务完成。
## Requirements
### Requirement: Supervisor Mode 启动

当 `CCC_SUPERVISOR=1` 环境变量设置时，系统 SHALL 启动 Supervisor Mode。

#### Scenario: 环境变量启用
- **GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置
- **WHEN** 用户执行 `ccc <provider>`
- **THEN** 应当生成带 Stop hook 的 `settings.json`
- **AND** 应当启动 claude（不带 `--settings` 参数）

#### Scenario: 环境变量未设置
- **GIVEN** 环境变量 `CCC_SUPERVISOR` 未设置或不为 "1"
- **WHEN** 用户执行 `ccc <provider>`
- **THEN** 应当使用普通模式启动 claude

### Requirement: Settings 文件生成

Supervisor Mode SHALL 生成包含 Stop hook 的单一 `settings.json` 文件。

#### Scenario: 生成带 Hook 的 Settings
- **GIVEN** Supervisor Mode 启用
- **WHEN** 系统生成配置
- **THEN** 应当将配置写入 `~/.claude/settings.json`
- **AND** settings 中应当包含 `hooks.Stop` 配置
- **AND** hook 命令应当是 ccc 的绝对路径加 `supervisor-hook`（不带参数）

#### Scenario: Hook 命令格式
- **GIVEN** ccc 安装在 `/usr/local/bin/ccc`
- **WHEN** 系统生成 hook 配置
- **THEN** hook 命令应当为 `/usr/local/bin/ccc supervisor-hook`

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

### Requirement: 防止死循环 - 环境变量

系统 SHALL 使用环境变量防止 Supervisor 的 hook 触发死循环。

#### Scenario: 检测到环境变量跳过执行
- **GIVEN** 环境变量 `CCC_SUPERVISOR_HOOK=1` 已设置
- **WHEN** 执行 `ccc supervisor-hook`
- **THEN** 应当输出 `{"decision":"","":""}` 到 stdout
- **AND** 应当立即返回（退出码 0）

#### Scenario: Supervisor Claude 启动时设置环境变量
- **GIVEN** hook 需要调用 Supervisor claude
- **WHEN** 构建 Supervisor claude 命令
- **THEN** 应当设置 `CCC_SUPERVISOR_HOOK=1` 环境变量
- **AND** Supervisor claude 应当继承该环境变量

#### Scenario: 完整防死循环流程
- **GIVEN** Agent claude 触发 Stop hook
- **WHEN** 第一次调用 `ccc supervisor-hook`（无 `CCC_SUPERVISOR_HOOK` 环境变量）
- **THEN** 应当启动 Supervisor claude（设置 `CCC_SUPERVISOR_HOOK=1`）
- **AND** 当 Supervisor claude 停止时触发 hook
- **AND** 第二次调用 `ccc supervisor-hook`（有 `CCC_SUPERVISOR_HOOK=1`）
- **AND** 应当返回 `{"decision":"","":""}`，允许 Supervisor 停止

### Requirement: 防止死循环 - 迭代次数限制

系统 SHALL 限制迭代次数防止无限循环。

#### Scenario: 迭代次数限制
- **GIVEN** session 的迭代次数已达到 10
- **WHEN** hook 被触发
- **THEN** 应当输出空内容
- **AND** 应当允许 Agent 停止

#### Scenario: 迭代次数递增
- **GIVEN** session 当前迭代次数为 3
- **WHEN** hook 被触发
- **THEN** 应当将迭代次数更新为 4
- **AND** 应当继续执行 Supervisor 检查

### Requirement: 状态管理

系统 SHALL 使用文件管理 session 状态。

#### Scenario: 状态目录确定
- **GIVEN** 环境变量 `CCC_CONFIG_DIR` 设置为 `/custom/path`
- **WHEN** 系统确定状态目录
- **THEN** 状态目录应当为 `/custom/path/ccc`

#### Scenario: 状态目录默认值
- **GIVEN** 环境变量 `CCC_CONFIG_DIR` 未设置
- **WHEN** 系统确定状态目录
- **THEN** 状态目录应当为 `~/.claude/ccc/`

#### Scenario: 状态文件路径
- **GIVEN** session_id 为 "abc123"
- **AND** 状态目录为 `.claude/ccc`
- **WHEN** 系统访问状态文件
- **THEN** 状态文件路径应当为 `.claude/ccc/supervisor-abc123.json`

#### Scenario: 状态文件结构
- **GIVEN** session_id 为 "abc123"
- **WHEN** 系统保存状态
- **THEN** 状态文件应当包含：
  - `session_id`: "abc123"
  - `count`: 迭代次数
  - `created_at`: 创建时间（ISO 8601）
  - `updated_at`: 更新时间（ISO 8601）

### Requirement: Supervisor Claude 调用

系统 SHALL 使用指定参数调用 Supervisor claude。

#### Scenario: Supervisor 命令构建
- **GIVEN** session_id 为 "abc123"
- **AND** SUPERVISOR.md 存在于 `~/.claude/SUPERVISOR.md`
- **AND** supervisor prompt 内容为 "你是严格的审查者..."
- **WHEN** 构建 Supervisor 命令
- **THEN** 命令应当包含：
  - `claude`
  - `--fork-session`（而不是 --print）
  - `--resume abc123`
  - `--verbose`
  - `--output-format stream-json`
  - `--json-schema` （包含 completed 和 feedback 字段）
  - user prompt 为 supervisor prompt + 具体指令（不使用 --system-prompt）
- **AND** 环境变量应当包含 `CCC_SUPERVISOR_HOOK=1`

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

