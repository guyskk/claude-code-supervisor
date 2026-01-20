# 快速入门：Supervisor Hook 支持 AskUserQuestion 工具调用审查

**功能**: 002-askuserquestion-hook
**创建日期**: 2026-01-20

## 关键验证场景

本文档描述如何验证 Supervisor Hook 对 AskUserQuestion 工具调用的审查功能。

### 场景 1: 启用 Supervisor 模式并触发 AskUserQuestion 审查

**目的**: 验证 AskUserQuestion 调用时正确触发 Supervisor 审查

**步骤**:
1. 启用 Supervisor 模式：
   ```
   /supervisor on
   ```
2. 在 Claude Code 中执行一个会触发 AskUserQuestion 的任务
3. 观察 hook 是否被触发（检查日志）

**预期结果**:
- AskUserQuestion 调用被拦截
- Supervisor hook 被触发
- 审查完成后返回决策

### 场景 2: Supervisor 阻止 AskUserQuestion 调用

**目的**: 验证 Supervisor 可以阻止不合理的 AskUserQuestion 调用

**步骤**:
1. 确保 Supervisor prompt 配置为严格模式
2. 触发一个不太合理的 AskUserQuestion 调用（例如过早提问）
3. 观察 Supervisor 的决策

**预期结果**:
- Supervisor 返回 `permissionDecision: "deny"`
- AskUserQuestion 调用被取消
- Claude Code 收到反馈并继续工作

### 场景 3: Supervisor 允许 AskUserQuestion 调用

**目的**: 验证 Supervisor 可以允许合理的 AskUserQuestion 调用

**步骤**:
1. 完成大部分工作后，触发一个合理的 AskUserQuestion 调用
2. 观察 Supervisor 的决策

**预期结果**:
- Supervisor 返回 `permissionDecision: "allow"`
- AskUserQuestion 调用正常执行
- 用户可以看到问题并选择答案

### 场景 4: 迭代计数在 AskUserQuestion hook 中递增

**目的**: 验证迭代计数在所有 hook 类型中一致递增

**步骤**:
1. 检查当前迭代计数：`cat ~/.claude/ccc/supervisor-<session-id>.json`
2. 触发 AskUserQuestion hook
3. 再次检查迭代计数

**预期结果**:
- 迭代计数增加 1

### 场景 5: 向后兼容 - Stop hook 继续工作

**目的**: 验证现有 Stop hook 功能不受影响

**步骤**:
1. 触发一个会触发 Stop hook 的场景（完成工作）
2. 观察 Stop hook 的行为

**预期结果**:
- Stop hook 正常工作
- 输出格式符合原有规范

## 测试检查清单

### 手动测试

- [ ] **TC-001**: Stop 事件 - 允许停止
- [ ] **TC-002**: Stop 事件 - 阻止停止
- [ ] **TC-003**: PreToolUse 事件 - 允许调用
- [ ] **TC-004**: PreToolUse 事件 - 阻止调用
- [ ] **TC-005**: 递归调用防护
- [ ] **TC-006**: 迭代计数限制
- [ ] **TC-007**: 向后兼容 - 无 hook_event_name

### 自动化测试

运行单元测试：
```bash
go test ./internal/cli/... -v
```

运行集成测试：
```bash
go test ./internal/cli/... -v -tags=integration
```

运行竞态检测：
```bash
go test ./internal/cli/... -race
```

## 调试技巧

### 启用详细日志

```bash
# 设置环境变量启用调试
export CCC_DEBUG=1
# 或
claude --debug
```

### 查看 hook 配置

```bash
# 查看当前 hooks 配置
cat ~/.claude/settings.json | jq '.hooks'
```

预期应看到：
```json
{
  "Stop": [...],
  "PreToolUse": [
    {
      "matcher": "AskUserQuestion",
      "hooks": [...]
    }
  ]
}
```

### 查看 Supervisor 状态

```bash
# 查看 supervisor 状态文件
cat ~/.claude/ccc/supervisor-<session-id>.json
```

### 查看 hook 日志

```bash
# 如果启用了日志文件
tail -f ~/.claude/ccc/supervisor-<session-id>.log
```

## 常见问题排查

### 问题 1: AskUserQuestion 没有被拦截

**可能原因**:
1. Supervisor 模式未启用
2. hooks 配置未正确更新
3. matcher 配置不正确

**排查步骤**:
1. 检查 Supervisor 模式状态：`/supervisor`（应该返回 "on"）
2. 检查 hooks 配置：`cat ~/.claude/settings.json | jq '.hooks.PreToolUse'`
3. 检查 ccc 版本：`ccc --version`

### 问题 2: hook 返回错误格式

**可能原因**:
1. 事件类型识别错误
2. 输出格式转换错误

**排查步骤**:
1. 查看日志中的事件类型
2. 检查 hook 输出的 JSON 格式
3. 验证 `hook_event_name` 字段值

### 问题 3: 迭代计数不递增

**可能原因**:
1. 状态文件写入失败
2. 计数逻辑未更新

**排查步骤**:
1. 检查状态文件权限
2. 查看日志中的计数输出
3. 手动检查状态文件内容

## 性能指标

### 预期响应时间

| 操作 | 预期时间 |
|------|----------|
| AskUserQuestion hook 触发 | < 1s |
| Supervisor SDK 调用 | < 30s |
| 总体审查时间 | < 30s |

### 资源使用

| 资源 | 预期值 |
|------|--------|
| 内存占用 | < 50MB (hook 进程) |
| CPU 使用 | < 10% (SDK 调用期间) |
| 状态文件大小 | < 1KB |

## 下一步

完成验证后，可以：
1. 查看 `tasks.md` 了解实现任务分解
2. 运行完整的测试套件
3. 提交 Pull Request
