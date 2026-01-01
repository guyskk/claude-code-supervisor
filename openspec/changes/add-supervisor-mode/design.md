# Design: add-supervisor-mode

## Context

ccc 当前只负责配置切换和启动 claude。Supervisor 模式需要在 ccc 和 claude 之间插入一个协调层，实现 Agent- Supervisor 自动循环。

### 约束条件

1. **用户无感知**：从用户视角仍然是直接与 Agent 交互
2. **向后兼容**：不影响现有的 ccc 行为
3. **单二进制**：所有功能静态链接到单个可执行文件
4. **跨平台**：支持 Linux/macOS（pty 支持），Windows 可选

## Goals / Non-Goals

### Goals
- 实现 Agent 执行 → Supervisor 检查 → 反馈改进的自动循环
- Supervisor 检查对用户透明
- 支持自定义 Supervisor 提示词（项目级和全局级）
- 可配置的完成标记

### Non-Goals
- Supervisor 的具体提示词内容（由用户定义）
- Windows 平台的完整 pty 支持（可降级到子进程模式）
- 可视化界面或进度条

## Decisions

### Decision 1: 使用 pty 实现 Agent 交互

**选择**：使用 `github.com/creack/pty` 库

**理由**：
- 需要捕获用户输入（pty 可以记录输入内容）
- 需要解析 stream-json 输出
- 需要检测 Agent 停止状态（stream 结束/特定消息）
- 用户交互体验接近原生 claude

**替代方案**：
- 直接使用 exec.Command().StdinPipe()：无法捕获终端输入
- 使用 --print 模式解析输出：失去交互性

### Decision 2: stream-json 解析

**选择**：实时逐行解析 stream-json

**理由**：
- stream-json 提供结构化数据（type, sessionId, content 等）
- 可以实时获取 session_id
- 可以检测 Agent 状态变化

**数据结构**：
```go
type StreamMessage struct {
    Type      string `json:"type"`       // "text", "result", "error", etc.
    SessionID string `json:"sessionId"`
    Content   string `json:"content"`
    // ...
}
```

### Decision 3: Supervisor 调用方式

**选择**：使用 --fork-session --resume <session_id> --print

**理由**：
- fork session 不修改原 session
- Supervisor 检查完全独立（用户无感知）
- --print 模式便于捕获输出

**命令**：
```bash
claude --fork-session --resume <session_id> \
       --system-prompt "$(cat SUPERVISOR.md)" \
       --print --output-format stream-json \
       "用户原始提问:\n<user_input>"
```

### Decision 4: 完成标记格式

**选择**：`[TASK_COMPLETED]`（独立一行）

**理由**：
- 简单明确，便于字符串匹配
- 不易误触发
- 可扩展（如 `[TASK_COMPLETED: optional_message]`）

### Decision 5: 用户输入捕获

**选择**：通过 pty 记录完整的用户输入（不包括 Supervisor 反馈）

**理由**：
- Supervisor 需要完整的对话上下文
- 排除 Supervisor 反馈避免污染上下文

**实现**：
- Agent Phase：pty 记录用户输入
- Supervisor Phase：不记录（内部检查）
- 反馈传递：作为新的用户消息，不加入上下文记录

## Architecture

### 组件关系

```
┌─────────────────────────────────────────────────────────────┐
│ internal/cli/cli.go                                         │
│  - 解析 --supervisor 参数                                    │
│  - 调用 supervisor.Run() 或原有 exec 逻辑                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ internal/supervisor/supervisor.go                           │
│  - 主循环：Agent Phase ↔ Supervisor Phase                    │
│  - 协调 session resume 和 fork                               │
└─────────────────────────────────────────────────────────────┘
          ↓                           ↓
┌─────────────────────┐   ┌─────────────────────────────────┐
│ agent.go            │   │ supervisor_check.go              │
│  - pty 启动 Agent   │   │  - Fork session 调用 Supervisor  │
│  - stream 解析      │   │  - 捕获输出，检测完成标记         │
│  - 用户输入捕获     │   │  - 返回反馈                       │
└─────────────────────┘   └─────────────────────────────────┘
          ↓                           ↓
┌─────────────────────┐   ┌─────────────────────────────────┐
│ stream.go           │   │ prompt.go                        │
│  - stream-json 解析 │   │  - 读取 SUPERVISOR.md            │
│  - 消息类型判断     │   │  - 项目级 > 全局级               │
└─────────────────────┘   └─────────────────────────────────┘
```

### 主循环逻辑

```go
func (s *Supervisor) loop() error {
    for {
        // Phase 1: Agent Phase
        sessionID, userInput, err := s.runAgent()
        s.sessionID = sessionID
        s.userInput = userInput  // 完整的用户输入（多轮）

        // Phase 2: Supervisor Phase
        completed, feedback, err := s.runSupervisorCheck()

        if completed {
            break  // 任务完成
        }

        // Phase 3: 反馈传给 Agent
        s.userInput = feedback  // 作为新的用户消息
    }

    // Phase 4: Resume 最终 session
    return s.resumeFinal()
}
```

## Data Flow

### Agent Phase

```
用户输入 → pty → claude → stream-json → 解析
                ↓
         捕获用户输入
         捕获 session_id
         检测停止状态
```

### Supervisor Phase

```
SUPERVISOR.md + 用户输入 → fork session → Supervisor → stream-json
                                                      ↓
                                               检测 [TASK_COMPLETED]
```

### 反馈传递

```
Supervisor 输出 → 用户消息 → resume Agent
```

## Error Handling

| 错误场景 | 处理方式 |
|---------|---------|
| claude 不在 PATH | 返回错误，退出 |
| SUPERVISOR.md 不存在 | 返回错误，提示创建 |
| pty 启动失败 | 返回错误，退出 |
| stream-json 解析失败 | 记录警告，继续处理 |
| Supervisor 调用失败 | 记录错误，退出循环 |

## Testing Strategy

1. **单元测试**
   - stream-json 解析
   - SUPERVISOR.md 读取（优先级）
   - 完成标记检测

2. **集成测试**
   - 使用 mock claude 测试循环逻辑
   - 测试 pty 启动和交互

3. **手动测试**
   - 实际运行 `ccc --supervisor`
   - 验证 Agent- Supervisor 循环
   - 验证完成标记检测

## Risks / Trade-offs

| 风险 | 缓解措施 |
|-----|---------|
| pty 跨平台兼容性 | Unix 优先，Windows 降级 |
| stream-json 格式变化 | 健壮解析，容错处理 |
| 性能开销 | 异步解析，缓冲输出 |
| 维护成本 | 清晰的模块边界，充分测试 |

## Open Questions

- 无
