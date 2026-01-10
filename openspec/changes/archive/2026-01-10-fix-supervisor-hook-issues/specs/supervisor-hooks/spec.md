# supervisor-hooks Spec Delta

## MODIFIED Requirements

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

#### Scenario: 正常 Hook 调用
- **GIVEN** 环境变量 `CCC_SUPERVISOR_HOOK` 未设置
- **AND** stdin 包含有效的 StopHookInput JSON
- **WHEN** 执行 `ccc supervisor-hook`
- **THEN** 应当调用 Supervisor claude 检查任务完成状态
- **AND** 应当根据 Supervisor 结果输出 JSON 到 stdout

#### Scenario: 任务完成
- **GIVEN** Supervisor 返回 `{"completed": true, "feedback": ""}`
- **WHEN** hook 处理 Supervisor 结果
- **THEN** 应当输出空内容（什么都不输出）
- **AND** 允许 Agent 停止

#### Scenario: 任务未完成
- **GIVEN** Supervisor 返回 `{"completed": false, "feedback": "需要补充测试"}`
- **WHEN** hook 处理 Supervisor 结果
- **THEN** 应当输出 `{"decision":"block","reason":"需要补充测试"}`
- **AND** Agent 应当继续工作

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

系统 SHALL 解析 Supervisor 的 stream-json 输出。

#### Scenario: 解析 stream-json
- **GIVEN** Supervisor 输出 stream-json 格式
- **WHEN** 系统处理输出
- **THEN** 应当将 `type: "text"` 的内容输出到 stderr
- **AND** 应当提取 `type: "result"` 中的 `structured_output` 字段
- **AND** 应当将原始输出保存到状态目录的 `supervisor-{session_id}-output.jsonl`

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

#### Scenario: 按行解析 JSON
- **GIVEN** Supervisor 输出多行 stream-json
- **WHEN** 系统处理输出
- **THEN** 应当逐行读取并尝试解析为 JSON
- **AND** 每行原始内容（无论是否能解析）都应写入 jsonl 文件
- **AND** 应当从解析成功的消息中提取 structured_output
