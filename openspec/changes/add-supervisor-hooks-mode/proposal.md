# Proposal: add-supervisor-hooks-mode

## Summary

使用 Claude Code Hooks 机制实现 Supervisor Mode，在每次 Agent 停止时自动进行 Supervisor 检查，根据反馈决定是否继续工作，形成自动迭代循环直到任务完成。

## Motivation

当前实现（`internal/supervisor` 包）的问题：
1. **需要独立管理输入输出**：ccc 需要自己处理用户输入、Agent 输出、Supervisor 输出的流转
2. **Session 管理复杂**：需要手动管理 session_id、resume、fork-session 等
3. **与 Claude Code 耦合**：需要模拟 Claude 的行为（--print, --output-format stream-json 等）

使用 Claude Code Hooks 的优势：
1. **利用 Claude 原生机制**：Claude 自己管理 Stop 事件的触发
2. **简化输入输出**：用户直接与 Claude 交互，不需要 ccc 中转
3. **更好的集成**：Hook 是 Claude 的官方机制，兼容性更好

## Proposed Solution

### 架构设计

```
用户执行: ccc --supervisor

1. ccc 生成两个 settings 文件：
   - settings-{provider}.json (包含 Stop hook，供 Claude 使用)
   - settings-{provider}-supervisor.json (无 hook，供 Supervisor 使用)

2. ccc 启动 claude --settings settings-{provider}.json

3. Claude 工作流程：
   用户输入 → Agent 执行 → 触发 Stop hook
     ↓
   ccc supervisor-hook 被调用
     ↓
   调用 claude --fork-session --resume <session_id> (Supervisor)
     ↓
   解析 Supervisor 结构化输出
     ↓
   输出 JSON: {"decision": "block", "reason": "反馈"}
     ↓
   Claude 收到反馈，继续工作
     ↓
   循环直到 Supervisor 确认完成
```

### 关键技术点

1. **Stop Hook 配置**：在 settings-{provider}.json 中添加 Stop hook
2. **supervisor-hook 子命令**：处理 hook 事件，调用 Supervisor
3. **结构化输出**：使用 `--json-schema` 让 Supervisor 返回 JSON
4. **状态管理**：用文件记录 session 的迭代次数（防止无限循环）
5. **输出保存**：将 Supervisor 原始输出保存到 jsonl 文件

### 防止无限循环

- 记录每个 session 的迭代次数到 `.claude/ccc/supervisor-{session_id}.json`
- 当迭代次数 >= 10 时，允许 Agent 停止
- `stop_hook_active` 字段用于检测是否已有 hook 在运行

### JSON Schema 输出

Supervisor 返回的结构：
```json
{
  "completed": boolean,  // true=任务完成，false=需要继续
  "feedback": string     // 当 completed=false 时，提供反馈
}
```

## Impact

- **新增命令**：`ccc supervisor-hook` 子命令
- **修改文件**：`settings-{provider}.json` 添加 hook 配置
- **新增文件**：
  - `settings-{provider}-supervisor.json` (Supervisor 专用配置)
  - `.claude/ccc/supervisor-{session_id}.json` (状态管理)
  - `.claude/ccc/supervisor-{session_id}-output.jsonl` (输出保存)

## Affected Specs

- `cli`：新增 `--supervisor` 参数和 `supervisor-hook` 子命令
- 新增 `supervisor-hooks` spec：定义 Supervisor Mode 的行为

## Files Changed

- `internal/cli/cli.go`：修改 supervisor 模式分支
- `internal/cli/hook.go` (新增)：supervisor-hook 子命令
- `internal/provider/provider.go`：支持生成带 hook 的 settings 和 supervisor 专用 settings
- `internal/supervisor/`：移除或重构（不再需要独立循环管理）
