# Tasks: integrate-claude-sdk

## Phase 1: 添加依赖

- [x] 1.1 在 `go.mod` 中添加 `github.com/schlunsen/claude-agent-sdk-go` 依赖
- [x] 1.2 运行 `go mod tidy` 更新依赖

## Phase 2: 重写 Supervisor Hook

- [x] 2.1 重写 `internal/cli/hook.go` 中的 Supervisor 执行逻辑
- [x] 2.2 使用 `sdk.NewClient()` 创建客户端
- [x] 2.3 使用 `WithForkSession(true)` 和 `WithSessionID()` 实现 fork session
- [x] 2.4 使用 `WithJSONSchema()` 实现结构化输出
- [x] 2.5 处理响应消息，提取 `structured_output`

## Phase 3: 清理旧代码

- [x] 3.1 删除 `internal/claude_agent_sdk/` 整个目录
- [x] 3.2 移除相关导入语句
- [x] 3.3 运行 `go mod tidy` 清理未使用的依赖

## Phase 4: 更新测试

- [x] 4.1 更新或删除 `internal/claude_agent_sdk/*_test.go`
- [ ] 4.2 添加 Supervisor Hook 的集成测试

## Phase 5: 验证和提交

- [x] 5.1 运行 `go test ./...` 确保所有测试通过
- [x] 5.2 运行 `go vet ./...` 检查
- [x] 5.3 运行 `gofmt -w .` 格式化代码
- [ ] 5.4 手动测试 supervisor 功能正常工作
