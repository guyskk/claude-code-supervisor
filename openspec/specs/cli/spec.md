# cli Specification

## Purpose

定义 ccc 命令行工具的交互行为和参数解析规范。包括基本命令、参数传递、帮助信息显示、版本信息显示以及 Supervisor Mode 相关的命令行行为。
## Requirements
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

### Requirement: 帮助信息显示

系统 SHALL 提供清晰的帮助信息，包括用法、命令和可用提供商列表。

#### Scenario: 显示帮助
- **WHEN** 用户请求帮助
- **THEN** 应当显示用法说明
- **AND** 应当显示可用命令列表
- **AND** 应当显示环境变量说明
- **AND** 如果配置可用，应当显示提供商列表

#### Scenario: 配置加载失败时的帮助
- **GIVEN** 配置文件不存在或格式错误
- **WHEN** 用户请求帮助
- **THEN** 应当显示帮助信息
- **AND** 应当显示配置文件路径
- **AND** 应当显示错误信息（截断到 40 字符）

### Requirement: 版本信息显示

系统 SHALL 能够显示版本和构建时间信息。

#### Scenario: 显示开发版本
- **GIVEN** 版本未设置（开发构建）
- **WHEN** 用户请求版本信息
- **THEN** 应当显示 "claude-code-supervisor version dev (built at unknown)"

#### Scenario: 显示发布版本
- **GIVEN** 版本为 "v0.1.2"，构建时间为 "2024-01-15T10:30:00Z"
- **WHEN** 用户请求版本信息
- **THEN** 应当显示 "claude-code-supervisor version v0.1.2 (built at 2024-01-15T10:30:00Z)"

### Requirement: Supervisor Mode 命令行

系统 SHALL 支持通过环境变量启用 Supervisor Mode。

#### Scenario: 环境变量启用 Supervisor Mode
- **GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置
- **WHEN** 用户执行 `ccc kimi`
- **THEN** 应当启用 Supervisor Mode
- **AND** 应当生成带 Stop hook 的 `settings.json`
- **AND** 应当启动 claude（不带 `--settings` 参数）

#### Scenario: 普通模式启动
- **GIVEN** 环境变量 `CCC_SUPERVISOR` 未设置
- **WHEN** 用户执行 `ccc kimi`
- **THEN** 应当使用普通模式
- **AND** 应当生成 `settings.json`（不带 hook）
- **AND** 应当启动 claude（不带 `--settings` 参数）

#### Scenario: 传递参数到 claude
- **GIVEN** 用户执行 `ccc kimi --debug /path/to/project`
- **WHEN** 系统启动 claude
- **THEN** 应当将 `--debug /path/to/project` 传递给 claude
- **AND** claude 启动时不应包含 `--settings` 参数

### Requirement: supervisor-hook 子命令

系统 SHALL 识别并处理 `supervisor-hook` 子命令。

#### Scenario: 识别 supervisor-hook 子命令
- **GIVEN** 用户执行 `ccc supervisor-hook --state-dir .claude/ccc`
- **WHEN** 系统解析命令
- **THEN** 应当识别为 supervisor-hook 子命令
- **AND** 应当调用 `RunSupervisorHook()` 函数

#### Scenario: 环境变量检查
- **GIVEN** 环境变量 `CCC_SUPERVISOR_HOOK=1` 已设置
- **WHEN** 执行 `ccc supervisor-hook`
- **THEN** 应当跳过 hook 执行
- **AND** 应当输出跳过信息到 stderr
- **AND** 应当返回空内容到 stdout

### Requirement: Claude 启动

系统 SHALL 使用 syscall.Exec 替换进程启动 claude。

#### Scenario: 普通模式启动 claude
- **GIVEN** 用户执行 `ccc kimi`
- **WHEN** 系统启动 claude
- **THEN** 应当使用 `syscall.Exec` 替换当前进程
- **AND** 应当设置 `ANTHROPIC_AUTH_TOKEN` 环境变量
- **AND** 命令行不应包含 `--settings` 参数

#### Scenario: Supervisor Mode 启动 claude
- **GIVEN** 环境变量 `CCC_SUPERVISOR=1` 已设置
- **WHEN** 系统启动 claude
- **THEN** 应当先调用 `provider.SwitchWithHook()` 生成配置
- **AND** 应当使用 `syscall.Exec` 替换当前进程
- **AND** 命令行不应包含 `--settings` 参数

### Requirement: Claude 进程执行

系统 SHALL 使用平台最优方式执行 claude 命令，在 Unix 系统上使用 exec 语义替换进程。

#### Scenario: Unix 系统使用 syscall.Exec
- **GIVEN** 系统为 Linux 或 macOS
- **AND** claude 可执行文件存在于 PATH 中
- **WHEN** ccc 切换到提供商并执行 claude
- **THEN** ccc 进程应当被 claude 进程替换
- **AND** 进程 PID 保持不变
- **AND** 环境变量正确传递给 claude

#### Scenario: Windows 系统使用子进程
- **GIVEN** 系统为 Windows
- **AND** claude 可执行文件存在于 PATH 中
- **WHEN** ccc 切换到提供商并执行 claude
- **THEN** ccc 应当创建子进程运行 claude
- **AND** ccc 等待子进程结束
- **AND** 环境变量正确传递给 claude

#### Scenario: claude 不在 PATH 中
- **GIVEN** claude 可执行文件不存在于 PATH 中
- **WHEN** ccc 尝试执行 claude
- **THEN** 应当返回错误 "claude not found in PATH"
- **AND** 退出码应当为非零

#### Scenario: 参数正确传递
- **GIVEN** 用户执行 `ccc kimi /path/to/project --help`
- **AND** claude_args 配置为 `["--verbose"]`
- **WHEN** ccc 执行 claude
- **THEN** claude 应当接收参数 `["--settings", "~/.claude/settings-kimi.json", "--verbose", "/path/to/project", "--help"]`

#### Scenario: 环境变量正确设置
- **GIVEN** 提供商配置包含 ANTHROPIC_AUTH_TOKEN
- **WHEN** ccc 执行 claude
- **THEN** claude 进程环境变量应当包含 ANTHROPIC_AUTH_TOKEN
- **AND** 其他环境变量应当从父进程继承

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

