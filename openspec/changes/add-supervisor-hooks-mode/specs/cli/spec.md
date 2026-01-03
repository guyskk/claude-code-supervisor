# cli Specification Delta

## ADDED Requirements

### Requirement: Supervisor Mode 参数

系统 SHALL 支持 `--supervisor` 参数，启用 Supervisor 模式。

#### Scenario: 启用 Supervisor Mode
- **GIVEN** 配置中存在提供商 "kimi"
- **WHEN** 用户执行 `ccc --supervisor`
- **THEN** 应当使用当前提供商（或第一个提供商）
- **AND** 应当生成包含 Stop hook 的 settings 文件
- **AND** 应当启动 claude 使用该 settings 文件

#### Scenario: Supervisor Mode 指定提供商
- **GIVEN** 配置中存在提供商 "glm"
- **WHEN** 用户执行 `ccc --supervisor glm`
- **THEN** 应当使用 glm 提供商
- **AND** 应当生成包含 Stop hook 的 settings 文件
- **AND** 应当启动 claude 使用该 settings 文件

#### Scenario: Supervisor Mode 传递参数
- **GIVEN** 用户执行 `ccc --supervisor kimi /path/to/project --help`
- **THEN** claude 应当接收参数 `["/path/to/project", "--help"]`
- **AND** Stop hook 应当被正确配置

### Requirement: supervisor-hook 子命令

系统 SHALL 提供 `supervisor-hook` 子命令，用于处理 Stop Hook 事件。

#### Scenario: 解析 hook 参数
- **WHEN** 执行 `ccc supervisor-hook --settings /path/to/settings.json --state-dir .claude/ccc`
- **THEN** 应当解析 settings 路径
- **AND** 应当解析状态目录路径
- **AND** 状态目录默认值应为 `.claude/ccc`

#### Scenario: 读取 hook 输入
- **GIVEN** stdin 包含 JSON: `{"session_id":"abc123","stop_hook_active":true,...}`
- **WHEN** supervisor-hook 被调用
- **THEN** 应当解析 session_id
- **AND** 应当解析 stop_hook_active

#### Scenario: 迭代次数限制
- **GIVEN** session 的迭代次数已达到 10 次
- **WHEN** supervisor-hook 被调用
- **THEN** 应当输出空到 stdout（允许停止）
- **AND** 退出码应为 0

#### Scenario: 调用 Supervisor
- **GIVEN** session 的迭代次数未达到限制
- **AND** stop_hook_active 为 true
- **WHEN** supervisor-hook 被调用
- **THEN** 应当调用 `claude --fork-session --resume <session_id> --json-schema <schema>`
- **AND** 应当使用 supervisor 专用 settings 文件
- **AND** 应当增加迭代计数

#### Scenario: Supervisor 返回完成
- **GIVEN** Supervisor 返回 `{"completed":true}`
- **WHEN** supervisor-hook 处理结果
- **THEN** 应当输出空到 stdout（允许停止）
- **AND** 退出码应为 0

#### Scenario: Supervisor 返回未完成
- **GIVEN** Supervisor 返回 `{"completed":false,"feedback":"需要补充..."}`
- **WHEN** supervisor-hook 处理结果
- **THEN** 应当输出 JSON: `{"decision":"block","reason":"需要补充..."}`
- **AND** 退出码应为 0

### Requirement: Settings 文件生成

系统 SHALL 生成两种 settings 文件用于 Supervisor Mode。

#### Scenario: 生成带 Hook 的 Settings
- **WHEN** 启用 Supervisor Mode
- **THEN** 应当生成 `settings-{provider}.json` 包含 Stop hook
- **AND** hook 命令应使用 ccc 的绝对路径
- **AND** hook 命令应包含正确的参数

#### Scenario: 生成 Supervisor 专用 Settings
- **WHEN** 启用 Supervisor Mode
- **THEN** 应当生成 `settings-{provider}-supervisor.json`
- **AND** 该文件不应包含任何 hooks 配置（避免递归）
- **AND** 其他配置应与主 settings 相同

#### Scenario: Hook 命令绝对路径
- **GIVEN** ccc 可执行文件位于 `/usr/local/bin/ccc`
- **WHEN** 生成 Stop hook 配置
- **THEN** hook 命令应为 `/usr/local/bin/ccc supervisor-hook ...`
- **AND** 应使用 `os.Executable()` 获取绝对路径

### Requirement: 状态管理

系统 SHALL 使用文件系统管理 Supervisor 状态。

#### Scenario: 创建状态文件
- **GIVEN** session_id 为 "abc123"
- **AND** 状态目录为 `.claude/ccc`
- **WHEN** supervisor-hook 首次被调用
- **THEN** 应当创建 `.claude/ccc/supervisor-abc123.json`
- **AND** 内容应为: `{"session_id":"abc123","count":1,...}`

#### Scenario: 更新迭代次数
- **GIVEN** 状态文件存在且 count 为 3
- **WHEN** supervisor-hook 再次被调用
- **THEN** 应当更新 count 为 4
- **AND** 应当更新 updated_at 时间戳

#### Scenario: 迭代次数达到限制
- **GIVEN** 状态文件中 count 为 10
- **WHEN** supervisor-hook 被调用
- **THEN** 应当返回空（允许停止）
- **AND** 不应再调用 Supervisor

### Requirement: 输出保存

系统 SHALL 保存 Supervisor 的原始输出到文件。

#### Scenario: 保存 stream-json 输出
- **GIVEN** session_id 为 "abc123"
- **AND** Supervisor 输出多行 stream-json
- **WHEN** supervisor-hook 处理输出
- **THEN** 应当创建 `.claude/ccc/supervisor-abc123-output.jsonl`
- **AND** 应当以 append 模式写入每一行
- **AND** 每行应为原始 JSON

#### Scenario: 原始输出到 stderr
- **GIVEN** Supervisor 输出 text 类型消息
- **WHEN** supervisor-hook 处理 stream-json
- **THEN** 应当将 content 输出到 stderr
- **AND** 不应影响 stdout 的 JSON 输出

## MODIFIED Requirements

### Requirement: 命令行参数解析

系统 SHALL 能够解析命令行参数，识别命令、选项和参数。

#### Scenario: 解析 --supervisor
- **WHEN** 用户执行 `ccc --supervisor`
- **THEN** 应当识别 Supervisor Mode 选项
- **AND** Command.Supervisor 应为 true

#### Scenario: 解析 --supervisor 与提供商
- **WHEN** 用户执行 `ccc --supervisor kimi`
- **THEN** 应当识别 "kimi" 为提供商名称
- **AND** Command.Supervisor 应为 true
