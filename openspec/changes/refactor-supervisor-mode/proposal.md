# Proposal: refactor-supervisor-mode

## 概述

全面重构 Supervisor Mode 实现，解决当前代码中存在的可维护性、可读性和可靠性问题，为后续的 Web 服务和 SDK 封装奠定基础。

## 动机

当前实现存在以下严重问题：

### 1. 配置硬编码
- 最大迭代次数（10次）硬编码在代码中
- Supervisor prompt 路径硬编码
- 超时时间（600秒）硬编码
- 无法通过配置灵活调整

### 2. 日志系统混乱
- 日志格式不统一，混杂着 `fmt.Fprintf` 和不同级别的输出
- 日志分散在文件和 stderr，难以追踪
- 缺少日志级别（info、warn、error、debug）
- 时间戳格式不一致
- 日志内容不够详细，缺少关键上下文

### 3. 进程管理脆弱
- `exec.Command` 的 stdout/stderr 处理复杂且容易出错
- 使用 goroutine 并发读取但没有超时控制
- 进程意外终止时缺少清理机制
- 没有统一的进程生命周期管理
- 缺少进程超时控制

### 4. 错误处理不规范
- 错误消息缺少上下文信息
- 没有统一的错误分类系统
- 错误包装不一致（有的用 `fmt.Errorf`，有的直接返回）
- 缺少错误码系统
- 用户看到的错误信息不够友好

### 5. 代码可维护性差
- `hook.go` 文件过长（374行），职责混杂
- 日志记录、进程管理、JSON 解析、状态管理混在一起
- 缺少清晰的抽象层
- 难以测试和扩展

## 变更内容

### 1. 配置化 Supervisor（ADDED）

在 `ccc.json` 中新增 `supervisor` 配置段：

```json
{
  "supervisor": {
    "enabled": false,
    "max_iterations": 20,
    "timeout_seconds": 600,
    "prompt_path": "~/.claude/SUPERVISOR.md",
    "log_level": "info"
  }
}
```

- `enabled`: 是否启用 Supervisor Mode（可通过环境变量覆盖）
- `max_iterations`: 最大迭代次数，默认 20
- `timeout_seconds`: 单次 Supervisor 调用超时时间
- `prompt_path`: Supervisor prompt 文件路径
- `log_level`: 日志级别（debug、info、warn、error）

### 2. 结构化日志系统（ADDED）

引入结构化日志系统：

```go
// internal/logger/logger.go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    With(fields ...Field) Logger
}

type Field struct {
    Key   string
    Value interface{}
}
```

日志输出格式：
```
[2025-01-07T10:30:45.123Z] [INFO] [supervisor-hook] session_id=abc123 iteration=3/20 Starting supervisor review
[2025-01-07T10:30:45.234Z] [DEBUG] [supervisor-hook] command="claude --fork-session --resume abc123"
[2025-01-07T10:30:47.456Z] [WARN] [supervisor-hook] duration=2.222s Supervisor took longer than expected
```

### 3. Claude 进程管理抽象（ADDED）

创建 `claude_agent_sdk` 包封装 Claude 命令行交互：

```go
// internal/claude_agent_sdk/agent.go
type Agent struct {
    config    *Config
    logger    Logger
    timeout   time.Duration
}

type RunOptions struct {
    SessionID   string
    Prompt      string
    OutputFormat string  // "stream-json", "json", "text"
    JSONSchema  string
    ForkSession bool
    Env         []string
}

type RunResult struct {
    Output           string
    StructuredOutput map[string]interface{}
    Duration         time.Duration
    Success          bool
    Error            error
}

func (a *Agent) Run(ctx context.Context, opts RunOptions) (*RunResult, error)
func (a *Agent) RunStream(ctx context.Context, opts RunOptions) <-chan StreamEvent
```

### 4. 统一错误处理（MODIFIED）

创建错误包实现统一错误处理：

```go
// internal/errors/errors.go
type ErrorType int

const (
    ErrTypeConfig ErrorType = iota
    ErrTypeNetwork
    ErrTypeProcess
    ErrTypeValidation
    ErrTypeTimeout
)

type AppError struct {
    Type    ErrorType
    Code    string
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e *AppError) Error() string
func (e *AppError) Unwrap() error
func NewError(typ ErrorType, code, message string, cause error) *AppError
func Wrap(err error, code, message string) *AppError
```

预定义错误码：
- `CCC_CONFIG_INVALID`: 配置文件无效
- `CCC_PROVIDER_NOT_FOUND`: 提供商不存在
- `CCC_CLAUDE_NOT_FOUND`: claude 命令未找到
- `CCC_SUPERVISOR_TIMEOUT`: Supervisor 调用超时
- `CCC_PROCESS_EXIT_ABNORMALLY`: 进程异常退出

### 5. 重构 hook.go（MODIFIED）

将 `hook.go` 拆分为多个职责清晰的模块：

- `internal/supervisor/hook.go` - Hook 处理核心逻辑（< 200 行）
- `internal/supervisor/executor.go` - Claude 进程执行（使用 SDK）
- `internal/supervisor/parser.go` - Stream JSON 解析
- `internal/supervisor/logger.go` - Supervisor 专用日志

## 影响范围

### 受影响的 specs

- `supervisor-hooks` - MODIFIED: 添加配置支持、改进日志、改进进程管理
- `cli` - MODIFIED: 支持从配置读取 supervisor 设置
- `error-handling` - ADDED: 新增统一错误处理规范

### 受影响的代码

- `internal/config/config.go` - 添加 Supervisor 配置解析
- `internal/cli/hook.go` - 重构为多模块
- `internal/cli/exec.go` - 使用新的 SDK
- `internal/supervisor/` - 重构和扩展
- 新增 `internal/logger/` - 日志系统
- 新增 `internal/errors/` - 错误处理
- 新增 `internal/claude_agent_sdk/` - Claude 命令封装
- `ccc.json` - 配置格式变更（向后兼容）

### 向后兼容性

- 现有的 `ccc.json` 配置继续有效
- `CCC_SUPERVISOR` 环境变量优先级高于配置文件
- 默认值保持与当前行为一致（max_iterations=10→20）
- 日志文件路径不变

## 实施计划

### Phase 1: 基础设施（独立可并行）
1. 实现日志系统
2. 实现错误处理系统
3. 创建 claude_agent_sdk 基础结构

### Phase 2: 配置支持
4. 扩展配置结构支持 supervisor 段
5. 更新 CLI 解析配置

### Phase 3: 重构实现
6. 重构 hook.go 使用新模块
7. 更新 exec.go 使用 SDK
8. 添加单元测试

### Phase 4: 验证和文档
9. 集成测试
10. 更新文档

## 风险和缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 重构范围大，可能引入新 bug | 高 | 充分测试，分阶段提交 |
| 配置格式变更可能破坏现有用户 | 中 | 向后兼容，提供迁移指南 |
| 日志格式变化影响现有解析 | 低 | 保持关键信息格式一致 |
| SDK 抽象可能过度设计 | 中 | 遵循 YAGNI 原则，按需简化 |

## 开放问题

1. **日志文件轮转**: 是否需要实现日志文件大小限制和轮转？
2. **进程池**: 对于未来的 Web 服务，是否需要进程池管理多个 claude 实例？
3. **遥测**: 是否需要添加指标收集（如 Supervisor 调用次数、耗时分布）？
