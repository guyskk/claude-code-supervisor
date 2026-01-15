# 实现方案：Claude 命令替换（Patch）

**分支**: `001-claude-patch` | **日期**: 2025-01-15 | **规格**: [spec.md](./spec.md)
**输入**: 来自 `/specs/001-claude-patch/spec.md` 的功能规格

## 特别说明：使用中文

**本文档必须使用中文编写。**

1. 所有技术描述、架构决策、实现细节必须使用中文。
2. 代码示例中的注释必须使用中文。
3. 变量名、函数名等标识符使用英文，但说明文字使用中文。

## 摘要

本功能为 ccc 添加 `patch` 命令，使其能够替代系统的 `claude` 命令。用户执行 `sudo ccc patch` 后，原始 claude 可执行文件被重命名为 ccc-claude，原位置创建一个 sh 包装脚本，该脚本设置环境变量 CCC_CLAUDE 后调用 ccc。ccc 运行时优先检查 CCC_CLAUDE 环境变量，使用真实 claude 路径，避免递归调用。`sudo ccc patch --reset` 恢复原始状态。

## 技术上下文

**语言/版本**: Go 1.21+
**主要依赖**: Go 标准库（os、exec、syscall、fmt）
**存储**: 无需配置文件变更，状态通过文件系统检测
**测试**: go test、竞态检测
**目标平台**: macOS (darwin-amd64, darwin-arm64)、Linux (linux-amd64, linux-arm64)
**项目类型**: single - 单一 Go 项目（CLI 工具）
**性能目标**: patch 操作 < 3 秒完成
**约束**: 单二进制、静态链接、跨平台、不修改配置文件
**规模/范围**: 新增约 200 行代码，新增 1 个文件

## 宪章检查

*门禁：必须在第 0 阶段研究前通过。第 1 阶段设计后再次检查。*

### ccc 项目宪章合规检查

- [x] **原则一：单二进制分发** - 最终产物是单一静态链接二进制文件
- [x] **原则二：代码质量标准** - 符合 gofmt、go vet 要求
- [x] **原则三：测试规范** - 包含单元测试和竞态检测
- [x] **原则四：向后兼容** - 不修改配置文件格式，完全向后兼容
- [x] **原则五：跨平台支持** - 支持 darwin/linux, amd64/arm64
- [x] **原则六：错误处理与可观测性** - 错误明确且可操作

### 复杂度跟踪

> **无违规项，无需填写**

## 项目结构

### 文档组织（本功能）

```text
specs/001-claude-patch/
├── plan.md              # 本文件
├── research.md          # 技术研究文档
├── data-model.md        # 数据结构设计
├── quickstart.md        # 验证场景
├── contracts/           # 命令接口契约
│   └── cli.md           # CLI 命令接口
└── tasks.md             # 任务分解（由 /speckit.tasks 生成）
```

### 源代码组织（仓库根目录）

```text
cmd/ccc/              # 主入口（无需修改）
└── main.go

internal/             # 私有应用代码
├── cli/              # CLI 命令处理
│   ├── cli.go        # [修改] 添加 patch 命令解析
│   ├── exec.go       # [修改] 优先使用 CCC_CLAUDE 环境变量
│   └── patch.go      # [新增] patch 命令实现
├── config/           # 配置管理（无需修改）
├── provider/         # 提供商切换逻辑（无需修改）
└── supervisor/       # Supervisor 模式（无需修改）

tests/                # 测试文件
└── unit/             # 单元测试
    └── patch_test.go # [新增] patch 功能测试
```

**结构决策**: 保持现有项目结构，在 `internal/cli/` 中新增 `patch.go`，修改 `cli.go` 和 `exec.go`。

## 实现阶段

### 第 -1 阶段：预实现门禁

> **重要：在开始任何实现工作前必须通过此阶段**

#### 宪章合规门禁

- [x] 所有 6 条核心原则已检查
- [x] 如有违规，已在"复杂度跟踪"表中记录理由

#### 技术决策门禁

