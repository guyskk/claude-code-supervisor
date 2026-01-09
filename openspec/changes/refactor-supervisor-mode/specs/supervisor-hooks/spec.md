# supervisor-hooks Specification Delta

## ADDED Requirements

### Requirement: Supervisor 配置支持

系统 SHALL 支持通过 `ccc.json` 配置 Supervisor Mode 参数。

#### Scenario: 从配置文件读取 max_iterations
- **GIVEN** `ccc.json` 中包含 `{"supervisor": {"max_iterations": 20}}`
- **WHEN** Supervisor Mode 启动
- **THEN** 最大迭代次数应当为 20

#### Scenario: 配置文件默认值
- **GIVEN** `ccc.json` 中 `supervisor` 段不存在或不完整
- **WHEN** 读取 Supervisor 配置
- **THEN** 应当使用默认值：
  - `max_iterations`: 20
  - `timeout_seconds`: 600
  - `log_level`: "info"
  - `prompt_path`: "~/.claude/SUPERVISOR.md"

#### Scenario: 环境变量覆盖配置
- **GIVEN** `ccc.json` 中 `supervisor.max_iterations` 为 20
- **AND** 环境变量 `CCC_SUPERVISOR_MAX_ITERATIONS` 设置为 30
- **WHEN** Supervisor Mode 启动
- **THEN** 最大迭代次数应当为 30（环境变量优先）

### Requirement: 结构化日志输出

系统 SHALL 使用结构化日志记录 Supervisor 执行过程。

#### Scenario: 日志输出格式
- **GIVEN** Supervisor hook 执行
- **WHEN** 记录日志
- **THEN** 日志格式应当为：
  ```
  [时间戳] [级别] [模块] 键=值 ... 消息
  ```
- **AND** 时间戳格式应当为 ISO 8601 (2006-01-02T15:04:05.000Z)
- **AND** 级别应当为: DEBUG, INFO, WARN, ERROR

#### Scenario: 日志包含关键上下文
- **GIVEN** Supervisor hook 执行
- **WHEN** 记录日志
- **THEN** 日志应当包含：
  - `session_id`: 会话标识
  - `iteration`: 当前迭代次数
  - `max_iterations`: 最大迭代次数
  - `duration`: 执行耗时（如适用）

#### Scenario: 日志级别过滤
- **GIVEN** 日志级别设置为 "info"
- **WHEN** 代码调用 `logger.Debug()`
- **THEN** 不应当输出 debug 日志
- **AND** info、warn、error 日志应当正常输出

### Requirement: 进程超时控制

系统 SHALL 为 Supervisor 调用设置超时限制。

#### Scenario: 超时配置
- **GIVEN** `supervisor.timeout_seconds` 配置为 300
- **WHEN** Supervisor 调用超过 300 秒
- **THEN** 应当终止进程
- **AND** 应当记录超时错误日志
- **AND** 应当允许 Agent 停止

#### Scenario: 超时默认值
- **GIVEN** 未配置 `timeout_seconds`
- **WHEN** Supervisor 调用超过 600 秒
- **THEN** 应当使用默认超时 600 秒

## MODIFIED Requirements

### Requirement: Settings 文件生成

Supervisor Mode SHALL 生成包含 Stop hook 的单一 `settings.json` 文件。

#### Scenario: 生成带 Hook 的 Settings（更新）
- **GIVEN** Supervisor Mode 启用
- **AND** 配置中 `supervisor.max_iterations` 为 20
- **WHEN** 系统生成配置
- **THEN** 应当将配置写入 `~/.claude/settings.json`
- **AND** settings 中应当包含 `hooks.Stop` 配置
- **AND** hook 命令应当为 ccc 的绝对路径加 `supervisor-hook`

#### Scenario: Hook 命令格式（更新）
- **GIVEN** ccc 安装在 `/usr/local/bin/ccc`
- **WHEN** 系统生成 hook 配置
- **THEN** hook 命令应当为 `/usr/local/bin/ccc supervisor-hook`
- **AND** 不应当包含 `--state-dir` 参数

