# PreToolUse Hook 手动验证指南

**功能**: 002-askuserquestion-hook
**创建日期**: 2026-01-20

本文档描述如何手动验证 Supervisor Hook 对 AskUserQuestion 工具调用的审查功能。

## 前置条件

1. **已编译的 ccc 二进制文件**
   ```bash
   cd /path/to/claude-code-supervisor1
   go build -o ccc ./cmd/ccc
   ```

2. **已配置的 Claude Code 环境**
   - Claude Code 已安装并可正常使用
   - `~/.claude/settings.json` 文件存在

3. **Supervisor 模式已启用**
   ```bash
   ./ccc supervisor on
   ```

## 验证步骤

### 步骤 1: 验证 Hook 配置已正确生成

**目的**: 确认 PreToolUse hook 配置已添加到 settings.json

**操作**:
```bash
cat ~/.claude/settings.json | jq '.hooks.PreToolUse'
```

**预期输出**:
```json
[
  {
    "matcher": "AskUserQuestion",
    "hooks": [
      {
        "type": "command",
        "command": "/path/to/ccc supervisor-hook",
        "timeout": 600
      }
    ]
  }
]
```

**验证点**:
- [ ] `PreToolUse` hook 存在于 hooks 配置中
- [ ] `matcher` 设置为 `"AskUserQuestion"`
- [ ] `command` 包含 `supervisor-hook`
- [ ] `timeout` 设置为 `600`

### 步骤 2: 验证 Stop Hook 仍然正常工作（向后兼容）

**目的**: 确认原有 Stop hook 功能未被破坏

**操作**:
```bash
# 创建测试输入
cat > /tmp/stop_hook_input.json <<'EOF'
{
  "session_id": "test-stop-$(date +%s)",
  "stop_hook_active": false
}
EOF

# 执行 hook
export CCC_SUPERVISOR_ID="test-stop-manual"
export CCC_SUPERVISOR_HOOK="1"  # 防止实际 SDK 调用
cat /tmp/stop_hook_input.json | ./ccc supervisor-hook
```

**预期输出**:
```json
{
  "reason": "called from supervisor hook"
}
```

**验证点**:
- [ ] Stop hook 返回正确的 JSON 格式
- [ ] 输出包含 `reason` 字段
- [ ] 没有 `hookSpecificOutput` 字段

### 步骤 3: 验证 PreToolUse Hook 输入解析

**目的**: 确认 PreToolUse hook 能正确解析输入

**操作**:
```bash
# 创建测试输入
cat > /tmp/pretooluse_input.json <<'EOF'
{
  "session_id": "test-pretooluse-$(date +%s)",
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {
    "questions": [
      {
        "question": "请选择实现方案",
        "header": "方案选择",
        "multiSelect": false,
        "options": [
          {"label": "方案A", "description": "使用方案A"},
          {"label": "方案B", "description": "使用方案B"}
        ]
      }
    ]
  },
  "tool_use_id": "toolu_manual_test_001"
}
EOF

# 执行 hook（使用 CCC_SUPERVISOR_HOOK=1 跳过 SDK 调用）
export CCC_SUPERVISOR_HOOK="1"
cat /tmp/pretooluse_input.json | ./ccc supervisor-hook
```

**预期输出**:
```json
{
  "reason": "called from supervisor hook"
}
```

**验证点**:
- [ ] Hook 成功解析 PreToolUse 输入
- [ ] 没有解析错误
- [ ] 输出包含 `reason` 字段

### 步骤 4: 验证 PreToolUse Hook 输出格式

**目的**: 确认 PreToolUse hook 返回正确的输出格式

**操作**:
```bash
# 启用 supervisor 模式
SESSION_ID="test-pretooluse-output-$(date +%s)"
export CCC_SUPERVISOR_ID="$SESSION_ID"

# 创建状态文件
cat > ~/.claude/ccc/supervisor-$SESSION_ID.json <<'EOF'
{
  "enabled": true,
  "iteration_count": 0
}
EOF

# 创建测试输入
cat > /tmp/pretooluse_output_test.json <<EOF
{
  "session_id": "$SESSION_ID",
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {},
  "tool_use_id": "toolu_output_test_001"
}
EOF

# 执行 hook（带 early return）
export CCC_SUPERVISOR_HOOK="1"
cat /tmp/pretooluse_output_test.json | ./ccc supervisor-hook
```

**注意**: 由于设置了 `CCC_SUPERVISOR_HOOK=1`，这会提前返回。完整流程需要真实的 SDK 调用。

**验证点**:
- [ ] 返回有效的 JSON
- [ ] 没有错误输出

### 步骤 5: 验证未知事件类型默认使用 Stop 格式

**目的**: 确认未知事件类型正确降级到 Stop 格式

**操作**:
```bash
export CCC_SUPERVISOR_ID="test-unknown-event-$(date +%s)"
export CCC_SUPERVISOR_HOOK="1"

cat > /tmp/unknown_event_input.json <<'EOF'
{
  "session_id": "test-unknown-event",
  "hook_event_name": "UnknownEventType",
  "tool_name": "SomeTool",
  "tool_input": {},
  "tool_use_id": "toolu_unknown_001"
}
EOF

cat /tmp/unknown_event_input.json | ./ccc supervisor-hook
```

