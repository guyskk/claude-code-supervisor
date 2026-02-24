# Settings.json 配置合并策略

## 问题背景

### Issue #71: 插件配置丢失

当用户在 ccc 启动的 Claude Code 会话中通过 `/plugin install` 安装新插件时，插件会被正确添加到 `~/.claude/settings.json` 的 `enabledPlugins` 字段中。但在下一次运行 ccc 时，新安装的插件会从 `enabledPlugins` 中消失。

### 根本原因

在 `internal/provider/provider.go` 的 `SwitchWithHook()` 函数中（第 60-65 行），使用了**浅层循环覆盖**来生成 settings.json：

```go
// Build settings with hook, but without env
settingsWithHook := make(map[string]interface{})
for k, v := range mergedSettings {
    if k != "env" {
        settingsWithHook[k] = v  // 整个 enabledPlugins map 被替换
    }
}
```

`enabledPlugins` 是嵌套的 `map[string]interface{}`。浅层覆盖会用 ccc.json 中保存的旧快照直接替换掉 settings.json 中可能包含新插件的内容。

### 更广泛的问题

这不仅仅是插件的问题。**任何用户在 settings.json 中的手动修改**都会在下一次 ccc 启动时被覆盖丢失，包括：
- 用户手动修改的 `permissions` 配置
- 用户配置的 `sandbox` 设置
- 用户添加的自定义 hooks（PreToolUse、SessionStart 等）
- 其他用户自定义配置

---

## 配置来源与优先级

### 配置来源

| 来源 | 文件路径 | 说明 |
|------|----------|------|
| **settings.json** | `~/.claude/settings.json` | 用户的实际配置文件，应由用户主导 |
| **provider settings** | `ccc.json` 中的 `providers.{name}` | provider 特定配置 |
| **base settings** | `ccc.json` 中的 `settings` | 共享模板/默认值 |

### 优先级原则

**settings.json 的配置优先级最高**。用户直接编辑的配置应该被保留，即使与 ccc.json 中的配置冲突。

优先级（从高到低）：
1. **userSettings** (settings.json - 用户实际配置)
2. **providerSettings** (provider 特定配置)
3. **baseSettings** (ccc.json settings - 模板)

---

## 解决方案设计

### 核心思想

**以 settings.json 为主体，只做最小必要干预**。

ccc 的核心职责是：
1. 管理 provider 切换（通过环境变量）
2. 确保 Supervisor hook 可用

ccc 不应该：
- 替代用户管理 settings.json 的所有配置
- 覆盖用户的手动修改

---

## 字段处理策略

### 1. env 字段 - 特殊处理

**处理方式**：清空特定键，避免配置冲突。

需要清空的键：
1. 特定前缀：`ANTHROPIC_*`、`CLAUDE_*`
2. 与 provider env 相同的 key

**原因**：
- provider 的环境变量通过命令行传递给 claude 子进程
- 如果 settings.json 中保留这些键，会产生不确定性（不确定哪边生效）
- 清空后确保 provider env 的行为可预测

**示例**：

```json
// settings.json 初始内容
{
  "env": {
    "ANTHROPIC_MODEL": "claude-3.7-sonnet",
    "MY_CUSTOM_VAR": "value",
    "ANTHROPIC_BASE_URL": "old-url"
  }
}

// provider env
{
  "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
  "ANTHROPIC_AUTH_TOKEN": "token123",
  "ANTHROPIC_MODEL": "glm-4.7"
}

// 处理后
{
  "env": {
    "MY_CUSTOM_VAR": "value"    // 保留（非 ANTHROPIC_* 且非 provider key）
  }
}
```

---

### 2. hooks 字段 - Selective 处理

**处理方式**：确保 Supervisor Stop hook 存在，保留用户其他 hooks。

需要保证的配置：
1. `hooks.Stop` 中包含 Supervisor hook
2. `disableAllHooks` 设置为 `false`
3. `allowManagedHooksOnly` 设置为 `false`

**保留的内容**：
- 用户配置的其他 hooks（PreToolUse、SessionStart、SessionEnd 等）

**示例**：

```json
// settings.json 初始内容
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{
          "type": "command",
          "command": "echo 'Running bash...'"
        }]
      }
    ]
  }
}

// 处理后
{
  "hooks": {
    "PreToolUse": [...],     // 保留用户配置
    "Stop": [                // 新增/确保 Stop hook
      {
        "hooks": [{
          "type": "command",
          "command": "/path/to/ccc supervisor-hook",
          "timeout": 600
        }]
      }
    ]
  },
  "disableAllHooks": false,
  "allowManagedHooksOnly": false
}
```

