# Tasks: add-supervisor-hooks-mode

## Implementation Tasks

### Phase 1: 基础设施

#### 1. 修改 provider 包支持生成带 hook 的 settings
- [ ] 在 `internal/provider/provider.go` 中添加 `SwitchWithHook()` 函数
- [ ] 生成 `settings-{provider}.json` 时添加 Stop hook 配置
- [ ] 生成独立的 `settings-{provider}-supervisor.json`（无 hook）
- [ ] 获取 ccc 绝对路径（使用 `os.Executable()`）
- [ ] 单元测试：验证生成的 settings 文件结构正确

#### 2. 创建状态管理模块
- [ ] 在 `internal/supervisor/` 中创建 `state.go`
- [ ] 实现 `LoadState(sessionID)` 函数
- [ ] 实现 `SaveState(sessionID, state)` 函数
- [ ] 实现 `IncrementCount(sessionID)` 函数
- [ ] 状态文件路径：`.claude/ccc/supervisor-{session_id}.json`
- [ ] 单元测试：验证状态的读写

### Phase 2: supervisor-hook 子命令

#### 3. 创建 hook 子命令
- [ ] 创建 `internal/cli/hook.go`
- [ ] 解析参数：`--settings`, `--state-dir`
- [ ] 读取 stdin JSON（StopHookInput 结构）
- [ ] 检查迭代次数限制（count >= 10 则返回空）
- [ ] 构建 Supervisor claude 命令
- [ ] 处理 stream-json 输出
- [ ] 输出 JSON 决定到 stdout
- [ ] 单元测试：验证输入输出处理

#### 4. 实现 stream-json 处理
- [ ] 解析 stream-json 行
- [ ] 提取 `result` 消息中的结构化 JSON
- [ ] 原始内容输出到 stderr
- [ ] 保存到输出文件（append 模式）
- [ ] 单元测试：验证 stream-json 解析

### Phase 3: CLI 集成

#### 5. 修改 CLI 支持 --supervisor 参数
- [ ] 在 `internal/cli/cli.go` 中修改 `Command` 结构
- [ ] 解析 `--supervisor` 参数
- [ ] 修改 `Run()` 函数的 supervisor 分支
- [ ] 调用 `provider.SwitchWithHook()` 生成配置
- [ ] 使用 `syscall.Exec` 启动 claude
- [ ] 更新帮助信息

#### 6. 更新 Supervisor Prompt
- [ ] 读取 `~/.claude/SUPERVISOR.md`
- [ ] 添加 JSON Schema 输出格式说明
- [ ] 创建默认的 SUPERVISOR.md（如果不存在）

### Phase 4: 测试和验证

#### 7. 运行单元测试
- [ ] `go test ./...` 确保所有测试通过
- [ ] `go test -race ./...` 确保无竞态条件

#### 8. 构建验证
- [ ] `./build.sh --all` 验证所有平台构建成功
- [ ] 验证：darwin-amd64, darwin-arm64, linux-amd64, linux-arm64

#### 9. 手动测试
- [ ] 测试 `ccc --supervisor` 启动
- [ ] 测试 supervisor-hook 子命令
- [ ] 测试迭代次数限制
- [ ] 测试任务完成检测

### Phase 5: 文档和清理

#### 10. 更新文档
- [ ] 更新 README.md 添加 supervisor mode 说明
- [ ] 更新 help 信息

#### 11. 清理旧代码
- [ ] 评估是否保留旧的 `internal/supervisor/supervisor.go`
- [ ] 如果不需要，删除或标记为 deprecated

## Dependencies

- **Phase 1**: 独立完成
- **Phase 2**: 依赖 Phase 1（状态管理）
- **Phase 3**: 依赖 Phase 1 和 Phase 2
- **Phase 4**: 依赖所有前面的 phases
- **Phase 5**: 依赖 Phase 4

## Parallelizable Tasks

以下任务可以并行执行：
- Task 1.1 和 Task 2（分别在不同的包中）
- Task 3 的各子任务可以分别开发和测试
- Task 6 可以与其他任务并行（只是文档更新）

## Notes

1. **绝对路径问题**：hook 命令中必须使用 ccc 的绝对路径，使用 `os.Executable()` 获取
2. **状态文件清理**：暂不实现自动清理，用户可以手动删除 `.claude/ccc/` 目录
3. **错误处理**：hook 中的错误应该输出到 stderr，不影响 Claude 的正常运行
