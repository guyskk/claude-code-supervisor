# Claude Code 配置切换器

[English](README.md)

**一条命令在多个 Claude Code 提供商（Kimi、GLM、MiniMax 等）之间切换。**

## 简介

`ccc` 是一个命令行工具，让你在不同 Claude Code API 提供商配置之间无缝切换。无需手动编辑配置文件——只需运行 `ccc <provider>` 即可。

## 功能特性

- 一条命令切换提供商（Kimi、GLM、MiniMax 等）
- 自动合并提供商配置
- 透传所有 Claude Code 参数
- 支持自定义配置目录调试
- 简洁直观的命令行界面

## 安装

### 下载预编译版本

预编译的二进制文件可在 [Releases 页面](https://github.com/guyskk/claude-code-config-switcher/releases) 下载。

```bash
# 下载你平台的版本
curl -LO https://github.com/guyskk/claude-code-config-switcher/releases/latest/download/ccc-$(uname -s)-$(uname -m)

# 安装到系统目录
sudo chmod +x ccc-$(uname -s)-$(uname -m)
sudo mv ccc-$(uname -s)-$(uname -m) /usr/local/bin/ccc

# 验证安装
ccc --version
```

**支持的平台：** `darwin-amd64`、`darwin-arm64`、`linux-amd64`、`linux-arm64`、`windows-amd64.exe`

### 从源码构建

```bash
# 构建所有平台
./build.sh --all

# 构建指定平台
./build.sh -p darwin-arm64,linux-amd64

# 自定义输出目录
./build.sh -o ./bin
```

**支持的平台：** `darwin-amd64`、`darwin-arm64`、`linux-amd64`、`linux-arm64`、`windows-amd64`

## 配置

创建 `~/.claude/ccc.json`：

```json
{
  "settings": {
    "permissions": {
      "allow": ["Edit", "MultiEdit", "Write", "WebFetch", "WebSearch"],
      "defaultMode": "acceptEdits"
    },
    "alwaysThinkingEnabled": true,
    "env": {
      "API_TIMEOUT_MS": "300000",
      "DISABLE_TELEMETRY": "1",
      "DISABLE_ERROR_REPORTING": "1",
      "DISABLE_NON_ESSENTIAL_MODEL_CALLS": "1",
      "DISABLE_BUG_COMMAND": "1",
      "DISABLE_COST_WARNINGS": "1"
    }
  },
  "current_provider": "kimi",
  "providers": {
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "kimi-k2-thinking",
        "ANTHROPIC_SMALL_FAST_MODEL": "kimi-k2-0905-preview"
      }
    },
    "glm": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "glm-4.7",
        "ANTHROPIC_SMALL_FAST_MODEL": "glm-4.7"
      }
    },
    "m2": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "MiniMax-M2.1",
        "ANTHROPIC_SMALL_FAST_MODEL": "MiniMax-M2.1"
      }
    }
  }
}
```

**配置结构：**
- `settings` — 所有提供商共享的基础模板
- `current_provider` — 最后使用的提供商（自动更新）
- `providers` — 提供商特定配置

**工作原理：** 切换提供商时，`ccc` 会将提供商配置与基础模板深度合并，然后保存到 `~/.claude/settings-{provider}.json`。

更多示例见 `./tmp/example/` 目录。

## 使用方法

```bash
# 显示可用提供商
ccc --help

# 使用当前提供商运行
ccc

# 切换到指定提供商
ccc kimi

# 传递参数给 Claude Code
ccc kimi --help
ccc kimi /path/to/project
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `CCC_CONFIG_DIR` | 覆盖配置目录（默认：`~/.claude/`） |

```bash
# 使用自定义配置目录调试
CCC_CONFIG_DIR=./tmp ccc kimi
```

## 提供商管理

使用 `ccc provider` 子命令管理 API 提供商配置，无需手动编辑 JSON 文件。

### 列出所有提供商

```bash
ccc provider list
```

显示所有已配置的提供商及其 BASE_URL 和 MODEL。当前提供商用 `*` 标记。

### 添加提供商

**交互式模式**（推荐首次配置使用）：
```bash
ccc provider add openai
```

系统会逐步提示您输入：
- ANTHROPIC_BASE_URL（必须是 HTTPS）
- ANTHROPIC_AUTH_TOKEN
- ANTHROPIC_MODEL
- ANTHROPIC_SMALL_FAST_MODEL（可选）

**非交互式模式**（适用于脚本和自动化）：
```bash
ccc provider add openai \
  --base-url=https://api.openai.com/v1 \
  --token=sk-your-token-here \
  --model=gpt-4 \
  --small-model=gpt-3.5-turbo
```

### 显示提供商详情

```bash
ccc provider show kimi
```

显示提供商的配置信息，敏感的 token 会自动脱敏显示。

### 更新提供商配置

```bash
ccc provider set kimi ANTHROPIC_MODEL kimi-k1.5
```

更新提供商的某个环境变量。如果是当前提供商，配置文件会自动重新生成。

### 删除提供商

```bash
ccc provider remove old-provider
```

从配置中删除提供商。以下情况无法删除：
- 当前正在使用的提供商（请先切换到其他提供商）
- 最后一个提供商（至少需要保留一个）

### 提供商命名规则

提供商名称必须：
- 仅包含小写字母、数字、连字符和下划线
- 不能为空

示例：`kimi`、`glm-4`、`mini_max`、`openai`

