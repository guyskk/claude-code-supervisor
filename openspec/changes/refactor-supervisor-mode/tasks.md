# Tasks: refactor-supervisor-mode

## Phase 1: 基础设施（独立可并行）

### 1. 实现日志系统

- [ ] 1.1 创建 `internal/logger/logger.go`
- [ ] 1.2 定义 `Logger` 接口和 `Field` 类型
- [ ] 1.3 实现基于 `log/slog` 的 `TextLogger`
- [ ] 1.4 支持日志级别: debug, info, warn, error
- [ ] 1.5 添加时间戳格式化（ISO 8601）
- [ ] 1.6 实现 `With(fields ...Field) Logger` 方法
- [ ] 1.7 单元测试: 验证日志输出格式
- [ ] 1.8 单元测试: 验证日志级别过滤

### 2. 实现错误处理系统

- [ ] 2.1 创建 `internal/errors/errors.go`
- [ ] 2.2 定义 `ErrorType` 枚举
- [ ] 2.3 定义 `AppError` 结构体
- [ ] 2.4 实现 `Error()` 和 `Unwrap()` 方法
- [ ] 2.5 实现 `NewError()` 和 `Wrap()` 函数
- [ ] 2.6 实现预定义错误码常量
- [ ] 2.7 实现 `WithContext()` 方法
- [ ] 2.8 单元测试: 验证错误链
- [ ] 2.9 单元测试: 验证错误格式化

### 3. 创建 Claude Agent SDK 基础

- [ ] 3.1 创建 `internal/claude_agent_sdk/agent.go`
- [ ] 3.2 定义 `Agent` 结构体
- [ ] 3.3 定义 `RunOptions` 结构体
- [ ] 3.4 定义 `RunResult` 结构体
- [ ] 3.5 定义 `StreamEvent` 结构体
- [ ] 3.6 创建 `internal/claude_agent_sdk/process.go`
- [ ] 3.7 定义 `Process` 结构体
- [ ] 3.8 实现 `Start()`, `Wait()`, `Kill()` 方法
- [ ] 3.9 实现 `StdoutLine()` 和 `StderrLine()` 通道
- [ ] 3.10 添加超时控制
- [ ] 3.11 单元测试: 验证进程启动和终止
- [ ] 3.12 单元测试: 验证超时机制

## Phase 2: 配置支持（依赖 Phase 1）

### 4. 扩展配置结构

- [ ] 4.1 创建 `internal/config/supervisor.go`
- [ ] 4.2 定义 `SupervisorConfig` 结构体
- [ ] 4.3 添加 `enabled`, `max_iterations`, `timeout_seconds` 等字段
- [ ] 4.4 在 `Config` 中添加 `Supervisor *SupervisorConfig` 字段
- [ ] 4.5 实现 `Load()` 时解析 supervisor 配置
- [ ] 4.6 实现默认值处理
- [ ] 4.7 实现环境变量覆盖逻辑
- [ ] 4.8 单元测试: 验证配置解析
- [ ] 4.9 单元测试: 验证默认值
- [ ] 4.10 单元测试: 验证环境变量覆盖

### 5. 更新 CLI 集成配置

- [ ] 5.1 修改 `internal/cli/exec.go`
- [ ] 5.2 从配置读取 `supervisor.max_iterations`
- [ ] 5.3 从配置读取 `supervisor.timeout_seconds`
- [ ] 5.4 保留环境变量优先级
- [ ] 5.5 单元测试: 验证配置优先级

## Phase 3: 重构实现（依赖 Phase 1, 2）

### 6. 重构 Supervisor Hook

- [ ] 6.1 创建 `internal/supervisor/executor.go`
- [ ] 6.2 实现 `SupervisorExecutor` 结构体
- [ ] 6.3 使用 `claude_agent_sdk.Agent` 执行 claude 命令
- [ ] 6.4 实现超时控制
- [ ] 6.5 集成 `logger.Logger`
- [ ] 6.6 集成 `errors.AppError`
- [ ] 6.7 单元测试: 验证执行流程
- [ ] 6.8 单元测试: 验证超时处理

### 7. 创建 Stream Parser

- [ ] 7.1 创建 `internal/supervisor/parser.go`
- [ ] 7.2 定义 `StreamParser` 结构体
- [ ] 7.3 实现 `ParseLine()` 方法
- [ ] 7.4 处理 `type: "text"` 消息
- [ ] 7.5 处理 `type: "result"` 消息
- [ ] 7.6 提取 `structured_output` 字段
- [ ] 7.7 单元测试: 验证 JSON 解析
- [ ] 7.8 单元测试: 验证错误处理

