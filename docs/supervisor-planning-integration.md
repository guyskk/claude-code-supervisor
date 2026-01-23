# Supervisor Mode 与 Planning with Files 集成分析

## 概述

本文档分析如何将 Planning with Files 的上下文管理能力集成到 Claude Code Supervisor (ccc) 的 Supervisor Mode 中，以增强 Agent 的工作质量和任务完成度。

## 两个系统的核心差异

### Planning with Files

**核心目标**: 通过持久化文件管理 Agent 的"工作记忆"

**机制**:
- PreToolUse Hook: 在操作前读取 task_plan.md，刷新目标到注意力窗口
- PostToolUse Hook: 提醒更新阶段状态
- Stop Hook: 验证所有阶段完成才允许停止

**特点**:
- Agent 自我管理
- 被动式验证（基于脚本检查）
- 关注"任务是否按计划完成"

### Supervisor Mode

**核心目标**: 通过独立审查确保 Agent 工作质量

**机制**:
- Stop Hook: Fork 当前会话，运行 Supervisor Agent 进行质量审查
- PreToolUse Hook: 拦截 AskUserQuestion，检查是否真正需要提问

**特点**:
- 独立的监督者审查
- 主动式质量把控（基于 LLM 审查）
- 关注"工作质量是否达标"

## 集成价值分析

### 互补性

| 维度 | Planning with Files | Supervisor Mode | 集成后 |
|------|---------------------|-----------------|-------|
| 目标追踪 | ✅ 强（持久化计划文件） | ❌ 弱（依赖会话历史） | 强化 |
| 质量审查 | ❌ 弱（只检查阶段完成） | ✅ 强（深度质量审查） | 强化 |
| 错误学习 | ✅ 强（持久化错误记录） | ❌ 弱（依赖会话历史） | 强化 |
| 进度可见 | ✅ 强（文件可读） | ❌ 弱（日志文件） | 强化 |

### 预期收益

1. **更准确的需求理解**: Supervisor 可以读取 task_plan.md 中的目标声明，而非仅依赖会话历史
2. **更客观的进度验证**: Supervisor 可以检查 task_plan.md 中的阶段完成状态
3. **更完整的错误上下文**: Supervisor 可以读取 findings.md 中的错误记录和解决方案
4. **更高效的审查**: 有结构化的计划文件，Supervisor 审查更有针对性

## 集成架构设计

### 方案 A: Supervisor Prompt 增强

**思路**: 在 Supervisor Prompt 中添加对规划文件的读取指导

**实现**:

```markdown
## 规划文件检查（如果存在）

如果项目中存在以下规划文件，必须读取并作为审查依据：

### task_plan.md
- 读取目标声明，与用户需求对比
- 检查各阶段状态，验证是否全部完成
- 查看错误记录，确认是否都已解决

### findings.md
- 检查研究发现是否被充分利用
- 验证技术决策是否合理

### progress.md
- 查看会话日志，了解实际工作历程
- 检查测试结果，验证功能有效性
```

**优点**:
- 实现简单，只需修改 Supervisor Prompt
- 向后兼容，不使用规划文件时不影响现有功能
- 灵活，Supervisor 自行判断是否需要读取

**缺点**:
- 依赖 Supervisor Agent 的执行能力
- 没有强制性，可能被忽略

### 方案 B: Hook 协同机制

**思路**: 在 ccc 的 Hook 机制中集成 Planning with Files 的检查逻辑

**实现**:

```go
// 在 Stop Hook 中添加规划文件检查
func checkPlanningFiles(cwd string) (*PlanningStatus, error) {
    taskPlan := filepath.Join(cwd, "task_plan.md")
    if _, err := os.Stat(taskPlan); os.IsNotExist(err) {
        return nil, nil // 不存在规划文件，跳过
    }

    // 解析 task_plan.md
    content, _ := os.ReadFile(taskPlan)

    // 检查阶段完成状态
    totalPhases := countPhases(content)
    completedPhases := countCompletedPhases(content)

    // 提取目标声明
    goal := extractGoal(content)

    return &PlanningStatus{
        Goal:            goal,
        TotalPhases:     totalPhases,
        CompletedPhases: completedPhases,
        AllComplete:     totalPhases == completedPhases,
    }, nil
}
```

**Hook 协同流程**:

