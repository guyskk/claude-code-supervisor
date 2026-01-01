# Tasks: add-supervisor-mode

## Implementation Tasks

### 1. 添加依赖
- [x] 在 `go.mod` 中添加 `github.com/creack/pty v1.1.21`
- [x] 运行 `go mod tidy` 更新依赖
- [x] 验证：`go build` 成功

### 2. 创建 internal/supervisor 包结构
- [ ] 创建 `internal/supervisor` 目录
- [ ] 创建 `supervisor.go`（主循环）
- [ ] 创建 `agent.go`（Agent pty 交互）
- [ ] 创建 `supervisor_check.go`（Supervisor 调用）
- [ ] 创建 `stream.go`（stream-json 解析）
- [ ] 创建 `prompt.go`（SUPERVISOR.md 读取）

### 3. 实现 stream.go - stream-json 解析
- [x] 定义 `StreamMessage` 结构体
- [x] 实现 `ParseStreamJSONLine` 函数
- [x] 实现 `ExtractSessionID` 函数
- [x] 实现 `DetectAgentWaiting` 函数
- [x] 编写单元测试
- [x] 验证：`go test ./internal/supervisor/...` 通过

### 4. 实现 prompt.go - SUPERVISOR.md 读取
- [x] 实现 `GetSupervisorPrompt` 函数
- [x] 支持项目级 `./SUPERVISOR.md`
- [x] 支持 fallback `~/.claude/SUPERVISOR.md`
- [x] 编写单元测试
- [x] 验证：测试覆盖两种路径

### 5. 实现 supervisor_check.go - Supervisor 调用
- [x] 实现 `runSupervisorCheck` 函数
- [x] 构建 fork-session 命令
- [x] 捕获 Supervisor 输出
- [x] 检测 `[TASK_COMPLETED]` 标记
- [x] 返回完成状态和反馈

### 6. 实现 agent.go - Agent pty 交互
- [x] 实现 `AgentSession` 结构体
- [x] 实现 `StartAgent` 函数（pty 启动）
- [x] 实现 stream-json 实时解析
- [x] 实现用户输入捕获
- [x] 实现 Agent 停止检测
- [x] 实现 pty 输出到用户终端

### 7. 实现 supervisor.go - 主循环
- [x] 实现 `Supervisor` 结构体
- [x] 实现 `Run` 入口函数
- [x] 实现 `loop` 主循环
- [x] 实现 `runAgent` 阶段
- [x] 实现 `runSupervisorCheck` 阶段
- [x] 实现 `resumeFinal` 最终状态

### 8. 修改 cli.go - CLI 参数解析
- [x] 在 `Command` 结构体添加 `Supervisor bool`
- [x] 在 `Parse` 函数添加 `--supervisor` 参数解析
- [x] 在 `Run` 函数添加 supervisor 分支
- [x] 更新 `ShowHelp` 添加 supervisor 说明

### 9. 创建默认 SUPERVISOR.md
- [x] 在项目根目录创建 `SUPERVISOR.md`
- [x] 编写默认 Supervisor 提示词
- [x] 说明 `[TASK_COMPLETED]` 完成标记

### 10. 创建 cli spec delta
- [x] 创建 `openspec/changes/add-supervisor-mode/specs/cli/spec.md`
- [x] 添加 `Requirement: --supervisor 参数`
- [x] 添加相关 scenarios

### 11. 创建 supervisor spec
- [x] 创建 `openspec/changes/add-supervisor-mode/specs/supervisor/spec.md`
- [x] 添加 Supervisor 循环相关 requirements
- [x] 添加相关 scenarios

### 12. 运行测试和验证
- [x] 运行 `go test ./...` 确保通过
- [x] 运行 `go test -race ./...` 确保无竞态
- [x] 运行 `go vet ./...` 确保无警告
- [x] 运行 `gofmt -w .` 确保格式正确

### 13. 构建验证
- [x] 运行 `./build.sh --all` 验证所有平台构建
- [x] 验证：darwin-amd64, darwin-arm64, linux-amd64, linux-arm64

### 14. 验证 OpenSpec
- [x] 运行 `openspec validate add-supervisor-mode --strict`
- [x] 修复所有验证错误

## Dependencies

- Task 2 → Task 3,4,5,6,7（包结构必须先创建）
- Task 3,4 → Task 5,6（stream 和 prompt 是基础组件）
- Task 5,6 → Task 7（依赖基础组件）
- Task 7 → Task 8（supervisor 包完成后修改 CLI）
- Task 10,11 → Task 14（spec 完成后验证）

## Notes

- Task 5,6,7 是核心实现，需要仔细设计接口
- Task 12 的竞态检测在 CI 中必须通过
- Task 14 的 --strict 验证必须全部通过