### 8. 创建 Result Handler

- [ ] 8.1 创建 `internal/supervisor/result.go`
- [ ] 8.2 定义 `ResultHandler` 结构体
- [ ] 8.3 实现 `Handle()` 方法
- [ ] 8.4 处理 `completed: true` 情况
- [ ] 8.5 处理 `completed: false` 情况
- [ ] 8.6 生成 Hook 输出 JSON
- [ ] 8.7 单元测试: 验证输出格式

### 9. 重构 Hook 入口

- [ ] 9.1 重写 `internal/cli/hook.go`（移除旧实现）
- [ ] 9.2 创建 `HookHandler` 结构体
- [ ] 9.3 组合 `executor`, `parser`, `result` 模块
- [ ] 9.4 实现 `RunSupervisorHook()` 主流程
- [ ] 9.5 添加详细日志记录
- [ ] 9.6 添加错误处理
- [ ] 9.7 单元测试: 验证完整流程

### 10. 更新 exec.go 使用 SDK

- [ ] 10.1 修改 `internal/cli/exec.go`
- [ ] 10.2 使用 `claude_agent_sdk.Agent` 替代 `syscall.Exec`
- [ ] 10.3 保留 `syscall.Exec` 用于进程替换
- [ ] 10.4 单元测试: 验证启动流程

### 11. 更新 state.go 使用新模块

- [ ] 11.1 修改 `internal/supervisor/state.go`
- [ ] 11.2 集成 `logger.Logger`
- [ ] 11.3 使用 `errors.AppError`
- [ ] 11.4 单元测试: 验证状态管理

## Phase 4: 测试和验证

### 12. 集成测试

- [ ] 12.1 创建 `integration/supervisor_test.go`
- [ ] 12.2 测试配置加载
- [ ] 12.3 测试完整 supervisor 流程
- [ ] 12.4 测试迭代次数限制
- [ ] 12.5 测试超时处理
- [ ] 12.6 测试错误恢复

### 13. 端到端测试

- [ ] 13.1 手动测试: `CCC_SUPERVISOR=1 ccc kimi`
- [ ] 13.2 手动测试: 配置文件启用 supervisor
- [ ] 13.3 手动测试: 环境变量覆盖配置
- [ ] 13.4 验证日志输出清晰易读
- [ ] 13.5 验证错误信息友好

### 14. 性能测试

- [ ] 14.1 基准测试: 日志系统性能
- [ ] 14.2 基准测试: JSON 解析性能
- [ ] 14.3 基准测试: 进程启动性能
- [ ] 14.4 对比重构前后性能

## Phase 5: 文档和清理

### 15. 更新文档

- [ ] 15.1 更新 README.md 添加 supervisor 配置说明
- [ ] 15.2 更新 help 信息
- [ ] 15.3 创建 docs/supervisor-config.md
- [ ] 15.4 创建 docs/error-codes.md
- [ ] 15.5 创建 docs/migration-guide.md

### 16. 代码清理

- [ ] 16.1 删除 `internal/supervisor/stream.go`（功能移到 SDK）
- [ ] 16.2 更新所有包的导入
- [ ] 16.3 运行 `go mod tidy`
- [ ] 16.4 运行 `gofmt -w .`
- [ ] 16.5 运行 `go vet ./...`

### 17. 提交和 Review

- [ ] 17.1 运行 `go test ./...` - 确保通过
- [ ] 17.2 运行 `go test -race ./...` - 确保无竞态
- [ ] 17.3 运行 `go build` - 确保编译成功
- [ ] 17.4 提交 PR
- [ ] 17.5 自我 Review PR 内容
- [ ] 17.6 修复所有发现的问题

## Dependencies

- **Phase 1**: 所有任务可并行执行
- **Phase 2**: 依赖 Phase 1 完成
- **Phase 3**: 依赖 Phase 1 和 Phase 2 完成
- **Phase 4**: 依赖 Phase 3 完成
- **Phase 5**: 依赖 Phase 4 完成

## Parallelizable Tasks

Phase 1 中的所有子任务（1.x, 2.x, 3.x）可以完全并行执行。

## Notes

1. **向后兼容**: 确保现有配置继续工作
2. **测试优先**: 每个模块先写测试
3. **小步提交**: 每个 Phase 完成后提交一次
4. **日志示例**: 在文档中提供日志输出示例
5. **错误示例**: 在文档中提供错误信息示例