```
Agent 停止
    ↓
Stop Hook 触发
    ↓
┌─────────────────────────────────────────┐
│ 1. 检查规划文件是否存在                    │
│ 2. 如果存在，解析阶段完成状态              │
│ 3. 将规划状态传递给 Supervisor            │
└─────────────────────────────────────────┘
    ↓
Supervisor Fork Session
    ↓
┌─────────────────────────────────────────┐
│ Supervisor 审查时获得额外上下文：          │
│ - 目标声明                              │
│ - 阶段完成状态                           │
│ - 错误记录                              │
└─────────────────────────────────────────┘
    ↓
输出审查结果
```

**优点**:
- 强制性检查，不依赖 Supervisor 的执行能力
- 结构化数据，审查更精确
- 可以实现"规划文件不完整则直接拒绝"的硬性规则

**缺点**:
- 实现复杂，需要修改 Go 代码
- 需要解析 Markdown 文件格式
- 与 Planning with Files 的格式耦合

### 方案 C: 双重 Hook 系统

**思路**: 同时使用 Planning with Files 的 Hook 和 Supervisor Mode 的 Hook

**实现**:

在 `settings.json` 中配置多层 Hook：

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit|Bash|Read|Glob|Grep",
        "hooks": [
          {
            "type": "command",
            "command": "cat task_plan.md 2>/dev/null | head -30 || true"
          }
        ]
      },
      {
        "matcher": "AskUserQuestion",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/ccc supervisor-hook"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "echo '[planning] If this completes a phase, update task_plan.md'"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash check-complete.sh"
          }
        ]
      },
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/ccc supervisor-hook"
          }
        ]
      }
    ]
  }
}
```

**执行顺序**:

```
Agent 停止
    ↓
Stop Hook 1: check-complete.sh (Planning with Files)
    ↓ (如果阶段未完成，直接阻止停止)
Stop Hook 2: ccc supervisor-hook (Supervisor Mode)
    ↓ (质量审查)
最终决策
```

**优点**:
- 双重保障：先验证计划完成，再验证质量
- 各自独立，松耦合
- 可以分别启用/禁用

**缺点**:
- 两次 Hook 调用，延迟增加
- 需要用户手动配置 Planning with Files 的 Hook
- 两个系统的状态不共享

### 方案 D: 统一规划框架（推荐）

**思路**: 在 ccc 中内置规划文件支持，作为 Supervisor Mode 的增强功能

**实现**:

1. **自动初始化规划文件**

```go
// 在启用 Supervisor Mode 时自动创建规划文件
func initPlanningFiles(cwd string) error {
    templates := map[string]string{
        "task_plan.md": taskPlanTemplate,
        "findings.md":  findingsTemplate,
        "progress.md":  progressTemplate,
    }

    for filename, template := range templates {
        path := filepath.Join(cwd, filename)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            if err := os.WriteFile(path, []byte(template), 0644); err != nil {
                return err
            }
        }
    }
    return nil
}
```

2. **增强 Supervisor Prompt**

```markdown
## 规划文件审查

在审查 Agent 工作前，必须检查以下规划文件：

### 1. 读取 task_plan.md
- 确认目标声明与用户需求一致
- 检查所有阶段是否标记为 complete
- 查看错误记录是否都已解决

### 2. 读取 findings.md
- 验证研究发现是否被充分利用
- 检查技术决策是否有合理的理由

### 3. 读取 progress.md
- 了解实际工作历程
- 验证测试结果

### 判断原则
- 如果 task_plan.md 存在未完成的阶段 → allow_stop = false
- 如果 task_plan.md 中有未解决的错误 → allow_stop = false
- 如果 findings.md 中的决策未被实施 → allow_stop = false
```

3. **新增 `/planning` 命令**

创建 `~/.claude/commands/planning.md`：

```markdown
---
description: Initialize planning files for current task
---

Create the following planning files in the current directory:

1. task_plan.md - Task roadmap with phases and progress tracking
2. findings.md - Research findings and technical decisions
3. progress.md - Session log and test results

