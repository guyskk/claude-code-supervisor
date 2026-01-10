# Tasks: use-syscall-exec

## Implementation Tasks

### 1. 创建 Unix 平台执行实现
- [x] 创建 `internal/cli/exec_unix.go`
- [x] 添加 `//go:build unix` build tag
- [x] 实现 `executeProcess` 函数，使用 `syscall.Exec`
- [x] 验证：`go build` 在 Linux/macOS 上成功

### 2. 创建 Windows 平台执行实现
- [x] 创建 `internal/cli/exec_windows.go`
- [x] 添加 `//go:build windows` build tag
- [x] 实现 `executeProcess` 函数，使用 `exec.Command().Run()`
- [x] 验证：交叉编译 `GOOS=windows go build` 成功

### 3. 重构 runClaude 函数
- [x] 使用 `exec.LookPath` 获取 claude 完整路径
- [x] 重构参数构建逻辑（argv[0] 必须是程序名）
- [x] 调用 `executeProcess` 替代 `exec.Command().Run()`
- [x] 移除不再需要的 stdin/stdout/stderr 设置（Unix 会继承）
- [x] 验证：`go build` 成功

### 4. 运行测试和验证
- [x] 运行 `go test ./...` 确保现有测试通过
- [x] 运行 `go test -race ./...` 确保无竞态条件
- [ ] 手动测试：`ccc kimi --help` 验证 claude 正常启动
- [ ] 手动测试：Ctrl+C 信号处理正常

### 5. 跨平台构建验证
- [x] 运行 `./build.sh --all` 验证所有平台构建成功
- [x] 验证：darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64

## Dependencies

- Task 3 依赖 Task 1 和 Task 2 完成
- Task 4 和 Task 5 可并行执行

## Notes

手动测试需要用户自行验证（需要实际配置的 ccc.json 和可用的 claude 命令）。
