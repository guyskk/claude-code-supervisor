# 功能规格：Claude 命令替换（Patch）

**功能分支**: `001-claude-patch`
**创建日期**: 2025-01-15
**状态**: 草稿
**输入**: 用户描述："添加 patch 命令，让 ccc 能够替代系统的 claude 命令"

## 特别说明：使用中文

**本文档必须使用中文编写。**

1. 所有用户故事、需求描述、验收场景必须使用中文。
2. 用户场景描述应该使用自然、易懂的中文。
3. 功能需求使用中文描述，技术术语保留英文。

## 用户场景与测试 *(必填)*

### 用户故事 1 - 启用 Claude 命令替换 (优先级: P1)

作为开发者，我希望执行 `ccc patch` 命令后，系统中的 `claude` 命令能够被 ccc 替代，这样我就可以在任何场景下自动享受 ccc 的 supervisor 模式和提供商切换功能，无需手动调用 ccc。

**为什么是这个优先级**: 这是核心功能，用户的主要需求就是让所有调用 `claude` 的地方都自动使用 ccc。

**独立测试**: 可以通过执行 `sudo ccc patch`，然后运行 `claude --version` 验证是否调用 ccc，完全测试并交付命令替换功能。

**验收场景**:

1. **给定** 系统中已安装 claude 且尚未 patch，**当** 用户执行 `sudo ccc patch`，**那么** claude 可执行文件被重命名为 ccc-claude，原位置创建包装脚本
2. **给定** 系统已 patch，**当** 用户执行 `claude --help`，**那么** 实际执行的是 ccc，显示 ccc 的帮助信息
3. **给定** 系统已 patch，**当** 用户执行 `claude glm`（指定提供商），**那么** ccc 使用 glm 提供商启动 claude
4. **给定** 任何第三方脚本调用 claude，**当** 执行该脚本，**那么** 实际调用的是 ccc

---

### 用户故事 2 - 恢复原始 Claude 命令 (优先级: P1)

作为开发者，我希望执行 `ccc patch --reset` 能够恢复原始的 claude 命令，这样我可以随时切换回使用官方 claude。

**为什么是这个优先级**: 这是 patch 功能的必要配套功能，用户需要能够恢复原状。

**独立测试**: 可以通过执行 `sudo ccc patch --reset`，然后运行 `claude --version` 验证是否恢复原始 claude，完全测试并交付恢复功能。

**验收场景**:

1. **给定** 系统已 patch，**当** 用户执行 `sudo ccc patch --reset`，**那么** ccc-claude 被重命名回 claude，包装脚本被删除
2. **给定** 系统已恢复，**当** 用户执行 `claude --help`，**那么** 显示原始 claude 的帮助信息
3. **给定** 系统未 patch，**当** 用户执行 `sudo ccc patch --reset`，**那么** 提示 "Not patched" 并退出

---

### 用户故事 3 - 防止重复 Patch (优先级: P2)

作为开发者，我希望系统检测到已 patch 的状态并提示，避免重复操作导致的问题。

**为什么是这个优先级**: 这是用户体验优化，防止用户误操作。

**独立测试**: 可以通过连续执行两次 `sudo ccc patch`，第二次应该显示 "Already patched" 提示。

**验收场景**:

1. **给定** 系统已 patch，**当** 用户执行 `sudo ccc patch`，**那么** 显示 "Already patched" 提示并退出
2. **给定** 系统已 patch，**当** 用户尝试再次 patch，**那么** 不对文件系统做任何修改

---

### 用户故事 4 - Ccc 内部调用真实 Claude (优先级: P1)

作为开发者，我希望 ccc 内部调用的 claude 是真实的 claude 可执行文件，不受 patch 影响，避免无限递归。

**为什么是这个优先级**: 这是正确性的关键，没有这个功能 patch 会导致无限循环。

**独立测试**: 可以在 patch 后直接执行 `ccc` 或 `claude`，两者都应该能正常启动 claude，不会出现循环调用。

