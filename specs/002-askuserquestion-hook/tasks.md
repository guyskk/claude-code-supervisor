# 任务清单：Supervisor Hook 支持 AskUserQuestion 工具调用审查

**输入**: 来自 `/specs/002-askuserquestion-hook/` 的设计文档
**前置条件**: plan.md（已完成）、spec.md（已完成）、research.md（已完成）、data-model.md（已完成）、contracts/hook-input-output.md（已完成）

**测试**: 本项目遵循测试先行的 TDD 方法，包含单元测试和集成测试。

**组织方式**: 任务按用户故事分组，以便独立实现和测试每个故事。

## 格式：`[ID] [P?] [故事] 描述`

- **[P]**: 可并行运行（不同文件，无依赖）
- **[故事]**: 此任务属于哪个用户故事（例如：US1、US2、US3）
- 在描述中包含确切的文件路径

## 特别说明：使用中文

**本文档必须使用中文编写。**

1. 所有任务描述必须使用中文。
2. 文件路径使用英文，但说明文字使用中文。
3. 变量名、函数名等标识符使用英文。

---

## 第 1 阶段：基础设施（共享）

**目的**: 项目已经存在，本阶段确保开发环境就绪

- [ ] T001 确认开发分支 `002-askuserquestion-hook` 已创建并切换
- [ ] T002 确认 Go 版本 >= 1.23，运行 `go version`
- [ ] T003 [P] 运行 `go mod tidy` 确保依赖完整

---

## 第 2 阶段：基础（阻塞前置条件）

**目的**: 扩展数据结构以支持所有 hook 事件类型

**⚠️ 关键**: 在此阶段完成前不能开始任何用户故事工作

- [ ] T004 [P] [US2] 扩展 `HookInput` 结构支持所有 hook 事件类型，位于 `internal/cli/hook.go`
- [ ] T005 [P] [US2] 添加 `HookOutput` 结构根据事件类型返回不同格式，位于 `internal/cli/hook.go`
- [ ] T006 [P] [US2] 添加 `HookSpecificOutput` 结构用于 PreToolUse 输出，位于 `internal/cli/hook.go`
- [ ] T007 [US2] 保留 `StopHookInput` 作为 `HookInput` 的别名确保向后兼容，位于 `internal/cli/hook.go`
- [ ] T008 [US2] 实现 `SupervisorResultToHookOutput` 转换函数，位于 `internal/cli/hook.go`

**检查点**: 数据结构扩展完成 - 可以开始用户故事实现

---

## 第 3 阶段：用户故事 1 - Supervisor 审查 AskUserQuestion 调用 (优先级: P1) 🎯 MVP

**目标**: 在 Claude Code 配置中添加 PreToolUse hook，使 Supervisor 能审查 AskUserQuestion 工具调用

**独立测试**: 启用 Supervisor 模式后，触发 AskUserQuestion 调用，验证是否正确触发审查并根据审查结果允许或阻止提问

### 用户故事 1 的测试 ⚠️

> **注意：先编写这些测试，确保它们在实现前失败**

- [ ] T009 [P] [US1] 测试 Stop 事件输入解析，位于 `internal/cli/hook_test.go`
- [ ] T010 [P] [US1] 测试 PreToolUse 事件输入解析，位于 `internal/cli/hook_test.go`
- [ ] T011 [P] [US1] 测试缺少 hook_event_name 字段时默认为 Stop，位于 `internal/cli/hook_test.go`
- [ ] T012 [P] [US1] 测试 SupervisorResult 转换为 Stop 事件输出，位于 `internal/cli/hook_test.go`
- [ ] T013 [P] [US1] 测试 SupervisorResult 转换为 PreToolUse 事件输出（allow），位于 `internal/cli/hook_test.go`
- [ ] T014 [P] [US1] 测试 SupervisorResult 转换为 PreToolUse 事件输出（deny），位于 `internal/cli/hook_test.go`
- [ ] T015 [P] [US1] 测试向后兼容 - 旧格式输入能正确解析，位于 `internal/cli/hook_test.go`
- [ ] T016 [P] [US1] 测试向后兼容 - 旧格式输出保持不变，位于 `internal/cli/hook_test.go`
- [ ] T017 [P] [US1] 集成测试：完整 PreToolUse hook 流程，位于 `internal/cli/hook_integration_test.go`

