# Tasks: refactor-supervisor-mode

## Phase 1: 简化配置结构

- [ ] 1.1 修改 `internal/config/supervisor.go` - 移除 `PromptPath` 和 `LogLevel` 字段
- [ ] 1.2 移除 `GetResolvedPromptPath()` 方法
- [ ] 1.3 更新 `LoadSupervisorConfig()` - 移除对 `prompt_path` 和 `log_level` 的解析
- [ ] 1.4 更新 `MarshalJSON()` 方法 - 移除相关字段
- [ ] 1.5 更新 `Validate()` 方法 - 移除对 `log_level` 的验证

## Phase 2: 简化 Hook 实现

- [ ] 2.1 修改 `internal/cli/hook.go` - 移除 `logLevel` 变量和相关逻辑
- [ ] 2.2 使用固定的 `logger.LevelInfo` 而非从配置读取
- [ ] 2.3 移除 `getSupervisorPrompt()` 函数，直接使用 `getDefaultSupervisorPrompt()`
- [ ] 2.4 更新日志输出，确保使用结构化日志

## Phase 3: 更新测试

- [ ] 3.1 更新 `internal/config/supervisor_test.go` - 移除相关测试
- [ ] 3.2 添加新测试验证配置向后兼容（忽略未知字段）
- [ ] 3.3 更新 `internal/cli/hook_test.go`（如存在）

## Phase 4: 验证和提交

- [ ] 4.1 运行 `go test ./...` 确保所有测试通过
- [ ] 4.2 运行 `go vet ./...` 检查
- [ ] 4.3 运行 `gofmt -w .` 格式化代码
- [ ] 4.4 手动测试 supervisor 功能正常工作