---

### 3. enabledPlugins - 完全保留

**处理方式**：完全保留 settings.json 中的 enabledPlugins，不进行任何修改。

**原因**：插件管理是 Claude Code 的职责，ccc 不应干预。

---

### 4. 其他字段 - Deep Merge

**处理方式**：使用 DeepMerge 进行深度合并，settings.json 优先。

包括的字段：
- `permissions`（allow、deny、defaultMode 等）
- `sandbox`
- `alwaysThinkingEnabled`
- `model`
- `attribution`
- 其他用户配置

**示例**：

```json
// ccc.json
{
  "settings": {
    "permissions": {
      "defaultMode": "bypassPermissions",
      "deny": ["Read(.env)"]
    }
  }
}

// settings.json
{
  "permissions": {
    "defaultMode": "acceptEdits",
    "allow": ["Bash(git *)"]
  }
}

// 合并后（settings.json 优先）
{
  "permissions": {
    "defaultMode": "acceptEdits",      // settings.json 覆盖
    "deny": ["Read(.env)"],          // 保留
    "allow": ["Bash(git *)"]         // 保留
  }
}
```

---

## 函数设计

### 1. LoadSettings()

**描述**：读取现有的 settings.json 文件。

**签名**：
```go
// LoadSettings reads the existing settings.json file.
// Returns nil if the file doesn't exist.
func LoadSettings() (map[string]interface{}, error)
```

**逻辑**：
1. 读取 `~/.claude/settings.json`
2. 文件不存在时返回 `nil`（不是错误）
3. 解析错误时返回 `error`

---

### 2. CleanEnvInSettings()

**描述**：清空 settings.env 中的特定环境变量键。

**签名**：
```go
// CleanEnvInSettings removes specific environment variable keys from settings.env.
// It removes:
//   1. Keys with specific prefixes (ANTHROPIC_*, CLAUDE_*)
//   2. Keys that match provider env keys
// Returns a new map without modifying the input.
func CleanEnvInSettings(settings map[string]interface{}, providerEnvKeys []string) map[string]interface{}
```

**逻辑**：
1. 深拷贝 settings（不修改输入）
2. 获取 `env` map（不存在则跳过）
3. 遍历每个 key
4. 删除满足以下任一条件的 key：
   - 以 `ANTHROPIC_` 开头
   - 以 `CLAUDE_` 开头
   - 存在于 `providerEnvKeys` 列表中
5. 返回新的 map

---

### 3. MergeWithPriority()

**描述**：按优先级深度合并多个配置源。

**签名**：
```go
// MergeWithPriority merges multiple settings with priority.
// Priority (highest to lowest):
//   1. userSettings (settings.json - the actual user config)
//   2. providerSettings (provider-specific config)
//   3. baseSettings (ccc.json settings - template)
//
// Returns a new merged map without modifying the inputs.
func MergeWithPriority(baseSettings, providerSettings, userSettings map[string]interface{}) map[string]interface{}
```

**逻辑**：
1. result = DeepCopy(baseSettings)
2. result = DeepMerge(result, providerSettings)
3. result = DeepMerge(result, userSettings)
4. 返回 result

**注意**：此函数用于一般字段，hooks 和 env 有特殊处理。

---

### 4. EnsureStopHook()

**描述**：确保 Supervisor Stop hook 存在于 settings 中。

**签名**：
```go
// EnsureStopHook ensures that Supervisor Stop hook exists in settings.
// It preserves user's other hooks configuration.
// Returns a new map with hook ensured.
func EnsureStopHook(settings map[string]interface{}, hookCommand string) map[string]interface{}
```

**逻辑**：
1. 深拷贝 settings（不修改输入）
2. 获取或创建 `hooks` map
3. 创建或替换 `Stop` hook 数组
4. 创建 Stop hook 配置（type、command、timeout）
5. 返回新的 settings

---

## 执行流程

### SwitchWithHook() 新流程

