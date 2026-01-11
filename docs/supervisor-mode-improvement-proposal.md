# CCC Supervisor Mode 改进方案 (AI-First)

> 深入分析 Ralph-claude-code 项目，提出 AI-First 的 Supervisor Mode 增强方案

## 文档信息

| 项目 | 内容 |
|------|------|
| **版本** | 2.0 (AI-First) |
| **日期** | 2025-01-09 |
| **状态** | 设计阶段 |
| **作者** | AI 分析 |

---

## 目录

1. [执行摘要](#执行摘要)
2. [Ralph 项目深度分析](#ralph-项目深度分析)
3. [核心洞察与设计原则](#核心洞察与设计原则)
4. [改进方案总览](#改进方案总览)
5. [详细设计方案](#详细设计方案)
6. [实施计划](#实施计划)
7. [附录](#附录)

---

## 执行摘要

### 问题重新定义

经过深入分析，我们发现：

1. **Ralph 是"半 AI + 半规则"**：让 AI 输出结构化状态块，然后用规则解析
2. **CCC 是"纯 AI 检测"**：每次停止时调用 Supervisor AI 审查
3. **规则检测的局限性**：需要预定义所有情况，难以覆盖边缘场景
4. **AI 检测的优势**：理解上下文，处理边缘情况，长期更优

### 核心洞察

```
Ralph 的检测机制：
┌─────────────────────────────────────────────────────────────┐
│  PROMPT.md 指导 Agent 输出结构化状态                          │
│              ↓                                               │
│  Agent 输出包含 ---RALPH_STATUS--- 块                        │
│              ↓                                               │
│  response_analyzer.sh 用规则解析这个块                       │
│              ↓                                               │
│  结合其他规则检测（git、错误模式、测试循环）                   │
│              ↓                                               │
│  熔断器判断是否继续                                          │
└─────────────────────────────────────────────────────────────┘

CCC 的检测机制：
┌─────────────────────────────────────────────────────────────┐
│  Agent 停止时触发 Stop Hook                                  │
│              ↓                                               │
│  调用 Supervisor AI (--fork-session) 审查工作                │
│              ↓                                               │
│  AI 返回 {"completed": bool, "feedback": string}           │
│              ↓                                               │
│  根据结果决定是否继续                                        │
└─────────────────────────────────────────────────────────────┘

AI-First 改进思路：
┌─────────────────────────────────────────────────────────────┐
│  把 Ralph 的检测指南整理到 Supervisor Prompt 中              │
│              ↓                                               │
│  让 Supervisor AI 理解如何检测各种情况                        │
│              ↓                                               │
│  AI 智能判断 + 最小兜底规则（迭代次数）                       │
│              ↓                                               │
│  灵活、可扩展、长期更优                                      │
└─────────────────────────────────────────────────────────────┘
```

### 改进方向

| 方面 | 当前状态 | 改进方向 |
|------|----------|----------|
| **Supervisor Prompt** | 简单审查指令 | 嵌入 Ralph 的检测指南和最佳实践 |
| **检测方式** | 纯 AI 判断 | AI 判断 + 最小兜底规则 |
| **状态可见性** | 仅日志 | 可选的实时监控面板 |
| **退出条件** | 仅迭代次数 | AI 智能判断 + 迭代兜底 |

---

## Ralph 项目深度分析

### 1. Ralph 的"AI + 规则"混合机制

#### 1.1 Ralph 不是"纯规则检测"

经过深入代码分析，Ralph 的检测机制是：

```
┌──────────────────────────────────────────────────────────────┐
│               Ralph 的检测机制（误解 vs 实际）                  │
├──────────────────────────────────────────────────────────────┤
│                                                                │
│  ❌ 常见误解：Ralph 是纯规则检测                                │
│     grep 关键词 → 检测完成                                     │
│     git diff → 检测进展                                       │
│                                                                │
│  ✅ 实际机制：AI 生成结构化输出 + 规则解析                      │
│                                                                │
│  1. PROMPT.md 要求 Agent 输出结构化状态块：                     │
│     ┌──────────────────────────────────────────────────┐      │
│     │ ---RALPH_STATUS---                               │      │
│     │ STATUS: IN_PROGRESS | COMPLETE | BLOCKED         │      │
│     │ TASKS_COMPLETED_THIS_LOOP: <number>              │      │
│     │ FILES_MODIFIED: <number>                         │      │
│     │ TESTS_STATUS: PASSING | FAILING | NOT_RUN        │      │
│     │ WORK_TYPE: IMPLEMENTATION | TESTING | ...        │      │
│     │ EXIT_SIGNAL: false | true                        │      │
│     │ RECOMMENDATION: <one line summary>               │      │
│     │ ---END_RALPH_STATUS---                           │      │
│     └──────────────────────────────────────────────────┘      │
│                                                                │
│  2. response_analyzer.sh 解析这个块（规则）：                    │
│     grep "STATUS:" → 提取状态                                  │
│     grep "EXIT_SIGNAL:" → 提取退出信号                          │
│                                                                │
│  3. 结合其他规则增强检测：                                      │
│     - git diff --name-only → 检测文件变更                      │
│     - error pattern → 检测错误                                 │
│     - test loop detection → 检测测试循环                       │
│                                                                │
│  关键：AI 根据 PROMPT.md 的指导生成结构化输出                    │
│       规则只是解析这个输出，不是"检测"Agent                      │
│                                                                │
└──────────────────────────────────────────────────────────────┘
```

#### 1.2 PROMPT.md 的核心指令结构

```markdown
# Ralph Development Instructions

## 🎯 Status Reporting (CRITICAL - Ralph needs this!)

**IMPORTANT**: At the end of your response, ALWAYS include this status block:

```
---RALPH_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED
TASKS_COMPLETED_THIS_LOOP: <number>
FILES_MODIFIED: <number>
TESTS_STATUS: PASSING | FAILING | NOT_RUN
WORK_TYPE: IMPLEMENTATION | TESTING | DOCUMENTATION | REFACTORING
EXIT_SIGNAL: false | true
RECOMMENDATION: <one line summary of what to do next>
---END_RALPH_STATUS---
```

### When to set EXIT_SIGNAL: true

Set EXIT_SIGNAL to **true** when ALL of these conditions are met:
1. ✅ All items in @fix_plan.md are marked [x]
2. ✅ All tests are passing (or no tests exist for valid reasons)
3. ✅ No errors or warnings in the last execution
4. ✅ All requirements from specs/ are implemented
5. ✅ You have nothing meaningful left to implement
```

**关键发现**：
- Ralph **不是**调用 AI 来做检测
- 而是**教 AI 如何报告自己的状态**
- 然后用规则解析这个状态报告

#### 1.3 Ralph 的检测规则

```bash
# lib/response_analyzer.sh 的核心检测逻辑

# 1. 解析结构化状态块（AI 生成）
if grep -q -- "---RALPH_STATUS---" "$output_file"; then
    status=$(grep "STATUS:" "$output_file" | cut -d: -f2 | xargs)
    exit_sig=$(grep "EXIT_SIGNAL:" "$output_file" | cut -d: -f2 | xargs)

    if [[ "$exit_sig" == "true" || "$status" == "COMPLETE" ]]; then
        has_completion_signal=true
        confidence_score=100
    fi
fi

# 2. 关键词检测（自然语言兜底）
COMPLETION_KEYWORDS=("done" "complete" "finished" "all tasks complete")
for keyword in "${COMPLETION_KEYWORDS[@]}"; do
    if grep -qi "$keyword" "$output_file"; then
        has_completion_signal=true
        ((confidence_score+=10))
        break
    fi
done

# 3. 行为模式检测
test_command_count=$(grep -c -i "running tests\|npm test\|bats" "$output_file")
implementation_count=$(grep -c -i "implementing\|creating\|writing" "$output_file")

if [[ $test_command_count -gt 0 ]] && [[ $implementation_count -eq 0 ]]; then
    is_test_only=true
fi

# 4. 客观进展检测（git）
files_modified=$(git diff --name-only 2>/dev/null | wc -l)
if [[ $files_modified -gt 0 ]]; then
    has_progress=true
    ((confidence_score+=20))
fi

# 5. 错误检测（两阶段过滤避免误报）
error_count=$(
    grep -v '"[^"]*error[^"]*":' "$output_file" |  # 排除JSON字段
    grep -cE '(^Error:|^ERROR:|^error:|...)'         # 匹配实际错误
)
```

### 2. Ralph 的 Tmux 优化

#### 2.1 Tmux 会话设置

```bash
setup_tmux_session() {
    local session_name="ralph-$(date +%s)"

    # 创建新的 tmux 会话
    tmux new-session -d -s "$session_name" -c "$(pwd)"

    # 垂直分割窗口
    tmux split-window -h -t "$session_name" -c "$(pwd)"

    # 右侧面板启动监控
    tmux send-keys -t "$session_name:0.1" "ralph-monitor" Enter

    # 左侧面板启动 Ralph 循环
    tmux send-keys -t "$session_name:0.0" "ralph --monitor" Enter

    # 聚焦左侧面板
    tmux select-pane -t "$session_name:0.0"

    # 附加到会话
    tmux attach-session -t "$session_name"
}
```

#### 2.2 实时监控实现

**监控面板布局**：
```
╔════════════════════════════════════════════════════════════╗
║                    🤖 RALPH MONITOR                         ║
╠════════════════════════════════════════════════════════════╣
║                                                              │
║  ┌─ Current Status ─────────────────────────────────────┐   │
║  │ Loop Count:     #47                                   │   │
║  │ Status:         running                              │   │
║  │ API Calls:      23/100                               │   │
║  └──────────────────────────────────────────────────────┘   │
║                                                              │
║  ┌─ Claude Code Progress ───────────────────────────────┐   │
║  │ Status:         ⠹ Working (230s elapsed)             │   │
║  │ Output:         Building component structure...      │   │
║  └──────────────────────────────────────────────────────┘   │
║                                                              │
║  ┌─ Circuit Breaker ─────────────────────────────────────┐   │
║  │ State:          ✅ CLOSED                            │   │
║  │ No Progress:    0                                   │   │
║  │ Last Progress:  Loop #47                            │   │
║  └──────────────────────────────────────────────────────┘   │
║                                                              │
║  ┌─ Recent Activity ─────────────────────────────────────┐   │
║  │ [2025-01-09 14:23:45] [LOOP] === Starting Loop #47 === │   │
║  │ [2025-01-09 14:23:45] [INFO] ⏳ Starting Claude Code... │   │
║  │ [2025-01-09 14:23:48] [SUCCESS] ✅ Claude Code completed│   │
║  └──────────────────────────────────────────────────────┘   │
║                                                              │
╚════════════════════════════════════════════════════════════╝
```

**数据传递机制**：
1. **status.json** - 主状态文件
2. **progress.json** - 实时进度
3. **logs/ralph.log** - 详细日志
4. 监控面板每 2 秒读取并显示

### 3. Ralph vs CCC 对比

| 维度 | Ralph | CCC |
|------|-------|-----|
| **检测方式** | AI 生成结构化输出 + 规则解析 | Supervisor AI 直接审查 |
| **指令传递** | PROMPT.md 嵌入结构化输出要求 | SUPERVISOR.md 审查指令 |
| **输出格式** | `---RALPH_STATUS---` 块 | `{"completed": bool, "feedback": string}` |
| **规则作用** | 解析 AI 输出 | 仅迭代次数兜底 |
| **实时监控** | Tmux 分屏 + JSON 文件 | 仅日志文件 |
| **进展检测** | git diff + 文件变更 | 无 |
| **错误检测** | 两阶段过滤 | 无 |

---

## 核心洞察与设计原则

### 1. 核心洞察

#### 洞察 1：Ralph 不是"用 AI 做检测"

```
误解：Ralph 调用 AI 来检测 Agent 是否完成
实际：Ralph 教 AI 如何报告状态，然后用规则解析这个报告

区别：
- "用 AI 做检测"：AI 分析 Agent 输出并判断
- "教 AI 报告状态"：AI 按照格式输出自己的判断
```

#### 洞察 2：规则检测的局限性

```
规则检测的问题：
1. 需要预定义所有情况
2. 难以理解上下文
3. 边缘场景难以覆盖
4. 维护成本高（新情况需要改代码）

例如：
- 规则："检测 'done' 关键词"
- 问题：Agent 说 "done waiting for user" 会被误判
```

#### 洞察 3：AI 检测的长期优势

```
AI 检测的优势：
1. 理解上下文和意图
2. 处理边缘情况
3. 随着模型能力提升自动改进
4. 通过 Prompt 更新即可适应新场景

长期来看：
- 规则：固定的模式，无法自我进化
- AI：随着模型升级自动获得更好的判断能力
```

### 2. AI-First 设计原则

#### 原则 1：AI 判断为主，规则为辅

```
优先级：
1. AI Supervisor 智能判断（主要）
2. 迭代次数兜底（防止无限循环）
3. 可选的客观信息传递给 AI（辅助）
```

#### 原则 2：Prompt 工程优于代码工程

```
传统方式：
检测到情况 X → 修改代码添加规则

AI-First：
检测到情况 X → 更新 Supervisor Prompt 教 AI 如何处理
```

#### 原则 3：渐进式增强

```
Phase 1: 增强 Supervisor Prompt
        ← 最快见效，零代码变更

Phase 2: 添加客观信息收集（可选）
        ← git status、文件变更等

Phase 3: 实时监控（可选）
        ← Tmux 集成
```

---

## 改进方案总览

### 方案对比

| 方案 | 核心思路 | 优势 | 劣势 |
|------|----------|------|------|
| **方案 A：纯规则检测** | 实现 Ralph 式的规则检测 | 可预测、成本低 | 不灵活、维护难 |
| **方案 B：AI-First** | 增强 Supervisor Prompt，嵌入检测指南 | 灵活、可扩展、长期更优 | 依赖 AI 质量 |
| **方案 C：混合模式** | AI 判断 + 规则增强 | 平衡 | 复杂度高 |

**推荐：方案 B（AI-First）**

### AI-First 架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                  AI-First Supervisor Architecture                │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │           ENHANCED SUPERVISOR PROMPT (核心)                  │  │
│  │                                                              │  │
│  │  ┌────────────────────────────────────────────────────────┐ │  │
│  │  │ 1. 角色定义                                             │ │  │
│  │  │    "你是一个严格的 Supervisor，负责审查 Agent 工作的     │ │  │
│  │  │     完成度和质量"                                        │ │  │
│  │  └────────────────────────────────────────────────────────┘ │  │
│  │                                                              │  │
│  │  ┌────────────────────────────────────────────────────────┐ │  │
│  │  │ 2. 检测指南（从 Ralph 整理）                            │ │  │
│  │  │    2.1 完成信号检测                                     │ │  │
│  │  │    2.2 进展检测（文件变更、代码实现）                     │ │  │
│  │  │    2.3 常见陷阱（只问不做、测试循环等）                   │ │  │
│  │  │    2.4 错误模式识别                                     │ │  │
│  │  └────────────────────────────────────────────────────────┘ │  │
│  │                                                              │  │
│  │  ┌────────────────────────────────────────────────────────┐ │  │
│  │  │ 3. 判断标准                                             │ │  │
│  │  │    3.1 completed: true 的条件                            │ │  │
│  │  │    3.2 completed: false 的场景                           │ │  │
│  │  │    3.3 feedback 的质量要求                              │ │  │
│  │  └────────────────────────────────────────────────────────┘ │  │
│  │                                                              │  │
│  │  ┌────────────────────────────────────────────────────────┐ │  │
│  │  │ 4. 检查清单                                             │ │  │
│  │  │    □ 是否完成了实际工作？                               │ │  │
│  │  │    □ 是否做了应该自己做的事？                           │ │  │
│  │  │    □ 代码质量是否达标？                                 │ │  │
│  │  │    □ 用户需求是否满足？                                 │ │  │
│  │  │    □ 是否达到了无可挑剔的状态？                         │ │  │
│  │  └────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────┘  │
│                           │                                       │
│                           ▼                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │            MINIMAL FALLBACK RULES (最小兜底)                │  │
│  │  ┌────────────────────────────────────────────────────────┐ │  │
│  │  │ • 迭代次数上限：10 次                                   │ │  │
│  │  │ • 防止 AI 判断失误导致的无限循环                        │ │  │
│  │  └────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────┘  │
│                           │                                       │
│                           ▼                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │          OPTIONAL: OBJECTIVE INFO COLLECTION                │  │
│  │  ┌────────────────────────────────────────────────────────┐ │  │
│  │  │ • git status（传递给 AI 作为上下文）                     │ │  │
│  │  │ • 文件变更统计                                          │ │  │
│  │  │ • 测试结果                                              │ │  │
│  │  └────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────┘  │
│                           │                                       │
│                           ▼                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │             OPTIONAL: REAL-TIME MONITORING                   │  │
│  │  ┌────────────────────────────────────────────────────────┐ │  │
│  │  │ • ccc monitor 命令（Tmux 集成）                         │ │  │
│  │  │ • 实时状态面板                                          │ │  │
│  │  │ • 进度追踪                                              │ │  │
│  │  └────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

---

## 详细设计方案

### 1. 增强 Supervisor Prompt

#### 1.1 当前 SUPERVISOR.md 分析

```markdown
# 任务：严格审查当前执行的工作并给出反馈意见

你是一个无比严格的 Supervisor，负责审查当前执行的工作，判断任务是否真正完成...

## 核心原则
**你的职责是检查是否完成了实际工作，检查是否把能做/该做的事情都做好了...

## 审查要点
1. **是否完成了实际工作？**
2. **是否做了应该自己做的事？**
3. **代码质量**
4. **任务完整性**
5. **无可挑剔**

## 判断标准
### completed: true
- Agent 完成了实际工作
- 测试已运行且通过
- 用户需求已满足
- 把能做/该做的事情都做好了，无可挑剔

### completed: false
- Agent 在等待用户确认
- Agent 问了应该自己解决的问题
- 测试未运行或未通过
- 任务未完成
```

**问题**：
- 缺少具体的检测指南
- 没有常见陷阱的识别方法
- 缺少对边缘情况的指导

#### 1.2 增强后的 SUPERVISOR.md

```markdown
# Supervisor: 任务完成度审查与反馈

你是一个严格的任务审查者，负责判断 Agent 的工作是否真正完成，是否达到了能够交付的状态。

## 核心职责

你的职责是：
1. **审查工作完成度**：判断任务是否真正完成，而不是把问题抛回给用户
2. **识别常见陷阱**：检测 Agent 是否陷入了常见的无效循环
3. **提供具体反馈**：当任务未完成时，给出明确的改进方向

## 审查框架

### 第一步：理解用户需求

首先，回顾用户的原始需求：
- 用户最初要求做什么？
- 是否有明确的目标或交付标准？
- 用户是否提到了任何约束或偏好？

### 第二步：检查实际工作

**关键问题**：Agent 是否在做事，还是只是在"思考"和"提问"？

**只问不做的信号**（如果出现这些，completed 应为 false）：
- 只是在问用户"是否要执行 X"、"如何处理 Y"
- 说"让我了解"、"需要更多信息"但没有任何实质行动
- 列出计划但没有执行
- 问"是否运行测试"这类应该自己决定的事

**实际工作的信号**：
- 创建/修改了文件
- 运行了测试或构建
- 分析了代码或配置
- 执行了具体的命令

### 第三步：检查常见陷阱

#### 陷阱 1：只问不做

**表现**：
- Agent 说"我应该做 X 吗？"
- Agent 问"最佳方式是什么？"然后等待回答
- Agent 列出多个选项让用户选择

**判断**：如果 Agent 只提问而没有执行任何实质性工作 → `completed: false`

**反馈**："请直接执行你认为最佳的操作，不要等待确认。"

#### 陷阱 2：测试循环

**表现**：
- 连续多次循环只运行测试，不做任何实现
- Agent 说"运行测试"但没有任何代码变更

**判断**：如果连续看到只测试无实现的模式 → `completed: false`

**反馈**："测试循环检测：请继续实现功能，不要只在测试中循环。"

#### 陷阱 3：计划而不执行

**表现**：
- Agent 说"我的计划是..."
- Agent 列出详细步骤但没有执行第一步

**判断**：如果只有计划没有行动 → `completed: false`

**反馈**："不要只列出计划，请立即开始执行第一步。"

#### 陷阱 4：虚假完成

**表现**：
- Agent 说"完成了"但实际上什么都没改
- Agent 说"ready"但没有任何实质性工作

**判断**：如果声称完成但无实际工作 → `completed: false`

**反馈**："你声称完成，但没有看到任何实质性工作，请继续完成任务。"

### 第四步：评估完成质量

即使 Agent 做了工作，也要检查质量：

**代码质量检查**：
- 代码是否完整，没有 TODO 或占位符？
- 是否有明显的 bug 或错误？
- 是否处理了边界情况？
- 是否有必要的错误处理？

**任务完整性检查**：
- 用户的原始需求是否全部满足？
- 是否有遗漏的功能或要求？
- 是否需要测试但未测试？

**可交付性检查**：
- 如果这是给用户的交付物，用户能直接使用吗？
- 是否需要用户自己"补一刀"？
- 是否有未解决的问题被跳过？

### 第五步：判断完成状态

#### `completed: true` 的条件

**必须同时满足**：
1. ✅ 完成了实际工作（不是只问/计划）
2. ✅ 工作质量达标（无明显 bug/TODO）
3. ✅ 用户需求全部满足
4. ✅ 如果需要测试，测试已运行
5. ✅ 结果可以直接交付，不需要用户补做

#### `completed: false` 的场景

**任何以下情况**：
- ❌ 只在提问/计划，没有实际执行
- ❌ 陷入了测试循环或其他无效循环
- ❌ 声称完成但无实质性工作
- ❌ 有明显的 bug、TODO 或错误未处理
- ❌ 用户需求未全部满足
- ❌ 需要测试但未测试
- ❌ 把应该自己做的事推给用户

### 第六步：提供反馈

当 `completed: false` 时，feedback 必须：

1. **具体指出问题**：不要笼统地说"继续完成"
2. **给出改进方向**：告诉 Agent 下一步应该做什么
3. **避免循环**：不要让 Agent 反复问相同的问题

**好的 feedback 示例**：
- "你只列出了计划但没有执行。请立即开始实现第一个功能。"
- "代码中有 TODO 标记，请完成所有待办事项。"
- "你声称完成了，但 tests 目录是空的。请添加测试。"
- "不要问'是否运行测试'，直接运行必要的测试。"

**不好的 feedback 示例**：
- "继续完成"（太笼统）
- "做得还不够"（没有方向）
- "请改进"（没有具体建议）

## 客观信息（可选）

如果提供了 git status 或文件变更信息：
- 使用这些信息验证 Agent 声称的工作
- 如果 Agent 说"修改了 X 文件"但 git 没显示，这可能是个问题
- 文件数量很少但声称"完成了复杂任务"需要警惕

## 输出格式

调用 StructuredOutput 工具提供 JSON 结果：
```json
{
  "completed": boolean,
  "feedback": "string"
}
```

提交后立即停止，不要做任何其他工作。
```

### 2. 最小兜底规则

#### 2.1 保留现有机制

当前的迭代次数限制已经足够：

```go
// internal/supervisor/state.go
const DefaultMaxIterations = 10

func ShouldContinue(sessionID string, max int) (bool, int, error) {
    count, err := GetCount(sessionID)
    if err != nil {
        return false, 0, err
    }
    return count < max, count, nil
}
```

**不需要添加**：
- ❌ 复杂的规则检测
- ❌ 熔断器状态机
- ❌ 信号历史追踪

**保留**：
- ✅ 迭代次数上限（防止无限循环）
- ✅ 状态文件记录（用于调试）

### 3. 可选：客观信息收集

如果需要给 AI 提供更多上下文，可以轻量级地收集一些客观信息：

```go
// internal/cli/hook.go

// collectObjectiveInfo 收集客观信息作为 AI 上下文
func collectObjectiveInfo(workingDir string) map[string]interface{} {
    info := make(map[string]interface{})

    // 1. Git 状态（如果可用）
    if gitStatus, err := getGitStatus(workingDir); err == nil {
        info["git_status"] = gitStatus
    }

    // 2. 文件变更统计
    if fileChanges, err := getFileChanges(workingDir); err == nil {
        info["file_changes"] = fileChanges
    }

    return info
}

// getGitStatus 获取 git 状态摘要
func getGitStatus(workingDir string) (map[string]interface{}, error) {
    cmd := exec.Command("git", "status", "--porcelain")
    cmd.Dir = workingDir

    var out bytes.Buffer
    cmd.Stdout = &out

    if err := cmd.Run(); err != nil {
        return nil, err
    }

    lines := strings.Split(out.String(), "\n")
    modified := 0
    added := 0

    for _, line := range lines {
        if line == "" {
            continue
        }
        if len(line) > 0 && line[0] == 'M' {
            modified++
        } else if len(line) > 0 && line[1] == 'A' {
            added++
        }
    }

    return map[string]interface{}{
        "modified_files": modified,
        "added_files":    added,
        "total_changes":  modified + added,
    }, nil
}
```

**使用方式**：将这些信息附加到 Supervisor Prompt 的末尾：

```markdown
## 客观信息

Git 状态：
- 修改文件：5
- 新增文件：2
- 总变更：7

（这些信息可以帮助验证 Agent 声称的工作）
```

### 4. 可选：实时监控

#### 4.1 ccc monitor 命令

```bash
#!/bin/bash
# ccc-monitor - Supervisor Mode 实时监控

STATUS_FILE="$HOME/.claude/ccc/monitor-status.json"
LOG_FILE="$HOME/.claude/ccc/supervisor.log"

display_monitor() {
    while true; do
        clear

        echo -e "${WHITE}╔════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${WHITE}║                      CCC SUPERVISOR MONITOR                   ║${NC}"
        echo -e "${WHITE}╚════════════════════════════════════════════════════════════╝${NC}"
        echo ""

        # 解析状态文件
        if [[ -f "$STATUS_FILE" ]]; then
            local session_id=$(jq -r '.session_id // "N/A"' "$STATUS_FILE")
            local iteration=$(jq -r '.iteration // 0' "$STATUS_FILE")
            local status=$(jq -r '.status // "unknown"' "$STATUS_FILE")

            echo -e "${YELLOW}┌─ Current Session ───────────────────────────────────────┐${NC}"
            echo -e "${YELLOW}│${NC} Session ID:    $session_id"
            echo -e "${YELLOW}│${NC} Iteration:     $iteration/10"
            echo -e "${YELLOW}│${NC} Status:        $status"
            echo -e "${YELLOW}└──────────────────────────────────────────────────────────┘${NC}"
            echo ""
        fi

        # 显示最近日志
        if [[ -f "$LOG_FILE" ]]; then
            echo -e "${BLUE}┌─ Recent Activity ────────────────────────────────────────┐${NC}"
            tail -n 10 "$LOG_FILE" | while IFS= read -r line; do
                echo -e "${BLUE}│${NC} $line"
            done
            echo -e "${BLUE}└──────────────────────────────────────────────────────────┘${NC}"
        fi

        sleep 2
    done
}

display_monitor
```

#### 4.2 Tmux 集成（可选）

```bash
# ccc monitor --tmux

setup_tmux_monitor() {
    local session_name="ccc-$(date +%s)"

    tmux new-session -d -s "$session_name"
    tmux split-window -h -t "$session_name"

    # 左侧：执行 ccc
    tmux send-keys -t "$session_name:0.0" "ccc kimi" Enter

    # 右侧：监控
    tmux send-keys -t "$session_name:0.1" "ccc-monitor" Enter

    tmux select-pane -t "$session_name:0.0"
    tmux attach-session -t "$session_name"
}
```

---

## 实施计划

### 阶段划分

```
┌─────────────────────────────────────────────────────────────────┐
│                    Implementation Roadmap                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Phase 1: Enhance Supervisor Prompt (核心改进)                   │
│  ├─ 重写 SUPERVISOR.md                                           │
│  ├─ 嵌入 Ralph 的检测指南                                        │
│  ├─ 添加常见陷阱识别                                             │
│  └─ 验证改进效果                                                 │
│  预计时间：1-2 天                                                │
│                                                                  │
│  Phase 2: Optional Enhancements (可选增强)                       │
│  ├─ 客观信息收集（git status）                                   │
│  ├─ 改进日志输出                                                 │
│  └─ 状态文件优化                                                 │
│  预计时间：1-2 天                                                │
│                                                                  │
│  Phase 3: Real-time Monitoring (可选)                            │
│  ├─ ccc-monitor 命令                                             │
│  ├─ Tmux 集成                                                    │
│  └─ 实时状态面板                                                 │
│  预计时间：2-3 天                                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 详细任务列表

#### Phase 1: 增强 Supervisor Prompt

| 任务 | 优先级 | 预估时间 | 依赖 |
|------|--------|----------|------|
| 整理 Ralph 的检测指南 | P0 | 2h | - |
| 重写 SUPERVISOR.md | P0 | 3h | 检测指南整理 |
| 添加常见陷阱识别 | P0 | 2h | - |
| 实际测试验证效果 | P0 | 4h | 以上全部 |
| 根据测试迭代优化 | P1 | 2h | 测试结果 |

#### Phase 2: 可选增强

| 任务 | 优先级 | 预估时间 | 依赖 |
|------|--------|----------|------|
| 实现客观信息收集 | P1 | 3h | - |
| 集成到 Hook 执行 | P1 | 2h | 信息收集 |
| 改进日志格式 | P2 | 1h | - |

#### Phase 3: 实时监控

| 任务 | 优先级 | 预估时间 | 依赖 |
|------|--------|----------|------|
| 实现 ccc-monitor | P2 | 4h | - |
| Tmux 集成 | P2 | 3h | monitor |
| 文档更新 | P1 | 1h | 以上全部 |

### 里程碑

| 里程碑 | 交付物 | 完成标准 |
|--------|--------|----------|
| M1: Prompt Enhanced | 增强的 SUPERVISOR.md | 测试显示更好的判断 |
| M2: Optional Features | 客观信息收集 | 可选功能可用 |
| M3: Monitoring | ccc-monitor 命令 | 实时监控可用 |

---

## 附录

### A. SUPERVISOR.md 完整模板

见前面"详细设计方案"第 1.2 节。

### B. 检测指南速查表

| 陷阱 | 信号 | 判断 | 反馈 |
|------|------|------|------|
| 只问不做 | 只提问无执行 | completed: false | "请直接执行，不要等待确认" |
| 测试循环 | 只测试无实现 | completed: false | "请继续实现功能" |
| 计划不执行 | 只列出计划 | completed: false | "请立即开始执行第一步" |
| 虚假完成 | 声称完成无工作 | completed: false | "请继续完成任务" |

### C. 与 Ralph 的最终对比

| 方面 | Ralph | CCC (AI-First 改进后) |
|------|-------|----------------------|
| 检测方式 | AI 生成结构化输出 + 规则解析 | AI Supervisor 直接判断 |
| 指令传递 | PROMPT.md 嵌入输出格式要求 | SUPERVISOR.md 嵌入检测指南 |
| 灵活性 | 中（需要更新规则） | 高（更新 Prompt 即可） |
| 长期可维护性 | 低（规则堆砌） | 高（AI 随模型进化） |
| 实时监控 | 有（Tmux） | 可选 |

### D. 参考资源

- Ralph 项目：https://github.com/anthropics/ralph-claude-code
- Claude Code Hooks 文档：https://docs.anthropic.com/en/docs/build-with-claude/claude-for-developers
- 当前项目：https://github.com/guyskk/claude-code-supervisor

---

## 总结

本改进方案采用 **AI-First** 的设计理念：

1. **核心改进**：增强 Supervisor Prompt，嵌入 Ralph 的检测指南
2. **最小兜底**：保留迭代次数限制，防止 AI 判断失误
3. **可选增强**：客观信息收集、实时监控面板

**优势**：
- ✅ 灵活：通过 Prompt 更新即可适应新场景
- ✅ 可扩展：AI 随着模型能力提升自动改进
- ✅ 简洁：无需实现复杂的规则系统
- ✅ 长期优：AI 判断长期来看优于固定规则

**实施路径**：
- Phase 1（核心）：增强 SUPERVISOR.md → 立即见效
- Phase 2（可选）：客观信息收集 → 提供更多上下文
- Phase 3（可选）：实时监控 → 更好的用户体验
