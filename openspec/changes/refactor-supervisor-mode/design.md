# Design: refactor-supervisor-mode

## Context

当前 Supervisor Mode 实现虽然功能可用，但存在严重的可维护性问题：
- 单一 374 行的 `hook.go` 文件混杂了太多职责
- 没有统一的日志和错误处理
- 进程管理代码脆弱且难以测试
- 配置硬编码导致灵活性差

随着产品规划的扩展（Web 服务、SDK 封装、多 agent 管理），需要一个更健壮的基础架构。

## Goals

- 提高代码可维护性和可测试性
- 统一日志和错误处理模式
- 抽象 Claude 命令行交互为可复用的 SDK
- 通过配置提供灵活性
- 为未来的 Web 服务和多 agent 管理奠定基础

## Non-Goals

- 不改变 Supervisor Mode 的核心工作流程
- 不修改已有的 spec 需求（只扩展）
- 不添加新的用户可见功能
- 不实现进程池（留给 Web 服务阶段）

## Decisions

### 决策 1: 日志系统 - 自实现而非依赖第三方库

**选择**: 自实现简单的结构化日志系统

**原因**:
- 项目目标是单一静态二进制，避免额外依赖
- 需求相对简单，不需要复杂的日志路由
- 标准库 `log/slog` (Go 1.21+) 功能足够

**替代方案**:
- `zap`: 性能最优，但增加二进制大小
- `logrus`: 功能丰富，但已不再维护
- `zerolog`: 零分配，但 API 设计不够直观

**实现**:
```go
// 使用标准库 log/slog
import "log/slog"

type Logger struct {
    slog *slog.Logger
}

func NewLogger(w io.Writer, level string) *Logger {
    var l slog.Level
    switch level {
    case "debug":
        l = slog.LevelDebug
    case "info":
        l = slog.LevelInfo
    case "warn":
        l = slog.LevelWarn
    case "error":
        l = slog.LevelError
    default:
        l = slog.LevelInfo
    }

    opts := &slog.HandlerOptions{
        Level: l,
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == slog.TimeKey {
                a.Value = slog.StringValue(a.Value.Time().Format("2006-01-02T15:04:05.000Z"))
            }
            return a
        },
    }

    logger := slog.New(slog.NewTextHandler(w, opts))
    return &Logger{logger}
}
```

### 决策 2: 错误处理 - 使用错误码和类型系统

**选择**: 定义 `AppError` 类型包含错误码、类型和上下文

**原因**:
- Go 1.13+ 错误包装 (`%w`) 提供了错误链基础
- 错误码允许程序化处理（如重试网络错误）
- 上下文字段帮助调试和日志记录

**实现**:
```go
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

func (e *AppError) Error() string {
    parts := []string{fmt.Sprintf("[%s]", e.Code)}
    if len(e.Context) > 0 {
        ctxParts := make([]string, 0, len(e.Context))
        for k, v := range e.Context {
            ctxParts = append(ctxParts, fmt.Sprintf("%s=%v", k, v))
        }
        parts = append(parts, strings.Join(ctxParts, " "))
    }
    parts = append(parts, e.Message)
    return strings.Join(parts, " ")
}

func (e *AppError) Unwrap() error {
    return e.Cause
}

// 使用示例
func LoadConfig(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return errors.NewError(
            errors.ErrTypeConfig,
            "CCC_CONFIG_READ_FAILED",
            "failed to read config file",
            err,
        ).With("path", path)
    }
    // ...
}
```

### 决策 3: Claude Agent SDK - 最小化抽象

**选择**: 创建轻量级的命令行封装，不实现完整的 Agent 抽象

**原因**:
- 当前只需要封装 Claude CLI 的调用
- 完整的 Agent SDK（如 agentsdk-go）过度设计
- 保持简单，按需扩展