- [x] 技术栈已确定（Go 1.21+，标准库）
- [x] 项目结构已定义
- [x] 数据模型已设计（见 data-model.md）
- [x] API 契约已定义（见 contracts/cli.md）

---

### 第 0 阶段：技术研究

**目标**: 调研技术选项，收集实现所需信息

**输出**: `research.md`

**研究内容**:
- [x] Go 标准库文件操作（os.Rename、os.WriteFile、os.Chmod）
- [x] Shell 包装脚本格式（#!/bin/sh + exec）
- [x] 权限检测方法（测试文件）
- [x] 错误处理与回滚机制
- [x] 跨平台兼容性考虑（POSIX sh 兼容）

---

### 第 1 阶段：架构设计

**目标**: 定义数据模型、API 契约和实现细节

**输出**: `data-model.md`、`contracts/`、`quickstart.md`

#### 数据模型 (data-model.md)

**核心数据结构**:

```go
// PatchCommandOptions 表示 patch 命令的选项
type PatchCommandOptions struct {
    Reset bool // --reset 标志，true 表示恢复原始状态
}

// PatchState 表示 patch 操作的状态
type PatchState struct {
    ClaudePath    string // claude 可执行文件的完整路径
    CccClaudePath string // ccc-claude 的完整路径（claudePath + ".real"）
}
```

**函数签名**:

```go
// RunPatch 执行 patch 命令
func RunPatch(opts *PatchCommandOptions) error

// findClaudePath 查找 claude 可执行文件路径
func findClaudePath() (string, error)

// checkAlreadyPatched 检查是否已经 patch 过
func checkAlreadyPatched() (bool, string, error)

// applyPatch 执行 patch 操作
func applyPatch(claudePath string) error

// resetPatch 执行 reset 操作
func resetPatch(cccClaudePath string) error

// createWrapperScript 创建包装脚本
func createWrapperScript(claudePath, cccClaudePath string) error

// rollbackPatch 回滚 patch 操作
func rollbackPatch(claudePath, cccClaudePath string) error
```

#### API 契约 (contracts/cli.md)

**CLI 命令接口**:

```bash
# Patch: 替换 claude 为 ccc
sudo ccc patch

# Reset: 恢复原始 claude
sudo ccc patch --reset
```

**错误处理契约**:

| 错误场景 | 返回消息 | 退出码 |
|----------|----------|--------|
| Already patched | "Already patched" | 0 |
| Not patched (reset) | "Not patched" | 0 |
| Claude not found | "claude not found in PATH" | 1 |
| Permission denied | "Permission denied, run with sudo" | 1 |
| Other error | 错误描述 | 1 |

#### 快速入门 (quickstart.md)

**关键验证场景**:

1. 首次 patch: `sudo ccc patch` → 成功
2. 验证 patch: `claude --help` → 显示 ccc 帮助
3. 重复 patch: `sudo ccc patch` → "Already patched"
4. Reset: `sudo ccc patch --reset` → 成功
5. 验证 reset: `claude --help` → 显示原始 claude 帮助
6. 重复 reset: `sudo ccc patch --reset` → "Not patched"

---

### 第 2 阶段：任务分解

**目标**: 将设计转化为可执行的任务列表

**输出**: `tasks.md` (由 `/speckit.tasks` 命令生成)

> **注意**: 第 2 阶段不在此方案中完成，由独立的 `/speckit.tasks` 命令处理

---

## 实施文件创建顺序

> **重要：按照此顺序创建文件以确保质量**

1. **contracts/cli.md** - 首先定义 CLI 命令接口和错误处理契约
2. **测试文件** - 按以下顺序创建：
   - `tests/unit/patch_test.go` - 单元测试
3. **源代码文件** - 创建使测试通过的实现：
   - `internal/cli/patch.go` - patch 命令实现
   - `internal/cli/cli.go` - 修改：添加 patch 命令解析
   - `internal/cli/exec.go` - 修改：优先使用 CCC_CLAUDE 环境变量

**理由**: 测试先行确保 API 设计可用，实现符合需求。

---

## 复杂度跟踪

> **无违规项，无需填写**
