# CLI 命令契约：Claude 命令替换（Patch）

**功能分支**: `001-claude-patch`
**创建日期**: 2025-01-15

## 命令接口

### ccc patch

替换系统中的 claude 命令为 ccc。

```bash
sudo ccc patch
```

**行为**:
1. 检查 ccc-claude 是否已存在，如果存在则输出 "Already patched" 并退出（退出码 0）
2. 查找 claude 可执行文件路径（使用 exec.LookPath）
3. 重命名 claude → ccc-claude
4. 在原 claude 位置创建包装脚本
5. 如果步骤 4 失败，回滚步骤 3

**成功输出**:
```
Patched successfully
Claude command now uses ccc
Run 'claude' to start ccc, or 'sudo ccc patch --reset' to undo
```

**失败输出**:
```
Already patched
```

或

```
Error: <错误描述>
```

---

### ccc patch --reset

恢复原始的 claude 命令。

```bash
sudo ccc patch --reset
```

**行为**:
1. 查找 ccc-claude 路径（使用 exec.LookPath）
2. 如果不存在，输出 "Not patched" 并退出（退出码 0）
3. 重命名 ccc-claude → claude（覆盖包装脚本）

**成功输出**:
```
Reset successfully
Claude command restored to original
```

**失败输出**:
```
Not patched
```

或

```
Error: <错误描述>
```

---

## 错误处理契约

### 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功（包括 "Already patched" 和 "Not patched"） |
| 1 | 错误 |

### 错误消息

| 场景 | 消息 | 输出流 |
|------|------|--------|
| Already patched | Already patched | stdout |
| Not patched (reset) | Not patched | stdout |
| Claude not found | claude not found in PATH | stderr |
| Permission denied | Permission denied, run with sudo | stderr |
| Rename failed | failed to rename claude: {详细错误} | stderr |
| Script creation failed | failed to create wrapper script: {详细错误} | stderr |
| 其他错误 | {详细错误描述} | stderr |

---

## 包装脚本格式

### 位置

由 claude 路径决定，例如 `/usr/local/bin/claude`

### 内容

```bash
#!/bin/sh
export CCC_CLAUDE=/usr/local/bin/ccc-claude
exec ccc "$@"
```

**关键点**:
1. 使用 `#!/bin/sh`（POSIX 兼容）
2. 使用 `export` 设置环境变量
3. 使用 `exec` 替换进程
4. 使用 `"$@"` 传递所有参数

### 权限

0755 (rwxr-xr-x)

---

## 环境变量

### CCC_CLAUDE

**设置者**: 包装脚本
**使用者**: ccc (internal/cli/exec.go)
**值**: ccc-claude 的完整路径

**用途**: ccc 优先使用此环境变量作为 claude 路径，避免递归调用