### Requirement: supervisor-hook 子命令

系统 SHALL 提供 `supervisor-hook` 子命令处理 Stop hook 事件。

#### Scenario: 正常 Hook 调用（更新）
- **GIVEN** 环境变量 `CCC_SUPERVISOR_HOOK` 未设置
- **AND** stdin 包含有效的 StopHookInput JSON
- **AND** 配置中 `max_iterations` 为 20
- **WHEN** 执行 `ccc supervisor-hook`
- **THEN** 应当调用 Supervisor claude 检查任务完成状态
- **AND** 应当记录结构化日志
- **AND** 应当根据 Supervisor 结果输出 JSON 到 stdout
- **AND** 应当在配置的超时时间内完成

#### Scenario: 记录详细执行日志（新增）
- **GIVEN** supervisor-hook 正在执行
- **WHEN** 发生关键事件
- **THEN** 应当记录：
  - 开始执行: INFO 级别，包含 session_id
  - 调用 claude: DEBUG 级别，包含完整命令
  - 收到输出: DEBUG 级别，包含原始行
  - 解析结果: INFO 级别，包含 completed 状态
  - 完成执行: INFO 级别，包含总耗时
- **AND** 所有日志应当写入 `supervisor-{session_id}.log` 文件

### Requirement: 防止死循环 - 迭代次数限制

系统 SHALL 限制迭代次数防止无限循环。

#### Scenario: 迭代次数限制（更新）
- **GIVEN** session 的迭代次数已达到配置的 `max_iterations`
- **WHEN** hook 被触发
- **THEN** 应当输出空内容
- **AND** 应当记录 WARN 日志说明达到限制
- **AND** 应当允许 Agent 停止

#### Scenario: 迭代次数递增（更新）
- **GIVEN** session 当前迭代次数为 3
- **AND** 配置的 `max_iterations` 为 20
- **WHEN** hook 被触发
- **THEN** 应当将迭代次数更新为 4
- **AND** 应当记录 DEBUG 日志: "iteration=4/20"
- **AND** 应当继续执行 Supervisor 检查

### Requirement: Supervisor Claude 调用

系统 SHALL 使用指定参数调用 Supervisor claude。

#### Scenario: Supervisor 命令构建（更新）
- **GIVEN** session_id 为 "abc123"
- **AND** 配置中 `timeout_seconds` 为 300
- **AND** SUPERVISOR.md 存在于配置的路径
- **WHEN** 构建 Supervisor 命令
- **THEN** 命令应当包含：
  - `claude`
  - `-p`
  - `--fork-session`
  - `--resume abc123`
  - `--verbose`
  - `--output-format stream-json`
  - `--json-schema` （包含 completed 和 feedback 字段）
  - supervisor prompt 作为位置参数
- **AND** 环境变量应当包含 `CCC_SUPERVISOR_HOOK=1`
- **AND** 进程超时应当设置为 300 秒

### Requirement: 结构化输出处理

系统 SHALL 解析 Supervisor 的 stream-json 输出。

#### Scenario: 解析 stream-json（更新）
- **GIVEN** Supervisor 输出 stream-json 格式
- **WHEN** 系统处理输出
- **THEN** 应当逐行解析输出
- **AND** 应当将 `type: "text"` 的内容记录到 DEBUG 日志
- **AND** 应当提取 `type: "result"` 中的结构化 JSON
- **AND** 应当记录解析事件到日志

#### Scenario: 解析错误处理（新增）
- **GIVEN** 某行无法解析为 JSON
- **WHEN** 系统处理该行
- **THEN** 应当记录 WARN 日志说明解析失败
- **AND** 应当继续处理后续行
- **AND** 不应当中断执行

#### Scenario: 进程异常终止（新增）
- **GIVEN** Supervisor claude 进程非零退出
- **WHEN** 系统检测到异常
- **THEN** 应当记录 ERROR 日志
- **AND** 应当包含 exit_code 和 stderr 内容
- **AND** 应当允许 Agent 停止

## REMOVED Requirements

无移除的需求。