**验收场景**:

1. **给定** 系统已 patch，**当** 用户执行 `claude`，**那么** 环境变量 CCC_CLAUDE 被设置，ccc 使用真实 claude 路径
2. **给定** 系统已 patch，**当** 用户直接执行 `ccc`，**那么** ccc 使用 LookPath 查找 claude（找到包装脚本）
3. **给定** 包装脚本被执行，**当** ccc 启动，**那么** ccc 检测到 CCC_CLAUDE 环境变量，直接使用真实 claude 路径

---

### 边缘情况

- **当** 用户在 patch 后手动删除了 ccc-claude **会发生什么**？系统应该检测到并提示用户修复状态
- **当** patch 期间出现错误（如权限不足、磁盘已满）**会发生什么**？系统应该尝试回滚已执行的操作
- **当** 系统中存在多个 claude 可执行文件时 **会发生什么**？patch 只处理 PATH 中第一个找到的 claude
- **当** 用户没有 sudo 权限时 **会发生什么**？系统应该提示需要使用 sudo 执行 patch 命令

## 需求 *(必填)*

### 功能需求

- **FR-001**: 系统必须支持 `ccc patch` 命令，将系统中的 claude 可执行文件替换为 ccc
- **FR-002**: 系统必须支持 `ccc patch --reset` 命令，恢复原始的 claude 可执行文件
- **FR-003**: 系统必须在 patch 前检测 ccc-claude 是否已存在，如果存在则提示 "Already patched"
- **FR-004**: 系统必须在 reset 前检测 ccc-claude 是否存在，如果不存在则提示 "Not patched"
- **FR-005**: 系统必须将原始 claude 重命名为 ccc-claude
- **FR-006**: 系统必须在原 claude 位置创建 sh 包装脚本
- **FR-007**: 包装脚本必须设置环境变量 CCC_CLAUDE 指向 ccc-claude 的完整路径
- **FR-008**: 包装脚本必须执行 `exec ccc "$@"` 将所有参数传递给 ccc
- **FR-009**: ccc 必须优先检查 CCC_CLAUDE 环境变量，如果存在则使用该路径作为 claude 路径
- **FR-010**: ccc 必须在 CCC_CLAUDE 环境变量不存在时使用 exec.LookPath("claude") 查找 claude
- **FR-011**: 包装脚本必须具有可执行权限 (0755)
- **FR-012**: 系统必须在 patch 失败时尝试回滚操作（将 ccc-claude 改回 claude）
- **FR-013**: 系统必须在权限不足时提示用户使用 sudo

### 核心实体 *(如果功能涉及数据则包含)*

- **Claude 可执行文件路径**: 系统中 claude 命令的完整路径（如 /usr/local/bin/claude）
- **Ccc-claude 路径**: 原始 claude 重命名后的路径（如 /usr/local/bin/ccc-claude）
- **包装脚本**: 在原 claude 位置创建的 sh 脚本，用于设置环境变量并调用 ccc

## 成功标准 *(必填)*

### 可衡量的结果

- **SC-001**: 用户执行 `sudo ccc patch` 后，100% 的 `claude` 命令调用都通过 ccc
- **SC-002**: 用户执行 `sudo ccc patch --reset` 后，`claude` 命令恢复为原始行为
- **SC-003**: patch 操作在 3 秒内完成
- **SC-004**: 重复 patch 时，100% 显示 "Already patched" 提示且不修改文件系统
- **SC-005**: ccc 内部调用 claude 时，100% 使用真实的 claude 可执行文件，不会出现递归调用

## 需求完整性检查

在继续到实现方案 (`/speckit.plan`) 之前，验证：

- [x] 没有 `[需要澄清]` 标记残留
- [x] 所有需求都可测试且无歧义
- [x] 成功标准可衡量
- [x] 每个用户故事都可独立实现和测试
- [x] 边缘情况已考虑
- [x] 与宪章原则一致（单二进制、跨平台、向后兼容）
