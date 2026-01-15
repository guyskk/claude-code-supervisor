# 任务清单：Claude 命令替换（Patch）

**输入**: 来自 `/specs/001-claude-patch/` 的设计文档
**前置条件**: plan.md、spec.md、research.md、data-model.md、contracts/cli.md

**组织方式**: 任务按用户故事分组，以便独立实现和测试每个故事。

## 格式：`[ID] [P?] [故事] 描述`

- **[P]**: 可并行运行（不同文件，无依赖）
- **[故事]**: 此任务属于哪个用户故事（例如：US1、US2、US3、US4）
- 在描述中包含确切的文件路径

---

## 第 1 阶段：基础设施（共享）

**目的**: 项目已有基础结构，本阶段只需确保准备就绪

- [X] T001 确认 Go 1.21+ 环境配置正确
- [X] T002 确认项目结构符合 plan.md 定义
- [X] T003 运行 `./check.sh --lint` 确保代码质量工具正常

**检查点**: 基础环境就绪 - 可以开始实现

---

## 第 2 阶段：基础（阻塞前置条件）

**目的**: 添加 patch 命令的基础设施

**⚠️ 关键**: 在此阶段完成前不能开始任何用户故事工作

- [X] T004 在 `internal/cli/cli.go` 中添加 Patch 命令结构体 `PatchCommandOptions`
- [X] T005 在 `internal/cli/cli.go` 中添加 `Patch bool` 和 `PatchOpts *PatchCommandOptions` 字段到 `Command` 结构
- [X] T006 在 `internal/cli/cli.go` 的 `Parse` 函数中添加 "patch" 子命令解析逻辑
- [X] T007 在 `internal/cli/cli.go` 中添加 `parsePatchArgs` 函数解析 --reset 标志
- [X] T008 在 `internal/cli/cli.go` 的 `Run` 函数中添加 patch 命令路由调用

**检查点**: patch 命令框架就绪 - 可以开始用户故事实现

---

## 第 3 阶段：用户故事 1 - 启用 Claude 命令替换 (优先级: P1) 🎯 MVP

**目标**: 用户执行 `sudo ccc patch` 后，系统中的 claude 命令被 ccc 替代

**独立测试**: 执行 `sudo ccc patch`，然后运行 `claude --help` 验证显示 ccc 帮助信息

### 用户故事 1 的实现

- [X] T009 [US1] 创建 `internal/cli/patch.go` 文件，添加 `PatchCommandOptions` 结构体定义
- [X] T010 [US1] 在 `internal/cli/patch.go` 中实现 `findClaudePath()` 函数，使用 `exec.LookPath` 查找 claude
- [X] T011 [US1] 在 `internal/cli/patch.go` 中实现 `checkAlreadyPatched()` 函数，检测 ccc-claude 是否存在
- [X] T012 [US1] 在 `internal/cli/patch.go` 中实现 `createWrapperScript()` 函数，创建 sh 包装脚本
- [X] T013 [US1] 在 `internal/cli/patch.go` 中实现 `rollbackPatch()` 函数，用于 patch 失败时回滚
- [X] T014 [US1] 在 `internal/cli/patch.go` 中实现 `applyPatch()` 函数，执行 patch 操作（重命名 + 创建脚本）
- [X] T015 [US1] 在 `internal/cli/patch.go` 中实现 `RunPatch()` 入口函数，处理 patch 和 reset 分发
- [X] T016 [US1] 在 `internal/cli/cli.go` 中添加 `RunPatch` 调用，返回错误到 main 处理

**检查点**: 此时 `ccc patch` 命令应完全功能化且可独立测试

---

## 第 4 阶段：用户故事 2 - 恢复原始 Claude 命令 (优先级: P1)

**目标**: 用户执行 `sudo ccc patch --reset` 恢复原始 claude 命令

**独立测试**: 执行 `sudo ccc patch --reset`，然后运行 `claude --help` 验证显示原始 claude 帮助

### 用户故事 2 的实现

- [X] T017 [US2] 在 `internal/cli/patch.go` 中实现 `resetPatch()` 函数，执行 reset 操作
- [X] T018 [US2] 在 `RunPatch()` 函数中添加 reset 分支逻辑，调用 `resetPatch()`
- [X] T019 [US2] 添加 "Not patched" 错误检测和输出处理

**检查点**: 此时 `ccc patch --reset` 命令应完全功能化且可独立测试

---

## 第 5 阶段：用户故事 3 - 防止重复 Patch (优先级: P2)

**目标**: 检测已 patch 状态并提示，避免重复操作

**独立测试**: 连续执行两次 `sudo ccc patch`，第二次显示 "Already patched"

### 用户故事 3 的实现

- [X] T020 [US3] 在 `RunPatch()` 函数中添加 "Already patched" 检测逻辑
- [X] T021 [US3] 添加 "Already patched" 消息输出，使用 fmt.Printf 到 stdout
- [X] T022 [US3] 确保 "Already patched" 情况下返回 nil（退出码 0）

**检查点**: 重复 patch 被正确拒绝且不影响文件系统

---

## 第 6 阶段：用户故事 4 - Ccc 内部调用真实 Claude (优先级: P1)

**目标**: ccc 优先使用 CCC_CLAUDE 环境变量，避免递归调用

**独立测试**: patch 后执行 `claude` 和 `ccc`，两者都能正常启动，不会循环

### 用户故事 4 的实现

