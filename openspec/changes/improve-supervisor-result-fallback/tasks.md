## 1. 实现

- [x] 1.1 修改 `parseResultJSON` 函数，添加 fallback 逻辑
- [x] 1.2 更新 `runSupervisorWithSDK` 中的错误处理逻辑
- [x] 1.3 添加测试用例覆盖解析失败场景
- [x] 1.4 运行测试验证实现

## 2. 验证

- [x] 2.1 运行 `go test ./internal/cli/...`
- [x] 2.2 运行 `go test ./...` 确保无回归
- [x] 2.3 手动测试验证 fallback 行为