**接口设计**:
```go
// Agent 封装 claude 命令行工具
type Agent struct {
    claudePath string
    logger     Logger
    timeout    time.Duration
}

type RunOptions struct {
    // 会话管理
    SessionID   string
    Resume      bool
    ForkSession bool

    // 输入
    Prompt      string

    // 输出控制
    OutputFormat string  // "stream-json", "json", "text"
    JSONSchema  string  // 结构化输出 schema

    // 环境
    Env []string
}

type StreamEvent struct {
    Type             string
    Content          string
    StructuredOutput map[string]interface{}
    Error            error
}

// Run 同步执行，返回完整结果
func (a *Agent) Run(ctx context.Context, opts RunOptions) (*RunResult, error)

// RunStream 流式执行，返回事件通道
func (a *Agent) RunStream(ctx context.Context, opts RunOptions) <-chan StreamEvent
```

**进程管理**:
```go
type Process struct {
    cmd     *exec.Cmd
    stdout  io.ReadCloser
    stderr  io.ReadCloser
    cancel  context.CancelFunc
    timeout time.Duration
}

func (p *Process) Start() error
func (p *Process) Wait() error
func (p *Process) Kill() error
func (p *Process) StdoutLine() <-chan string
func (p *Process) StderrLine() <-chan string
```

### 决策 4: 配置格式 - 扩展 ccc.json

**选择**: 在 `ccc.json` 顶层添加 `supervisor` 字段

**原因**:
- Supervisor 是核心功能，应该在主配置文件中
- 与 `providers`、`settings` 同级，结构清晰
- JSON 格式易于解析和验证

**配置结构**:
```json
{
  "settings": {...},
  "claude_args": [...],
  "current_provider": "kimi",
  "providers": {...},
  "supervisor": {
    "enabled": false,
    "max_iterations": 20,
    "timeout_seconds": 600,
    "prompt_path": "~/.claude/SUPERVISOR.md",
    "log_level": "info"
  }
}
```

**优先级**: 环境变量 > 配置文件 > 默认值
- `CCC_SUPERVISOR=1` > `supervisor.enabled`
- `CCC_SUPERVISOR_MAX_ITERATIONS` > `supervisor.max_iterations`

### 决策 5: 模块拆分 - 按职责分离

**原 `hook.go` (374行) 拆分为**:

| 新文件 | 职责 | 预估行数 |
|--------|------|----------|
| `hook.go` | Hook 入口，协调各模块 | ~150 |
| `executor.go` | Claude 进程执行 | ~200 |
| `parser.go` | Stream JSON 解析 | ~100 |
| `result.go` | 结果处理和决策 | ~100 |

**目录结构**:
```
internal/
├── logger/              # 新增
│   └── logger.go
├── errors/              # 新增
│   └── errors.go
├── claude_agent_sdk/    # 新增
│   ├── agent.go
│   ├── process.go
│   └── stream.go
├── config/
│   └── config.go        # MODIFIED
├── supervisor/
│   ├── state.go         # 保持
│   ├── stream.go        # 保持或移除（整合到 SDK）
│   ├── hook.go          # MODIFIED (重构)
│   ├── executor.go      # NEW
│   ├── parser.go        # NEW
│   └── result.go        # NEW
└── cli/
    ├── cli.go           # MODIFIED
    ├── exec.go          # MODIFIED (使用 SDK)
    └── hook.go          # 移除（功能移到 supervisor/)
```

## Risks / Trade-offs

| 风险 | 影响 | 缓解 |
|------|------|------|
| 重构破坏现有功能 | 高 | 充分测试，分阶段提交 |
| 抽象过度设计 | 中 | 遵循 YAGNI，只抽象当前需要 |
| 性能回归 | 低 | 日志系统设计考虑性能 |
| 配置迁移问题 | 中 | 向后兼容，提供文档 |

## Migration Plan

### 阶段 1: 准备（不破坏现有功能）
1. 创建新包但不集成
2. 添加测试验证行为
3. 文档更新

### 阶段 2: 逐步迁移
1. 先迁移日志系统（其他模块可用）
2. 再迁移错误处理
3. 最后重构 supervisor

### 阶段 3: 清理
1. 删除旧代码
2. 更新文档
3. 归档 OpenSpec change

### Rollback
- 每个 PR 独立可回滚
- 保持向后兼容，用户无感知
- Git 提供完整历史

## Open Questions

1. **日志持久化**: 当前日志写入文件，是否需要同时支持 stdout？（用于容器化部署）
2. **错误恢复**: 哪些错误应该自动重试？（如网络超时）
3. **指标收集**: 是否需要添加 Prometheus/OpenTelemetry 指标？
