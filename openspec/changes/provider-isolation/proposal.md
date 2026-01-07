# Proposal: provider-isolation

## 概述

通过环境变量隔离实现多进程独立提供商配置，无需修改全局 `settings.json`，每个进程读取独立的配置，互不干扰。

## 动机

### 当前问题

1. **单进程限制**: 全局 `settings.json` 只能配置一个提供商
2. **配置污染**: 切换提供商需要修改共享文件
3. **进程冲突**: 多个进程无法同时使用不同提供商
4. **不可测试**: 难以在测试中模拟不同提供商

### 新方案优势

1. **进程隔离**: 每个进程独立配置，互不影响
2. **环境变量驱动**: 通过环境变量控制，无需文件修改
3. **零配置**: 全局 `settings.json` 保持固定不变
4. **易于测试**: 测试中轻松切换提供商

## 变更内容

### 1. 设计原则

```
核心思想：全局 settings.json 保持固定，所有提供商配置通过环境变量传递

┌─────────────────────────────────────────────────────────┐
│           ~/.claude/settings.json (固定不变)              │
│  {                                                      │
│    "permissions": {...},                                │
│    "hooks": {                                           │
│      "Stop": "ccc supervisor-hook"                       │
│    }                                                    │
│  }                                                      │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  Process A: Kimi Provider                               │
│  ANTHROPIC_BASE_URL=https://api.moonshot.cn/anthropic  │
│  ANTHROPIC_AUTH_TOKEN=sk-kimi-xxx                       │
│  ANTHROPIC_MODEL=kimi-k2-thinking                       │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  Process B: GLM Provider                                │
│  ANTHROPIC_BASE_URL=https://open.bigmodel.cn/api       │
│  ANTHROPIC_AUTH_TOKEN=glm-xxx                           │
│  ANTHROPIC_MODEL=glm-4.7                                │
└─────────────────────────────────────────────────────────┘
```

### 2. 实现方式

#### 2.1 环境变量注入

`internal/provider/env.go`:

```go
package provider

import (
	"os"
	"os/exec"
	"path/filepath"
)

// LaunchWithProvider 启动 claude 进程，注入提供商环境变量
func LaunchWithProvider(cfg *Config, providerName string) error {
	provider, exists := cfg.Providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	// 构建环境变量
	env := buildProviderEnv(provider)

	// 设置全局固定配置
	env = append(env, buildFixedEnv(cfg)...)

	// 启动 claude 进程
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found: %w", err)
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Env = env

	return cmd.Run()
}

// buildProviderEnv 从提供商配置构建环境变量
func buildProviderEnv(provider map[string]interface{}) []string {
	envMap, ok := provider["env"].(map[string]interface{})
	if !ok {
		return nil
	}

	var env []string
	for k, v := range envMap {
		// 支持 ${VAR} 替换
		valStr := fmt.Sprintf("%v", v)
		valStr = os.ExpandEnv(valStr)
		env = append(env, fmt.Sprintf("%s=%s", k, valStr))
	}

	return env
}

// buildFixedEnv 构建固定的全局环境变量
func buildFixedEnv(cfg *Config) []string {
	var env []string

	// 从 settings 继承的环境变量
	if settingsEnv, ok := cfg.Settings["env"].(map[string]interface{}); ok {
		for k, v := range settingsEnv {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// 固定的 hook 配置
	env = append(env, "CCC_SUPERVISOR_HOOK=1")

	return env
}
```

#### 2.2 Supervisor Mode 优化

Supervisor hook 通过环境变量检测是否启用：

```go
// internal/cli/hook.go
func RunSupervisorHook(args []string) error {
	// 检查 supervisor 是否启用
	if !isSupervisorModeEnabled() {
		// 直接允许停止，不执行审查
		return nil
	}

	// 原有的 supervisor 逻辑...
}

func isSupervisorModeEnabled() bool {
	// 优先检查配置文件
	cfg, err := config.LoadSupervisorConfig()
	if err == nil && cfg.Enabled {
		return true
	}

	// 检查环境变量
	return os.Getenv("CCC_SUPERVISOR") == "1"
}
```

