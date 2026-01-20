# 功能规格：Supervisor Hook 支持 AskUserQuestion 工具调用审查

**功能分支**: `002-askuserquestion-hook`
**创建日期**: 2026-01-20
**状态**: 草稿
**输入**: 用户描述："Supervisor hook 支持 AskUserQuestion 工具调用审查。当 Claude Code 调用 AskUserQuestion 工具时，supervisor hook 也应该进行审查。根据 allow_stop 决定是 deny 还是 allow，在 permissionDecisionReason 字段填写 feedback。需要扩展输出格式支持 PreToolUse 的决策控制，输入格式也需要扩展，支持 tool_name、hook_event_name 字段。Supervisor prompt 保持不变，迭代计数应该增加。"

## 特别说明：使用中文

**本文档必须使用中文编写。**

1. 所有用户故事、需求描述、验收场景必须使用中文。
2. 用户场景描述应该使用自然、易懂的中文。
3. 功能需求使用中文描述，技术术语保留英文。

## 用户场景与测试 *(必填)*

### 用户故事 1 - Supervisor 审查 AskUserQuestion 调用 (优先级: P1)

当 Claude Code 准备向用户提问时，Supervisor 能够审查这个问题是否合理，并在必要时阻止或允许这次提问。

**为什么是这个优先级**: 这是核心功能，确保 Supervisor 不仅在任务结束时审查，也能在任务执行过程中对关键交互（向用户提问）进行质量控制，保持审查的一致性。

**独立测试**: 可以通过启用 Supervisor 模式后，触发一个会导致 Claude Code 调用 AskUserQuestion 的场景，验证是否正确触发审查并根据审查结果允许或阻止提问。

**验收场景**:

1. **给定** Supervisor 模式已启用，**当** Claude Code 准备调用 AskUserQuestion 工具，**那么** Supervisor hook 应被触发，审查这次提问
2. **给定** Supervisor 决定阻止提问（allow_stop=false），**当** PreToolUse hook 返回 deny 决策，**那么** Claude Code 应该取消这次提问并收到反馈
3. **给定** Supervisor 决定允许提问（allow_stop=true），**当** PreToolUse hook 返回 allow 决策，**那么** Claude Code 应该正常执行 AskUserQuestion 调用

---

### 用户故事 2 - 扩展输入输出格式支持 (优先级: P1)

Supervisor hook 需要能够识别不同的 hook 事件类型（Stop vs PreToolUse），并根据事件类型返回正确的决策格式。

**为什么是这个优先级**: 这是技术基础，没有正确的输入输出格式支持，Supervisor 无法区分不同事件并返回相应决策。

**独立测试**: 可以通过模拟不同 hook 事件的输入（Stop 和 PreToolUse），验证 hook 命令能正确解析输入并返回对应格式的输出。

**验收场景**:

1. **给定** hook 输入包含 `tool_name` 和 `hook_event_name` 字段，**当** Supervisor hook 处理 PreToolUse 事件，**那么** 输出应包含 `permissionDecision` 和 `permissionDecisionReason` 字段
2. **给定** hook 输入是 Stop 事件格式，**当** Supervisor hook 处理 Stop 事件，**那么** 输出应保持现有格式（`decision` 和 `reason` 字段）

---

### 用户故事 3 - 迭代计数一致性 (优先级: P2)

无论 hook 事件类型是 Stop 还是 PreToolUse，都应该计入迭代计数，防止无限循环。

**为什么是这个优先级**: 这是保护机制，确保 Supervisor 不会因为审查点增多而导致无限循环。

**独立测试**: 可以通过多次触发不同类型的 hook 事件，验证迭代计数是否正确递增并在达到上限时停止。

**验收场景**:

1. **给定** 最大迭代次数为 20，**当** AskUserQuestion hook 被触发，**那么** 迭代计数应增加 1
2. **给定** 迭代计数已达上限，**当** 任何 hook 事件被触发，**那么** 应自动允许操作并停止审查

---

### 边缘情况

- 当 AskUserQuestion hook 返回的决策格式不正确时，系统如何处理？
- 当 hook 输入中缺少 tool_name 或 hook_event_name 字段时，系统如何识别事件类型？
- 当 Supervisor 在 AskUserQuestion hook 中被递归调用时（例如 Supervisor 本身需要提问），如何防止无限循环？
- 当 PreToolUse hook 超时时，Claude Code 的默认行为是什么？

## 需求 *(必填)*

### 功能需求

- **FR-001**: 系统必须在 Claude Code 配置中添加 PreToolUse hook，匹配 AskUserQuestion 工具
- **FR-002**: 系统必须扩展 hook 输入解析，支持 `tool_name` 和 `hook_event_name` 字段
- **FR-003**: 系统必须根据 `allow_stop` 决定返回 "allow" 或 "deny" 决策
- **FR-004**: 系统必须在 `permissionDecisionReason` 字段中填写 feedback 内容
- **FR-005**: 系统必须在 PreToolUse hook 触发时增加迭代计数
- **FR-006**: 系统必须保持 Supervisor prompt 不变
- **FR-007**: 系统必须支持向后兼容，Stop hook 事件继续使用现有格式

### 核心实体

- **Hook 输入结构**: 包含 session_id、tool_name（PreToolUse 特有）、hook_event_name（PreToolUse 特有）、tool_input（PreToolUse 特有）等字段
- **Hook 输出结构 (PreToolUse)**: 包含 permissionDecision（"allow"/"deny"）、permissionDecisionReason、hookSpecificOutput 等字段
- **Hook 输出结构 (Stop)**: 保持现有格式，包含 decision（"block"/undefined）、reason 字段

## 成功标准 *(必填)*

### 可衡量的结果

- **SC-001**: AskUserQuestion hook 触发时，Supervisor 审查响应时间在 30 秒内完成
- **SC-002**: 100% 的 AskUserQuestion 调用都能正确触发 Supervisor 审查（当 Supervisor 模式启用时）
- **SC-003**: PreToolUse hook 返回的决策格式符合 Claude Code 规范，能够正确控制工具调用
- **SC-004**: 迭代计数在所有 hook 事件类型中保持一致，不会因为增加审查点而导致无限循环
- **SC-005**: 现有 Stop hook 功能不受影响，继续正常工作

## 需求完整性检查

在继续到实现方案 (`/speckit.plan`) 之前，验证：

- [x] 没有 `[需要澄清]` 标记残留
- [x] 所有需求都可测试且无歧义
- [x] 成功标准可衡量
- [x] 每个用户故事都可独立实现和测试
- [x] 边缘情况已考虑
- [x] 与宪章原则一致（单二进制、跨平台、向后兼容）
