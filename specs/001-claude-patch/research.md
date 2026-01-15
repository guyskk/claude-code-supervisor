# 技术研究：Claude 命令替换（Patch）

**功能分支**: `001-claude-patch`
**创建日期**: 2025-01-15
**状态**: 完成

## 研究目标

研究 patch 功能实现所需的技术选项和最佳实践。

---

## 研究内容

### 1. Go 标准库相关功能

#### 文件操作
- `os.Rename(old, new)` - 重命名/移动文件
  - 跨设备重命名会失败，返回 `syscall.EXDEV` 错误
  - 对于同文件系统操作是原子的
- `os.WriteFile(name, data, perm)` - 创建文件并设置权限
- `os.Chmod(name, perm)` - 修改文件权限

#### 路径查找
- `os/exec.LookPath(file)` - 在 PATH 中查找可执行文件
  - 返回完整路径或错误
  - 检查文件是否可执行

#### 环境变量
- `os.Getenv(key)` - 获取环境变量
- `os.Setenv(key, value)` - 设置环境变量（仅对当前进程）

**决策**: 使用标准库，无第三方依赖。

---

### 2. Shell 包装脚本格式

包装脚本需要：
1. 设置环境变量
2. 执行 ccc 并传递所有参数
3. 使用 `exec` 替换进程（避免额外 shell 层）

**脚本模板**:
```bash
#!/bin/sh
export CCC_CLAUDE=/usr/local/bin/ccc-claude
exec ccc "$@"
```

**关键点**:
- 使用 `#!/bin/sh` 而非 `#!/bin/bash`，更通用（POSIX 兼容）
- 使用 `exec` 替换 shell 进程，避免创建额外进程
- `"$@"` 正确处理带空格的参数

**决策**: 使用上述模板格式。

---

### 3. 权限检测

检测是否有写权限的方法：
1. 尝试创建临时文件
2. 检查文件/目录的权限位

**最佳实践**: 尝试创建测试文件，删除后判断。

```go
func checkWritePermission(path string) error {
    dir := filepath.Dir(path)
    testFile := filepath.Join(dir, ".ccc_write_test")
    if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
        return err
    }
    os.Remove(testFile)
    return nil
}
```

**决策**: 使用测试文件方法检测权限。

---

### 4. 错误处理与回滚

Patch 操作的关键步骤：
1. 检测 ccc-claude 是否存在
2. 检测 claude 路径
3. 重命名 claude → ccc-claude
4. 创建包装脚本

**失败场景**:
- 步骤 3 失败: 文件系统错误，需要回滚（无操作）
- 步骤 4 失败: 已重命名但脚本创建失败，需要将 ccc-claude 改回 claude

**决策**: 在创建包装脚本失败时，执行回滚操作。

---

### 5. 跨平台考虑

本功能不涉及 Windows（用户明确说明）。

支持的平台：
- macOS (darwin-amd64, darwin-arm64)
- Linux (linux-amd64, linux-arm64)

**平台差异**:
- macOS 可能使用 zsh，Linux 可能使用 bash
- 包裹脚本使用 `#!/bin/sh` 兼容两者

**决策**: 无需特殊平台处理。

---

## 研究结论

### 技术选型

| 类别 | 选择 | 理由 |
|------|------|------|
| 文件操作 | Go 标准库 `os` 包 | 无需额外依赖，功能完整 |
| 路径查找 | `exec.LookPath` | 标准库函数，可靠 |
| 包装脚本 | `#!/bin/sh` | POSIX 兼容，适用于所有目标平台 |
| 权限检测 | 测试文件方法 | 最可靠的检测方式 |

### 架构决策

1. **不修改配置文件**: 状态通过文件系统检测（ccc-claude 是否存在）
2. **包装脚本使用绝对路径**: 避免路径问题
3. **失败回滚**: patch 失败时自动恢复原状
4. **环境变量优先**: ccc 优先使用 CCC_CLAUDE 环境变量

### 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 跨设备重命名失败 | 在同目录操作，避免跨设备 |
| 权限不足 | 提示用户使用 sudo |
| 并发 patch | 文件锁或检测（暂不处理，假设用户不会并发操作） |
| ccc-claude 被手动删除 | 运行时检测并提示 |

---

## 实现建议

1. 创建 `internal/cli/patch.go` 实现 patch 命令
2. 修改 `internal/cli/cli.go` 添加 patch 命令解析
3. 修改 `internal/cli/exec.go` 的 `runClaude` 函数，优先检查 CCC_CLAUDE 环境变量

无需修改配置结构（config.go），不需要存储 patch 状态。