#### 2.3 配置生成优化

不再动态修改 `settings.json`，而是：

```go
// internal/provider/switch.go
func Switch(cfg *Config, providerName string) error {
	provider := cfg.Providers[providerName]

	// 生成临时 settings 文件（仅包含全局配置）
	settingsPath := config.GetSettingsPath()
	settings := map[string]interface{}{
		"permissions": cfg.Settings["permissions"],
		"hooks": map[string]interface{}{
			"Stop": "ccc supervisor-hook",
		},
	}

	// 写入固定 settings
	if err := config.SaveSettings(settings); err != nil {
		return err
	}

	// 通过环境变量启动 claude
	return provider.LaunchWithProvider(cfg, providerName)
}
```

### 3. 进程隔离架构

```
┌───────────────────────────────────────────────────────────┐
│                    ccc CLI (主进程)                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│  │ Kimi 配置    │  │ GLM 配置     │  │ M2 配置      │      │
│  └─────────────┘  └─────────────┘  └─────────────┘      │
│         │                │                │               │
│         ▼                ▼                ▼               │
│  ┌────────────────────────────────────────────────────┐ │
│  │       环境变量注入                                  │ │
│  │  - ANTHROPIC_BASE_URL                             │ │
│  │  - ANTHROPIC_AUTH_TOKEN                            │ │
│  │  - ANTHROPIC_MODEL                                 │ │
│  └────────────────────────────────────────────────────┘ │
│         │                                                │
│         ▼                                                │
│  ┌────────────────────────────────────────────────────┐ │
│  │       启动 claude 子进程                           │ │
│  │       (继承环境变量)                               │ │
│  └────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────┘

每个子进程独立运行，互不干扰
```

### 4. 多进程场景

#### 4.1 并行测试

```bash
# 终端 1: Kimi provider
CCC_CONFIG_DIR=./kimi-env ccc kimi

# 终端 2: GLM provider (同时运行)
CCC_CONFIG_DIR=./glm-env ccc glm

# 两个进程完全独立，互不影响
```

#### 4.2 Supervisor 子进程

```bash
# 主进程使用 Kimi
export ANTHROPIC_BASE_URL=https://api.moonshot.cn/anthropic
export ANTHROPIC_AUTH_TOKEN=sk-kimi-xxx
ccc kimi

# Supervisor hook 继承主进程环境变量
# 无需重新配置，自动使用 Kimi
```

### 5. 配置文件结构

保持 `ccc.json` 简洁：

```json
{
  "current_provider": "kimi",
  "providers": {
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "${KIMI_API_KEY}",
        "ANTHROPIC_MODEL": "kimi-k2-thinking"
      }
    },
    "glm": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "${GLM_API_KEY}",
        "ANTHROPIC_MODEL": "glm-4.7"
      }
    }
  }
}
```

**注意**：全局 `settings.json` 不再包含 `env` 字段，所有提供商配置通过环境变量传递。

## 影响范围

### 受影响代码

- `internal/provider/switch.go` - 改用环境变量注入
- `internal/cli/exec.go` - 简化，不需要生成 settings
- `internal/cli/hook.go` - 添加环境变量检测

### 不受影响

- 配置文件格式 (`ccc.json`)
- CLI 接口
- Supervisor Mode 逻辑

## 实施计划

### Phase 1: 环境变量注入 (1 周)
1. 实现 `buildProviderEnv()`
2. 实现 `LaunchWithProvider()`
3. 单元测试

### Phase 2: 优化配置生成 (1 周)
1. 移除动态 settings 生成
2. 生成固定全局 settings
3. 测试验证

### Phase 3: Supervisor 优化 (1 周)
1. 添加环境变量检测
2. 优化 hook 性能
3. 端到端测试

## 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 环境变量长度限制 | 低 | 使用配置文件存储敏感信息 |
| 进程启动开销 | 低 | 环境变量传递很快 |
| 兼容性 | 中 | 保持 CLI 接口不变 |
