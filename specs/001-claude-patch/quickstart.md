# 快速入门验证：Claude 命令替换（Patch）

**功能分支**: `001-claude-patch`
**创建日期**: 2025-01-15

## 关键验证场景

### 场景 1: 首次 Patch

**目标**: 验证 patch 操作成功

**步骤**:
```bash
# 1. 执行 patch
sudo ccc patch

# 2. 验证 ccc-claude 存在
which ccc-claude

# 3. 验证 claude 是包装脚本
cat $(which claude)

# 预期输出：
# #!/bin/sh
# export CCC_CLAUDE=...
# exec ccc "$@"
```

**预期结果**: 输出 "Patched successfully"

---

### 场景 2: 验证 Patch 后行为

**目标**: 验证 patch 后调用 claude 实际调用 ccc

**步骤**:
```bash
# 1. 执行 claude --help
claude --help

# 预期输出：显示 ccc 的帮助信息
# Usage: ccc [provider] [args...]
#        ccc validate [provider] [--all]
#
# Claude Code Supervisor and Configuration Switcher
```

**预期结果**: 显示 ccc 的帮助信息（不是原始 claude 的帮助）

---

### 场景 3: 重复 Patch

**目标**: 验证重复 patch 被正确拒绝

**步骤**:
```bash
# 1. 再次执行 patch
sudo ccc patch

# 预期输出：
# Already patched
```

**预期结果**: 输出 "Already patched"，不修改文件系统

---

### 场景 4: Reset

**目标**: 验证 reset 操作成功

**步骤**:
```bash
# 1. 执行 reset
sudo ccc patch --reset

# 2. 验证 ccc-claude 不存在
which ccc-claude

# 预期输出：
# ccc-claude not found
```

**预期结果**: 输出 "Reset successfully"

---

### 场景 5: 验证 Reset 后行为

**目标**: 验证 reset 后 claude 恢复原始行为

**步骤**:
```bash
# 1. 执行 claude --help
claude --help

# 预期输出：显示原始 claude 的帮助信息
```

**预期结果**: 显示原始 claude 的帮助信息（不是 ccc 的帮助）

---

### 场景 6: 重复 Reset

**目标**: 验证重复 reset 被正确拒绝

**步骤**:
```bash
# 1. 再次执行 reset
sudo ccc patch --reset

# 预期输出：
# Not patched
```

**预期结果**: 输出 "Not patched"

---

### 场景 7: Ccc 直接调用

**目标**: 验证 ccc 直接调用时不受 patch 影响

**步骤**:
```bash
# 1. Patch
sudo ccc patch

# 2. 直接调用 ccc
ccc --help

# 预期输出：显示 ccc 的帮助信息
```

**预期结果**: ccc 正常工作，显示帮助信息

---

### 场景 8: 第三方脚本调用

**目标**: 验证第三方脚本调用 claude 时使用 ccc

**步骤**:
```bash
# 1. Patch
sudo ccc patch

# 2. 创建测试脚本
echo '#!/bin/sh
claude --version' > test.sh
chmod +x test.sh

# 3. 执行测试脚本
./test.sh

# 预期输出：显示 ccc 的版本信息
```

**预期结果**: 实际调用的是 ccc

---

## 测试检查清单

- [ ] 场景 1: 首次 patch 成功
- [ ] 场景 2: patch 后 claude 显示 ccc 帮助
- [ ] 场景 3: 重复 patch 被拒绝
- [ ] 场景 4: reset 成功
- [ ] 场景 5: reset 后 claude 恢复原始行为
- [ ] 场景 6: 重复 reset 被拒绝
- [ ] 场景 7: ccc 直接调用正常工作
- [ ] 场景 8: 第三方脚本调用使用 ccc

---

## 手动清理（如果测试失败）

如果测试中途失败，手动清理：

```bash
# 查找 claude 相关文件
which claude
which ccc-claude

# 手动恢复
sudo mv /usr/local/bin/ccc-claude /usr/local/bin/claude

# 或删除包装脚本
sudo rm /usr/local/bin/claude
```