### 用户故事 1 的实现

- [ ] T018 [US1] 修改 `RunSupervisorHook` 函数扩展输入解析，支持识别事件类型，位于 `internal/cli/hook.go`
- [ ] T019 [US1] 修改 `RunSupervisorHook` 函数扩展输出格式，根据事件类型返回对应格式，位于 `internal/cli/hook.go`
- [ ] T020 [P] [US1] 在 `provider.go` 的 `SwitchWithHook` 函数中添加 PreToolUse hook 配置，位于 `internal/provider/provider.go`

**检查点**: 此时用户故事 1 应完全功能化且可独立测试 - AskUserQuestion 调用能被 Supervisor 审查

---

## 第 4 阶段：用户故事 2 - 扩展输入输出格式支持 (优先级: P1)

**目标**: 确保 Supervisor hook 能正确识别不同事件类型并返回对应格式

**独立测试**: 模拟不同 hook 事件的输入（Stop 和 PreToolUse），验证 hook 命令能正确解析输入并返回对应格式的输出

**注意**: 用户故事 2 的实现任务已在第 2 阶段完成（数据结构扩展）。此阶段主要进行验证和测试。

- [ ] T021 [P] [US2] 验证 Stop 事件输出格式符合规范，位于 `internal/cli/hook_test.go`
- [ ] T022 [P] [US2] 验证 PreToolUse 事件输出格式符合规范，位于 `internal/cli/hook_test.go`

**检查点**: 此时用户故事 1 和 2 都应独立工作 - 输入输出格式正确支持两种事件类型

---

## 第 5 阶段：用户故事 3 - 迭代计数一致性 (优先级: P2)

**目标**: 确保所有 hook 事件类型都正确增加迭代计数，防止无限循环

**独立测试**: 多次触发不同类型的 hook 事件，验证迭代计数是否正确递增并在达到上限时停止

**注意**: 当前实现已在 SDK 调用前增加迭代计数（第 134-157 行），因此无需修改代码，只需验证。

- [ ] T023 [P] [US3] 验证 PreToolUse 事件触发时迭代计数正确递增，位于 `internal/cli/hook_test.go`
- [ ] T024 [P] [US3] 验证迭代计数达上限时自动允许操作，位于 `internal/cli/hook_test.go`

**检查点**: 所有用户故事现在都应独立功能化 - 迭代计数在所有事件类型中保持一致

---

## 第 6 阶段：完善与横切关注点

**目的**: 代码质量检查和文档更新

- [ ] T025 [P] 运行 `gofmt` 格式化所有修改的 Go 文件
- [ ] T026 [P] 运行 `go vet ./...` 检查代码静态问题
- [ ] T027 运行 `go test ./... -v` 执行所有测试
- [ ] T028 运行 `go test ./... -race` 检查竞态条件
- [ ] T029 [P] 运行 `./check.sh --lint` 执行完整 lint 检查
- [ ] T030 更新 CHANGELOG.md（如需要）

---

## 依赖关系与执行顺序

### 阶段依赖

- **基础设施（第 1 阶段）**: 无依赖 - 可立即开始
- **基础（第 2 阶段）**: 依赖基础设施完成 - 阻塞所有用户故事
- **用户故事 1（第 3 阶段）**: 依赖基础阶段完成（T004-T008）
- **用户故事 2（第 4 阶段）**: 依赖基础阶段和用户故事 1 完成
- **用户故事 3（第 5 阶段）**: 依赖基础阶段和用户故事 1 完成
- **完善（第 6 阶段）**: 依赖所有用户故事完成

### 用户故事依赖

- **用户故事 1 (P1)**: 依赖基础阶段（T004-T008）完成 - 无其他故事依赖
- **用户故事 2 (P1)**: 与用户故事 1 共享基础阶段实现，主要验证功能
- **用户故事 3 (P2)**: 依赖基础阶段完成，验证现有迭代计数逻辑

### 每个用户故事内

- 测试必须先编写并在实现前失败（TDD）
- 数据结构定义在转换函数之前
- 转换函数在主逻辑修改之前
- 主逻辑修改在配置生成之前