```
开始
  │
  ├─→ LoadSettings() ──────────────────────────→ userSettings
  │   └─ settings.json 不存在 → nil
  │
  ├─→ baseSettings = cfg.Settings
  ├─→ providerSettings = cfg.Providers[providerName]
  │
  ├─→ 提取 provider env keys
  │
  ├─→ MergeWithPriority(baseSettings, providerSettings, userSettings)
  │   │
  │   └─→ merged = DeepMerge(DeepCopy(baseSettings), providerSettings)
  │           merged = DeepMerge(merged, userSettings)  ← userSettings 优先
  │
  ├─→ CleanEnvInSettings(merged, providerEnvKeys)
  │   └─→ 清空 ANTHROPIC_*, CLAUDE_*, provider env keys
  │
  ├─→ EnsureStopHook(merged, hookCommand)
  │   └─→ 确保 Supervisor Stop hook 存在
  │
  ├─→ 确保 hooks 可执行
  │   ├─→ merged["disableAllHooks"] = false
  │   └─→ merged["allowManagedHooksOnly"] = false
  │
  └─→ 保存 merged 到 settings.json
```

---

## 配置合并场景分析

### 场景 1：简单字段（非嵌套）

```json
// ccc.json
{
  "settings": {
    "alwaysThinkingEnabled": true
  }
}

// settings.json
{
  "alwaysThinkingEnabled": false
}
```

**处理**：保留 `false`（settings.json 优先）

---

### 场景 2：嵌套 map 字段（permissions）

```json
// ccc.json
{
  "settings": {
    "permissions": {
      "defaultMode": "bypassPermissions",
      "deny": ["Read(.env)"]
    }
  }
}

// settings.json
{
  "permissions": {
    "defaultMode": "acceptEdits",
    "allow": ["Bash(git *)"]
  }
}
```

**处理**：DeepMerge 后 settings.json 优先：

```json
{
  "permissions": {
    "defaultMode": "acceptEdits",      // settings.json 覆盖
    "deny": ["Read(.env)"],          // 保留
    "allow": ["Bash(git *)"]         // 保留
  }
}
```

---

### 场景 3：env 字段清空

```json
// settings.json 初始内容
{
  "env": {
    "ANTHROPIC_MODEL": "claude-3.7-sonnet",
    "MY_CUSTOM_VAR": "value",
    "CLAUDE_BASH_MAINTAIN_PROJECT_WORKING_DIR": "1"
  }
}

// provider env
{
  "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
  "ANTHROPIC_AUTH_TOKEN": "token123",
  "ANTHROPIC_MODEL": "glm-4.7"
}
```

**处理后**：

```json
{
  "env": {
    "MY_CUSTOM_VAR": "value"    // 保留（非 ANTHROPIC_* 且非 CLAUDE_* 且非 provider key）
  }
}
```

---

### 场景 4：hooks selective 处理

```json
// settings.json 初始内容
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{
          "type": "command",
          "command": "echo 'Running bash...'"
        }]
      }
    ]
  }
}
```

**处理后**（确保 Stop hook）：

```json
{
  "hooks": {
    "PreToolUse": [...],     // 保留用户配置
    "Stop": [                // 新增/确保 Stop hook
      {
        "hooks": [{
          "type": "command",
          "command": "/path/to/ccc supervisor-hook",
          "timeout": 600
        }]
      }
    ]
  },
  "disableAllHooks": false,
  "allowManagedHooksOnly": false
}
```

---

## 代码修改文件

| 文件 | 修改内容 |
|------|----------|
| `internal/config/config.go` | 新增 LoadSettings、CleanEnvInSettings、MergeWithPriority、EnsureStopHook |
| `internal/provider/provider.go` | 重写 SwitchWithHook() 函数逻辑 |
| `internal/config/config_test.go` | 为新函数添加测试 |

---

## 向后兼容性

### settings.json 不存在

当 settings.json 不存在时：
- `userSettings` = nil
- `MergeWithPriority` 只合并 baseSettings 和 providerSettings
- 行为与之前一致

### 首次运行

首次运行 ccc（没有 settings.json）：
- 创建新的 settings.json
- 包含必要配置（hooks、disableAllHooks、allowManagedHooksOnly）
- 行为与之前一致

---

## 总结

### 核心原则

1. **settings.json 是用户配置的权威来源**，优先级最高
2. **ccc 只做最小必要干预**：确保 Supervisor 功能可用、避免 env 冲突
3. **使用统一的合并策略**：DeepMerge + 特殊字段处理
4. **保持向后兼容**：settings.json 不存在时，行为与之前一致

### 预期效果

- ✅ 用户通过 `/plugin install` 安装的插件被保留
- ✅ 用户手动修改的 permissions 配置被保留
- ✅ 用户配置的其他 hooks 不被覆盖
- ✅ Supervisor Stop hook 正确保存在
- ✅ env 冲突被避免
- ✅ 向后兼容
