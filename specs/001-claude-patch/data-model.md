# 数据模型：Claude 命令替换（Patch）

**功能分支**: `001-claude-patch`
**创建日期**: 2025-01-15

## 核心数据结构

### PatchCommandOptions

patch 命令的选项，从命令行参数解析得到。

```go
// PatchCommandOptions 表示 patch 命令的选项
type PatchCommandOptions struct {
    Reset bool // --reset 标志，true 表示恢复原始状态
}
```

**字段说明**:
- `Reset`: 布尔值，当用户执行 `ccc patch --reset` 时为 true

---

### PatchState

patch 操作的状态信息，用于在函数间传递路径信息。

```go
// PatchState 表示 patch 操作的状态
type PatchState struct {
    ClaudePath    string // claude 可执行文件的完整路径
    CccClaudePath string // ccc-claude 的完整路径（claudePath + ".real"）
}
```

**字段说明**:
- `ClaudePath`: 原始 claude 可执行文件的完整路径（如 `/usr/local/bin/claude`）
- `CccClaudePath`: 重命名后的路径（如 `/usr/local/bin/ccc-claude`）

**约束**:
- `CccClaudePath` 必须与 `ClaudePath` 在同一目录下
- `CccClaudePath` 的名称是 `filepath.Base(ClaudePath) + ".real"`

---

## 函数签名

### RunPatch

执行 patch 命令的入口函数。

```go
// RunPatch 执行 patch 命令
// opts.Reset 为 false 时执行 patch，为 true 时执行 reset
// 返回 error 表示操作失败，nil 表示成功
func RunPatch(opts *PatchCommandOptions) error
```

**错误处理**:
- Already patched: 返回特定错误或打印消息后返回 nil
- Not patched (reset): 返回特定错误或打印消息后返回 nil
- 其他错误: 返回包装后的错误信息

---

### findClaudePath

查找 claude 可执行文件路径。

```go
// findClaudePath 查找 claude 可执行文件路径
// 使用 exec.LookPath 在 PATH 中查找
// 返回完整路径和 error（未找到时）
func findClaudePath() (string, error)
```

**错误处理**:
- 未找到 claude: 返回错误 "claude not found in PATH"

---

### checkAlreadyPatched

检查是否已经 patch 过。

```go
// checkAlreadyPatched 检查是否已经 patch 过
// 使用 exec.LookPath("ccc-claude") 检测
// 返回：是否已 patch、ccc-claude 路径、error
func checkAlreadyPatched() (bool, string, error)
```

**逻辑**:
- 如果 ccc-claude 存在：返回 (true, path, nil)
- 如果 ccc-claude 不存在：返回 (false, "", nil)

---

### applyPatch

执行 patch 操作。

```go
// applyPatch 执行 patch 操作
// 步骤：
// 1. 重命名 claude → ccc-claude
// 2. 创建包装脚本
// 3. 如果步骤 2 失败，回滚步骤 1
// 返回 error 表示操作失败
func applyPatch(claudePath string) error
```

**错误处理**:
- 重命名失败: 返回错误
- 创建脚本失败: 回滚重命名操作，返回错误

---

### resetPatch

执行 reset 操作。

```go
// resetPatch 执行 reset 操作
// 步骤：
// 1. 找到 ccc-claude 路径
// 2. 重命名 ccc-claude → claude（覆盖包装脚本）
// 返回 error 表示操作失败
func resetPatch(cccClaudePath string) error
```

**错误处理**:
- ccc-claude 不存在: 返回 "Not patched" 错误
- 重命名失败: 返回错误

---

### createWrapperScript

创建包装脚本。

```go
// createWrapperScript 创建包装脚本
// 脚本内容：
//   #!/bin/sh
//   export CCC_CLAUDE=<cccClaudePath>
//   exec ccc "$@"
// 返回 error 表示创建失败
func createWrapperScript(claudePath, cccClaudePath string) error
```

**脚本权限**: 0755 (rwxr-xr-x)

**错误处理**:
- 创建文件失败: 返回错误
- 设置权限失败: 返回错误

---

### rollbackPatch

回滚 patch 操作。

```go
// rollbackPatch 回滚 patch 操作
// 在 patch 失败时调用，将 ccc-claude 改回 claude
// 返回 error 表示回滚失败
func rollbackPatch(claudePath, cccClaudePath string) error
```

**错误处理**:
- 回滚失败: 记录错误日志，但返回原始错误

---

## 环境变量

### CCC_CLAUDE

由包装脚本设置，ccc 使用此环境变量找到真实的 claude。

```bash
export CCC_CLAUDE=/usr/local/bin/ccc-claude
```

**用途**:
- ccc 在 `runClaude` 函数中优先检查此环境变量
- 如果存在，直接使用此路径作为 claude 路径
- 如果不存在，使用 `exec.LookPath("claude")` 查找

**设置位置**: 包装脚本（`claude` 文件）

**读取位置**: `internal/cli/exec.go` 中的 `runClaude` 函数

---

## 错误类型

### 错误消息规范

| 场景 | 错误消息 | 退出码 |
|------|----------|--------|
| Already patched | Already patched | 0 |
| Not patched (reset) | Not patched | 0 |
| Claude not found | claude not found in PATH | 1 |
| Permission denied | Permission denied, run with sudo | 1 |
| Rename failed | failed to rename claude: {error} | 1 |
| Script creation failed | failed to create wrapper script: {error} | 1 |
| Rollback failed | failed to rollback patch: {error} | 1 |

### 错误包装

所有错误都应使用 `fmt.Errorf` 包装上下文：

```go
return fmt.Errorf("failed to rename claude: %w", err)
```

---

## 配置文件

### 无需修改

本功能不修改 `ccc.json` 配置文件，状态通过文件系统检测（ccc-claude 是否存在）。
