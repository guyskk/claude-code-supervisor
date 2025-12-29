# Claude Code 配置切换器

[English](README.md)

**一条命令在多个 Claude Code 提供商（Kimi、GLM、MiniMax 等）之间切换。**

## 简介

`ccc` 是一个命令行工具，让你在不同 Claude Code API 提供商配置之间无缝切换。无需手动编辑配置文件——只需运行 `ccc <provider>` 即可。

## 功能特性

- 一条命令切换提供商（Kimi、GLM、MiniMax 等）
- 自动合并提供商配置
- 配置验证和 API 连通性测试
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

# 验证当前提供商配置
ccc validate

# 验证指定提供商
ccc validate kimi

# 验证所有提供商
ccc validate --all

# 传递参数给 Claude Code
ccc kimi --help
ccc kimi /path/to/project
```

### 配置验证命令

`ccc validate` 命令帮助您验证提供商配置：

- **检查配置格式**：验证 JSON 语法和必需字段
- **验证环境变量**：确保 `ANTHROPIC_BASE_URL` 和 `ANTHROPIC_AUTH_TOKEN` 存在
- **测试 API 连通性**：尝试连接提供商的 API 端点
- **显示详细结果**：彩色输出（绿色=有效，红色=无效，黄色=警告）

示例输出：
```
$ ccc validate --all
Validating 3 provider(s)...

  Valid: kimi
    Base URL: https://api.moonshot.cn/anthropic
    Model: kimi-k2-thinking
    API connection: OK

  Valid: glm
    Base URL: https://open.bigmodel.cn/api/anthropic
    Model: glm-4.7
    API connection: OK

  Warning: m2
    Base URL: https://api.minimaxi.com/anthropic
    Model: MiniMax-M2.1
    API connection: HTTP 503: <错误信息>

2/3 providers valid, 1 with API warnings
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `CCC_CONFIG_DIR` | 覆盖配置目录（默认：`~/.claude/`） |

```bash
# 使用自定义配置目录调试
CCC_CONFIG_DIR=./tmp ccc kimi
```