### 并行机会

- 第 1 阶段所有标记为 [P] 的任务可并行运行
- 第 2 阶段所有标记为 [P] 的任务可并行运行（不同数据结构定义）
- 用户故事 1 的所有测试（T009-T017）可并行运行
- 第 4、5 阶段的验证任务可并行运行
- 第 6 阶段所有标记为 [P] 的任务可并行运行

---

## 并行示例：用户故事 1 测试

```bash
# 一起启动用户故事 1 的所有测试（测试先行）：
Task: "测试 Stop 事件输入解析，位于 internal/cli/hook_test.go"
Task: "测试 PreToolUse 事件输入解析，位于 internal/cli/hook_test.go"
Task: "测试缺少 hook_event_name 字段时默认为 Stop，位于 internal/cli/hook_test.go"
Task: "测试 SupervisorResult 转换为 Stop 事件输出，位于 internal/cli/hook_test.go"
Task: "测试 SupervisorResult 转换为 PreToolUse 事件输出（allow），位于 internal/cli/hook_test.go"
Task: "测试 SupervisorResult 转换为 PreToolUse 事件输出（deny），位于 internal/cli/hook_test.go"
Task: "测试向后兼容 - 旧格式输入能正确解析，位于 internal/cli/hook_test.go"
Task: "测试向后兼容 - 旧格式输出保持不变，位于 internal/cli/hook_test.go"
Task: "集成测试：完整 PreToolUse hook 流程，位于 internal/cli/hook_integration_test.go"
```

---

## 并行示例：第 2 阶段数据结构

```bash
# 一起启动第 2 阶段的所有数据结构任务：
Task: "扩展 HookInput 结构支持所有 hook 事件类型，位于 internal/cli/hook.go"
Task: "添加 HookOutput 结构根据事件类型返回不同格式，位于 internal/cli/hook.go"
Task: "添加 HookSpecificOutput 结构用于 PreToolUse 输出，位于 internal/cli/hook.go"
```

---

## 实施策略

### MVP 优先（用户故事 1）

1. 完成第 1 阶段：基础设施（T001-T003）
2. 完成第 2 阶段：基础（T004-T008）
3. 完成第 3 阶段：用户故事 1（T009-T020）
4. **停止并验证**: 独立测试用户故事 1
5. 运行 `./check.sh` 验证代码质量

### 增量交付

1. 完成基础设施 + 基础 → 数据结构就绪
2. 添加用户故事 1 → AskUserQuestion 审查功能 → 验证（MVP！）
3. 添加用户故事 2 验证 → 确认输入输出格式正确
4. 添加用户故事 3 验证 → 确认迭代计数一致性
5. 完善阶段 → 代码质量检查和文档更新

### 单人执行顺序

由于是单人开发，建议按顺序执行：

1. 第 1 阶段 → 第 2 阶段 → 第 3 阶段 → 第 4 阶段 → 第 5 阶段 → 第 6 阶段
2. 在每个阶段内，先执行测试任务，再执行实现任务
3. 利用并行机会同时运行多个测试（T009-T017 可并行运行）

---

## 注意事项

- [P] 任务 = 不同文件，无依赖，可并行执行
- [故事] 标签将任务映射到特定用户故事以实现可追溯性
- 用户故事 1 和 2 共享基础阶段实现（数据结构扩展）
- 用户故事 2 和 3 主要是验证任务，核心实现在用户故事 1
- 实现前验证测试失败（TDD）
- 每完成一个任务或逻辑组后提交代码
- 在任何检查点停止以独立验证故事
- 所有修改遵循 Go 代码规范和项目宪章要求

---

## 文件修改清单

### 主要修改文件

1. **`internal/cli/hook.go`** - 扩展数据结构和输入输出逻辑
2. **`internal/provider/provider.go`** - 添加 PreToolUse hook 配置
3. **`internal/cli/hook_test.go`** - 单元测试（新增或修改）

### 预期代码量

- 新增代码：约 200-300 行（数据结构、转换函数、测试）
- 修改代码：约 50-100 行（输入解析、输出格式、配置生成）
- 总计：约 300-400 行代码变更
