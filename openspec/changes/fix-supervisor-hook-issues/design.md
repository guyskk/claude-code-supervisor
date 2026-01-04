# Design: fix-supervisor-hook-issues

## 背景

`add-supervisor-hooks-mode` 的实现存在多个问题，需要进行代码重构和bug修复。

## 目标

1. 修复已知bug
2. 消除代码重复
3. 简化实现
4. 确保正确的行为

## 决策

### 决策1: 删除 ccc 二进制文件

**原因**: 二进制构建产物不应该提交到版本控制系统。

**替代方案**:
- 添加到 `.gitignore` 确保以后不会误提交
- CI/CD 负责构建和发布二进制文件

### 决策2: 移除 providerName 参数

**当前问题**: `runSupervisor` 接收 `providerName` 参数，但 `SwitchWithHook` 已经保存了 `cfg.CurrentProvider`。

**解决方案**: 从 `cfg.CurrentProvider` 直接获取 provider 名称。

**影响**: 函数签名简化，减少参数传递。

### 决策3: 合并 runClaude 和 runSupervisor

**当前问题**: 两个函数有大量重复代码：
- 查找 claude 可执行文件路径
- 构建命令行参数
- 设置环境变量（ANTHROPIC_AUTH_TOKEN）
- 执行进程（syscall.Exec）

**解决方案**: 创建统一的 `executeClaude` 函数：
```go
func executeClaude(cfg *config.Config, claudeArgs []string, mergedSettings map[string]interface{}) error
```

**替代方案考虑**:
- 保留两个函数但提取公共辅助函数 - 但差异很小，不需要额外抽象
- 使用函数选项模式 - 过度设计

### 决策4: CCC_SUPERVISOR_HOOK=1 时返回固定JSON

**当前问题**: 直接 `return nil`，没有输出到 stdout。

**问题**: Claude Code hook 需要从 stdout 读取 JSON 来决定是否阻止 stop。

**解决方案**: 返回 `{"decision":"","":""}`，空的 decision 表示不阻止。

### 决策5: 移除 --state-dir 参数

**当前问题**: 需要通过参数传递 state 目录，但这个目录是固定的。

**解决方案**:
1. 检查 `CCC_WORK_DIR` 环境变量
2. 如果未设置，使用 `~/.claude/ccc/`
3. 从 `GetStateDir()` 函数统一获取

**优势**:
- 简化 hook command: `ccc supervisor-hook` 而不是 `ccc supervisor-hook --state-dir ...`
- 用户可以通过环境变量控制，不需要修改 hook 配置

### 决策6: 使用 --fork-session 而不是 --print

**当前问题**: 使用 `--print` 不会保留 session 状态。

**解决方案**: 使用 `--fork-session` 创建子 session。

**区别**:
- `--print`: 只输出结果，不创建 session
- `--fork-session`: 创建子 session，保留父 session 的完整上下文

### 决策7: 不使用 --system-prompt

**当前问题**: 使用 `--system-prompt` 设置 supervisor prompt。

**问题**: Supervisor 应该使用与 Agent 相同的 system prompt（来自 settings.json），不能覆盖。

**解决方案**:
- Supervisor prompt + 具体指令 作为 user prompt 传递
- Claude 会继承 settings.json 中的 hooks 配置（包括 system prompt）

### 决策8: 流式输出处理

**当前问题**: 需要正确解析 stream-json 格式并记录原始输出。

**解决方案**:
1. 按行读取 stdout
2. 尝试解析每行为 JSON
3. 如果是有效的 `StreamMessage`，处理相应类型
4. 将原始内容（不管是否能解析）写入 jsonl 文件
5. 从 `type: "result"` 的消息中提取 `structured_output`

## 风险和权衡

| 风险 | 缓解措施 |
|------|----------|
| 合并函数可能引入回归 | 充分测试普通模式和 supervisor 模式 |
| 更改输出格式可能破坏 hook | 验证 Claude Code 能正确解析返回的 JSON |
| 移除参数可能影响现有配置 | 这是内部实现细节，不涉及用户配置 |

## 迁移计划

1. 修改代码实现
2. 运行测试验证
3. 更新 spec 文档（如有必要）

## 未决问题

无
