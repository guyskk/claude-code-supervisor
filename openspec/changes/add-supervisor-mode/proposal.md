# Proposal: add-supervisor-mode

## Summary

添加 Supervisor 模式，实现 Agent 执行与 Supervisor 检查的自动循环，提升 Agent 任务完成质量。

## Why

当前 ccc 只是简单地切换配置并启动 claude，用户与 Agent 交互后无法自动检查工作质量。Supervisor 模式通过以下方式增强：

1. **自动检查点**：每次 Agent 停止后，自动启动 Supervisor 检查工作质量
2. **行动-反馈闭环**：Supervisor 反馈自动传给 Agent，形成改进循环
3. **用户无感知**：从用户视角仍然是直接与 Agent 交互
4. **任务完成判定**：Supervisor 确认任务完成后自动退出循环

## Proposed Solution

### 架构设计

```
用户运行: ccc --supervisor
    ↓
输出 "Supervisor mode enabled"
    ↓
进入循环：
┌─────────────────────────────────────────────────────────────┐
│ Agent Phase                                                 │
│  - 使用 pty 启动 claude（stream-json 模式）                  │
│  - 实时解析 stream，捕获 session_id 和用户输入               │
│  - Agent 停止（等待用户输入）时 → Supervisor Phase           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Supervisor Phase（用户无感知）                               │
│  - Fork session，传入 SUPERVISOR.md 作为 system-prompt       │
│  - 传入完整对话上下文（用户所有输入）                        │
│  - 检测 [TASK_COMPLETED] 标记                                │
│  - 未完成 → 反馈传给 Agent，回到 Agent Phase                 │
│  - 完成 → 退出循环，resume 原始 session                      │
└─────────────────────────────────────────────────────────────┘
```

### 核心组件

1. **internal/supervisor 包**
   - `supervisor.go`：主循环逻辑
   - `agent.go`：Agent pty 交互
   - `supervisor_check.go`：Supervisor 调用
   - `stream.go`：stream-json 解析
   - `prompt.go`：SUPERVISOR.md 读取

2. **SUPERVISOR.md**
   - 项目根目录或 `~/.claude/SUPERVISOR.md`
   - Supervisor 的系统提示词

3. **CLI 扩展**
   - 新增 `--supervisor` 参数

## Impact

- **新增依赖**：`github.com/creack/pty`（pty 交互）
- **新增配置**：项目级 SUPERVISOR.md
- **命令行变化**：新增 `--supervisor` 参数
- **行为变化**：supervisor 模式下 ccc 不使用 syscall.Exec，而是持续运行协调循环

## Affected Specs

- `cli`：新增 `--supervisor` 参数相关 requirements
- `supervisor`：新增 capability，定义 supervisor 循环逻辑

## Files Changed

- `internal/cli/cli.go`：新增 --supervisor 参数解析
- `internal/supervisor/supervisor.go`（新增）：主循环
- `internal/supervisor/agent.go`（新增）：Agent pty 交互
- `internal/supervisor/supervisor_check.go`（新增）：Supervisor 调用
- `internal/supervisor/stream.go`（新增）：stream-json 解析
- `internal/supervisor/prompt.go`（新增）：SUPERVISOR.md 读取
- `SUPERVISOR.md`（新增）：默认 Supervisor 提示词
- `go.mod`：新增 pty 依赖
- `openspec/specs/cli/spec.md`：新增 requirements
- `openspec/specs/supervisor/spec.md`（新增）：supervisor capability

## Risks / Trade-offs

- **pty 复杂性**：pty 交互比直接 exec 复杂，需要充分测试
- **平台兼容性**：pty 在 Unix 和 Windows 行为可能不同
- **性能**：stream-json 解析增加少量开销
- **维护成本**：新增约 500-800 行代码

## Migration Plan

无需迁移，--supervisor 是可选功能。

## Open Questions

- 无
