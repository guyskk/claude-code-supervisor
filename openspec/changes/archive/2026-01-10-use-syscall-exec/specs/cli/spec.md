# cli Specification Delta

## ADDED Requirements

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
