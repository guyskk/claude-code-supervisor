# Tasks: provider-isolation

## Phase 1: 修改配置生成逻辑

- [ ] 1.1 修改 `internal/provider/provider.go` 的 `Switch()` 函数
- [ ] 1.2 移除将 `env` 写入 settings.json 的逻辑
- [ ] 1.3 保留其他配置（permissions、hooks 等）的写入
- [ ] 1.4 更新 `SwitchWithHook()` 函数，移除 env 写入逻辑

## Phase 2: 实现环境变量传递

- [ ] 2.1 修改 `internal/cli/exec.go` 的 `runClaude()` 函数
- [ ] 2.2 实现环境变量合并：`settings.env` + `provider.env`
- [ ] 2.3 通过环境变量传递合并后的 env 给 claude 子进程
- [ ] 2.4 支持 `${VAR}` 环境变量展开

## Phase 3: 更新测试

- [ ] 3.1 更新 `internal/provider/provider_test.go`
- [ ] 3.2 添加测试验证 env 不写入 settings.json
- [ ] 3.3 添加测试验证环境变量正确传递
- [ ] 3.4 集成测试验证完整流程

## Phase 4: 验证和提交

- [ ] 4.1 运行 `go test ./...` 确保所有测试通过
- [ ] 4.2 运行 `go vet ./...` 检查
- [ ] 4.3 运行 `gofmt -w .` 格式化代码
- [ ] 4.4 手动测试 provider 切换功能正常工作
