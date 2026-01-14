## 1. State 文件增加 enabled 字段
- [ ] 1.1 修改 `State` 结构体，增加 `Enabled bool` 字段
- [ ] 1.2 `LoadState()` 时处理旧文件（字段缺失默认 false）
- [ ] 1.3 更新 `state_test.go` 增加 `Enabled` 字段相关测试

## 2. SupervisorConfig 去掉 Enabled 字段
- [ ] 2.1 `SupervisorConfig` 结构体去掉 `Enabled` 字段
- [ ] 2.2 保留 `MaxIterations` 和 `TimeoutSeconds` 字段
- [ ] 2.3 `LoadSupervisorConfig()` 去掉 `CCC_SUPERVISOR` 环境变量读取逻辑
- [ ] 2.4 `DefaultSupervisorConfig()` 去掉 `Enabled: false`
- [ ] 2.5 更新 `supervisor_test.go`

## 3. 新增 supervisor-mode 子命令
- [ ] 3.1 `Command` 结构体增加 `SupervisorMode bool` 和 `SupervisorModeOpts` 字段
- [ ] 3.2 `SupervisorModeCommand` 结构体（包含 `Enabled bool` 字段）
- [ ] 3.3 `Parse()` 中识别 `supervisor-mode` 子命令
- [ ] 3.4 `parseSupervisorModeArgs()` 函数解析参数（on/off，默认 on）
- [ ] 3.5 `Run()` 中处理 `supervisor-mode` 子命令
- [ ] 3.6 新增 `RunSupervisorMode()` 函数：
  - 从 `CCC_SUPERVISOR_ID` 环境变量读取 ID
  - 加载 state 文件
  - 修改 `Enabled` 字段
  - 保存 state 文件
  - 通过 supervisor logger 输出到 stderr
- [ ] 3.7 更新 `ShowHelp()` 帮助信息（去掉 `CCC_SUPERVISOR` 说明）
- [ ] 3.8 为 `supervisor-mode` 子命令编写单元测试

## 4. runClaude 修改
- [ ] 4.1 检查 `CCC_SUPERVISOR_ID` 环境变量，如果没有则生成新的并设置
- [ ] 4.2 去掉 `supervisorMode bool` 参数
- [ ] 4.3 总是调用 `SwitchWithHook()`
- [ ] 4.4 更新相关测试

## 5. Hook 修改
- [ ] 5.1 去掉 `os.Getenv("CCC_SUPERVISOR")` 判断
- [ ] 5.2 去掉 `os.Getenv("CCC_SUPERVISOR_HOOK")` 判断
- [ ] 5.3 从 state 文件读取 `Enabled` 字段判断
- [ ] 5.4 如果 `Enabled == false`，直接允许 stop
- [ ] 5.5 更新 `hook_test.go`

## 6. Provider 修改
- [ ] 6.1 删除 `Switch()` 函数
- [ ] 6.2 `SwitchWithHook()` 中增加创建 `~/.claude/commands/supervisor.md` 的逻辑
- [ ] 6.3 `SwitchWithHook()` 中增加创建 `~/.claude/commands/supervisoroff.md` 的逻辑
- [ ] 6.4 每次调用都覆盖创建这两个文件
- [ ] 6.5 更新 `provider_test.go`（去掉 `Switch()` 相关测试）

## 7. 更新文档
- [ ] 7.1 README.md 更新新的使用方式
  - 去掉 `CCC_SUPERVISOR` 环境变量说明
  - 去掉 `supervisor.enabled` 配置说明
  - 增加 `/supervisor` slash command 使用说明
- [ ] 7.2 README-CN.md 同步更新

## 8. 验证和测试
- [ ] 8.1 运行 `go test ./...` 确保所有测试通过
- [ ] 8.2 运行 `go test -race ./...` 确保没有竞态条件
- [ ] 8.3 运行 `./check.sh --lint` 确保 lint 检查通过
- [ ] 8.4 运行 `./check.sh --build` 确保构建成功
- [ ] 8.5 手动测试新使用流程