- [X] T023 [US4] 在 `internal/cli/exec.go` 的 `runClaude()` 函数开头添加 CCC_CLAUDE 环境变量检查
- [X] T024 [US4] 修改 `runClaude()` 中的 claudePath 获取逻辑，优先使用环境变量
- [X] T025 [US4] 确保环境变量不存在时仍使用 `exec.LookPath("claude")` 作为后备

**检查点**: ccc 内部调用真实 claude，不会出现递归调用

---

## 第 7 阶段：完善与横切关注点

**目的**: 代码质量、文档和验证

- [X] T026 [P] 在 `internal/cli/patch.go` 中添加所有导出函数的 Go doc 注释（中文）
- [X] T027 确保所有代码通过 `gofmt` 格式化检查
- [X] T028 确保所有代码通过 `go vet` 静态分析检查
- [X] T029 [P] 创建 `tests/unit/patch_test.go`，添加单元测试
- [X] T030 [P] 在测试中添加 `TestFindClaudePath` 测试用例
- [X] T031 [P] 在测试中添加 `TestCheckAlreadyPatched` 测试用例
- [X] T032 [P] 在测试中添加 `TestCreateWrapperScript` 测试用例
- [X] T033 [P] 在测试中添加 `TestApplyPatch` 测试用例（使用 Mock）
- [X] T034 [P] 在测试中添加 `TestResetPatch` 测试用例（使用 Mock）
- [X] T035 更新 `ShowHelp` 函数，添加 patch 命令的帮助信息
- [X] T036 运行 `./check.sh --all` 执行完整检查（lint、test、build）
- [ ] T037 根据 quickstart.md 中的验证场景进行手动测试
- [X] T038 更新 README.md（如有需要），添加 patch 功能说明

---

## 依赖关系与执行顺序

### 阶段依赖

- **基础设施（第 1 阶段）**: 无依赖 - 可立即开始
- **基础（第 2 阶段）**: 依赖基础设施完成 - 阻塞所有用户故事
- **用户故事（第 3-6 阶段）**: 都依赖基础阶段完成
  - US1、US2、US4 为 P1 优先级
  - US3 为 P2 优先级
  - US4 依赖 US1 完成（需要 patch 功能才能测试环境变量逻辑）
- **完善（第 7 阶段）**: 依赖所有期望的用户故事完成

### 用户故事依赖

- **用户故事 1 (P1)**: 基础完成后可开始 - 无其他故事依赖
- **用户故事 2 (P1)**: 基础完成后可开始 - 与 US1 独立
- **用户故事 3 (P2)**: 依赖 US1 完成（使用相同的检测和输出机制）
- **用户故事 4 (P1)**: 依赖 US1 完成（需要 patch 后才能测试）

### 每个用户故事内

- US1: T009 → T010 → T011 → T012 → T013 → T014 → T015 → T016（顺序执行）
- US2: T017 → T018 → T019（顺序执行）
- US3: T020 → T021 → T022（顺序执行）
- US4: T023 → T024 → T025（顺序执行）

### 并行机会

- 第 1 阶段所有任务可并行运行
- 第 7 阶段所有标记 [P] 的任务可并行运行
- US1 和 US2 可以并行开发（不同功能分支）
- US2 完成后，US3 可以并行开发

---

## 并行示例：用户故事 1

```bash
# 用户故事 1 的任务必须顺序执行，无并行机会
# 但 US1 和 US2 可以并行开发：

# 开发者 A：实现 US1（patch 功能）
T009 → T010 → T011 → T012 → T013 → T014 → T015 → T016

# 开发者 B：同时实现 US2（reset 功能）
T017 → T018 → T019
```

---

## 实施策略

### MVP 优先（用户故事 1 + 2 + 4）

1. 完成第 1 阶段：基础设施
2. 完成第 2 阶段：基础（关键 - 阻塞所有故事）
3. 完成第 3 阶段：用户故事 1（patch 功能）
4. 完成第 4 阶段：用户故事 2（reset 功能）
5. 完成第 6 阶段：用户故事 4（环境变量优先）
6. **停止并验证**: 运行 quickstart.md 中的场景 1-8
7. 如准备就绪则提交 PR

### 增量交付

1. 完成基础设施 + 基础 → 命令框架就绪
2. 添加用户故事 1 → 可执行 patch → 测试验证
3. 添加用户故事 2 → 可执行 reset → 测试验证
4. 添加用户故事 4 → 环境变量逻辑 → 测试验证
5. 添加用户故事 3 → 重复检测 → 测试验证
6. 完成完善阶段 → 代码质量检查 → 提交 PR

### 并行团队策略

有多个开发者时：

1. 团队一起完成基础设施 + 基础
2. 基础完成后：
   - 开发者 A：用户故事 1（patch）
   - 开发者 B：用户故事 2（reset）
3. US1 和 US2 完成后：
   - 开发者 A：用户故事 4（环境变量）
   - 开发者 B：用户故事 3（重复检测）
4. 一起完成完善阶段

---

## 注意事项

- [P] 任务 = 不同文件，无依赖
- [故事] 标签将任务映射到特定用户故事以实现可追溯性
- 每个用户故事应可独立完成和测试
- 每个任务或逻辑组后提交
- 在任何检查点停止以独立验证故事
- 避免：模糊任务、同文件冲突、破坏独立性的跨故事依赖
- 所有代码注释使用中文，变量名函数名使用英文

---

## 特别说明：使用中文

**本文档必须使用中文编写。**

1. 所有任务描述必须使用中文。
2. 文件路径使用英文，但说明文字使用中文。
3. 变量名、函数名等标识符使用英文。