Use templates from Planning with Files methodology (Manus-style).
```

**优点**:
- 完整集成，统一的用户体验
- 规划文件作为 Supervisor 审查的结构化输入
- 可以通过命令快速初始化

**缺点**:
- 实现工作量较大
- 需要维护模板
- 增加了 ccc 的复杂度

## 推荐实现路径

### 阶段 1: Supervisor Prompt 增强（短期）

**工作量**: 小（1-2 小时）

**步骤**:
1. 修改 `internal/cli/supervisor_prompt_default.md`
2. 添加规划文件检查指导
3. 测试验证

**预期效果**:
- Supervisor 在审查时会主动读取规划文件（如果存在）
- 审查更有针对性
- 向后兼容，不影响现有功能

### 阶段 2: /planning 命令（中期）

**工作量**: 中（2-4 小时）

**步骤**:
1. 创建 `~/.claude/commands/planning.md` 命令文件
2. 在命令中嵌入模板内容
3. 文档更新

**预期效果**:
- 用户可以通过 `/planning` 快速初始化规划文件
- 与 Supervisor Mode 无缝配合

### 阶段 3: 双重 Hook 系统（中期）

**工作量**: 中（4-8 小时）

**步骤**:
1. 修改 `provider.SwitchWithHook` 支持配置额外的 Hook
2. 添加配置选项 `planning_hooks_enabled`
3. 在 ccc.json 中支持规划文件 Hook 配置

**预期效果**:
- 自动刷新目标到注意力窗口（PreToolUse）
- 自动提醒更新阶段状态（PostToolUse）
- 双重 Stop Hook 保障

### 阶段 4: 规划状态集成（长期）

**工作量**: 大（1-2 天）

**步骤**:
1. 实现 `internal/planning` 包
2. 解析 task_plan.md 提取结构化数据
3. 将规划状态传递给 Supervisor
4. 在 Supervisor 结果中包含规划文件更新建议

**预期效果**:
- Supervisor 获得结构化的规划上下文
- 可以实现"规划未完成则直接拒绝"的硬性规则
- 规划文件成为 Supervisor 审查的核心输入

## Supervisor Prompt 增强示例

以下是建议添加到 `supervisor_prompt_default.md` 的内容：

```markdown
---

## 规划文件审查（可选但推荐）

如果项目中存在规划文件，你应该读取它们作为审查的重要依据：

### 检查 task_plan.md

如果存在 `task_plan.md`，使用 Read 工具读取并检查：

1. **目标声明**: 与用户需求对比，确认理解一致
2. **阶段状态**: 所有阶段是否都标记为 "complete"
3. **错误记录**: 是否所有错误都已解决

**判断原则**:
- 存在未完成的阶段 → `allow_stop = false`，反馈应指出哪些阶段未完成
- 存在未解决的错误 → `allow_stop = false`，反馈应要求解决错误

### 检查 findings.md

如果存在 `findings.md`，检查：

1. **研究发现**: 是否被充分利用
2. **技术决策**: 是否有合理的理由
3. **资源链接**: 是否有遗漏的参考资料

### 检查 progress.md

如果存在 `progress.md`，检查：

1. **工作日志**: 是否有完整的工作记录
2. **测试结果**: 测试是否通过
3. **错误日志**: 是否有遗漏的问题

### 规划文件审查示例

```
# 审查步骤
1. 使用 Read 工具读取 task_plan.md
2. 检查 Goal 部分与用户需求是否一致
3. 检查每个 Phase 的 Status 是否为 complete
4. 检查 Errors Encountered 是否都有 Resolution
5. 如果有任何未完成项，将其纳入反馈
```

---
```

## 配置选项设计

建议在 `ccc.json` 中添加以下配置：

```json
{
  "supervisor": {
    "max_iterations": 20,
    "timeout_seconds": 600,
    "planning": {
      "enabled": true,
      "auto_init": false,
      "require_complete": false,
      "files": {
        "task_plan": "task_plan.md",
        "findings": "findings.md",
        "progress": "progress.md"
      }
    }
  }
}
```

**配置说明**:

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `planning.enabled` | 是否在 Supervisor 审查时检查规划文件 | `true` |
| `planning.auto_init` | 启用 Supervisor Mode 时是否自动创建规划文件 | `false` |
| `planning.require_complete` | 是否要求所有阶段完成才能通过审查 | `false` |
| `planning.files.*` | 规划文件的文件名 | 默认文件名 |

## 总结

将 Planning with Files 集成到 Supervisor Mode 可以带来显著的增强效果：

1. **更准确的审查**: Supervisor 可以基于结构化的计划文件进行审查，而非仅依赖会话历史
2. **更完整的上下文**: 规划文件提供了目标、进度、错误等关键信息
3. **双重保障**: 计划完成验证 + 质量审查，双重把关
4. **更好的用户体验**: 用户可以通过规划文件了解 Agent 的工作状态

推荐从 **Supervisor Prompt 增强** 开始，这是最简单且最有效的集成方式。后续可以根据实际需求逐步实现更深度的集成。