**预期输出**: 应该返回 Stop 格式的输出（包含 `reason` 字段）

**验证点**:
- [ ] 未知事件类型不会导致错误
- [ ] 输出使用 Stop 格式

### 步骤 6: 验证迭代计数递增

**目的**: 确认 PreToolUse 事件会正确增加迭代计数

**操作**:
```bash
SESSION_ID="test-iteration-$(date +%s)"
export CCC_SUPERVISOR_ID="$SESSION_ID"

# 创建初始状态
cat > ~/.claude/ccc/supervisor-$SESSION_ID.json <<'EOF'
{
  "enabled": true,
  "iteration_count": 0
}
EOF

echo "初始迭代计数:"
jq '.iteration_count' ~/.claude/ccc/supervisor-$SESSION_ID.json

# 触发 PreToolUse hook
export CCC_SUPERVISOR_HOOK="1"
cat > /tmp/iteration_test_input.json <<EOF
{
  "session_id": "$SESSION_ID",
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {},
  "tool_use_id": "toolu_iteration_001"
}
EOF

cat /tmp/iteration_test_input.json | ./ccc supervisor-hook >/dev/null 2>&1 || true

echo "PreToolUse 后迭代计数:"
jq '.iteration_count' ~/.claude/ccc/supervisor-$SESSION_ID.json

# 触发 Stop hook
cat > /tmp/stop_iteration_input.json <<EOF
{
  "session_id": "$SESSION_ID",
  "stop_hook_active": false
}
EOF

cat /tmp/stop_iteration_input.json | ./ccc supervisor-hook >/dev/null 2>&1 || true

echo "Stop 后迭代计数:"
jq '.iteration_count' ~/.claude/ccc/supervisor-$SESSION_ID.json
```

**预期输出**:
```
初始迭代计数:
0
PreToolUse 后迭代计数:
1
Stop 后迭代计数:
2
```

**验证点**:
- [ ] PreToolUse 事件增加迭代计数
- [ ] Stop 事件也增加迭代计数
- [ ] 两种事件类型共享同一个计数器

### 步骤 7: 真实环境测试（可选）

**目的**: 在真实 Claude Code 环境中验证功能

**前提**:
- Supervisor 模式已启用
- 有一个正在进行的 Claude Code 会话

**操作**:
1. 在 Claude Code 中执行一个会触发 AskUserQuestion 的任务
2. 观察 hook 是否被触发
3. 检查 `~/.claude/ccc/` 目录中的日志文件

```bash
# 查看最新的 supervisor 状态文件
ls -lt ~/.claude/ccc/supervisor-*.json | head -1

# 查看迭代计数
jq '.' ~/.claude/ccc/supervisor-<session-id>.json
```

**验证点**:
- [ ] AskUserQuestion 调用被 hook 拦截
- [ ] 迭代计数正确递增
- [ ] Supervisor 的决策正确应用（允许/拒绝）

## 自动化测试脚本

项目中包含一个自动化端到端测试脚本：

```bash
./tests/e2e_pretooluse_hook_test.sh
```

该脚本会自动运行以上所有测试步骤并报告结果。

## 故障排查

### 问题 1: Hook 配置未生成

**症状**: `settings.json` 中没有 `PreToolUse` hook

**排查**:
```bash
# 检查是否正确切换了 provider
./ccc switch <provider-name>

# 检查 settings.json 内容
cat ~/.claude/settings.json | jq '.hooks'
```

### 问题 2: Hook 调用失败

**症状**: Hook 返回错误

**排查**:
```bash
# 检查 ccc 路径是否正确
which ccc

# 检查 hook 命令是否可执行
cat ~/.claude/settings.json | jq '.hooks.PreToolUse[0].hooks[0].command'

# 手动测试 hook 命令
echo '{}' | $(cat ~/.claude/settings.json | jq -r '.hooks.PreToolUse[0].hooks[0].command')
```

### 问题 3: Supervisor 模式未启用

**症状**: Hook 直接返回允许，没有经过审查

**排查**:
```bash
# 检查 supervisor 状态
./ccc supervisor

# 查看状态文件
cat ~/.claude/ccc/supervisor-*.json
```

## 测试检查清单

完成以下检查以确认功能正常：

- [ ] PreToolUse hook 配置正确生成
- [ ] Stop hook 仍然正常工作（向后兼容）
- [ ] PreToolUse 输入解析正确
- [ ] PreToolUse 输出格式正确
- [ ] 未知事件类型默认使用 Stop 格式
- [ ] 迭代计数正确递增
- [ ] 两种事件类型共享计数器
- [ ] 自动化测试全部通过

## 性能指标

| 操作 | 预期时间 |
|------|----------|
| Hook 配置生成 | < 1s |
| Hook 输入解析 | < 100ms |
| Hook 输出生成 | < 100ms |
| 完整 hook 调用（含 SDK） | < 30s |

## 下一步

完成验证后：
1. 更新 `specs/002-askuserquestion-hook/` 中的验证状态
2. 提交代码到 PR
3. 合并到主分支
