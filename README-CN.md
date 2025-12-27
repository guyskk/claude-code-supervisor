# Claude Code 配置切换器 (ccc)

一个用于在不同 Claude Code 配置之间切换的命令行工具。

## 简介

`ccc`（Claude Code Config）允许你在不同的 Claude Code 提供商配置（如 Kimi、GLM、MiniMax）之间轻松切换，无需手动编辑配置文件。

## 功能特性

- 使用单个命令即可在多个 Claude Code 配置之间切换
- 自动更新 `current_provider` 设置
- 将所有参数传递给 Claude Code
- 支持使用自定义配置目录的调试模式
- 简单直观的命令行界面
- 在帮助信息中显示可用的提供商和当前提供商

## 安装

### 从源码构建

构建工具：

```bash
./build.sh
```

### 构建选项

构建脚本支持多种平台和选项：

```bash
# 仅构建当前平台（默认）
./build.sh

# 构建所有支持的平台
./build.sh --all

# 构建特定平台（逗号分隔）
./build.sh -p darwin-arm64,linux-amd64

# 指定输出目录
./build.sh -o ./bin

# 指定二进制文件名
./build.sh -n myccc
```

**支持的平台：**
- `darwin-amd64` - macOS x86_64
- `darwin-arm64` - macOS ARM64 (Apple Silicon)
- `linux-amd64` - Linux x86_64
- `linux-arm64` - Linux ARM64
- `windows-amd64` - Windows x86_64

### 系统级安装

```bash
# 为当前平台安装
sudo cp dist/ccc-darwin-arm64 /usr/local/bin/ccc

# 或为特定平台安装
sudo cp dist/ccc-linux-amd64 /usr/local/bin/ccc
```

## 配置

创建 `~/.claude/ccc.json` 配置文件：

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

配置结构说明：
- `settings`：所有提供商共享的基础设置模板
- `current_provider`：最后使用的提供商（自动更新）
- `providers`：特定提供商的设置，将与基础模板合并

切换提供商时，工具会：
1. 从基础 `settings` 开始
2. 将提供商的设置深度合并到基础设置之上
3. 提供商设置对于相同的键会覆盖基础设置
4. 将合并结果保存到 `~/.claude/settings-{provider}.json`

示例配置文件位于 `./tmp/example/` 目录中。

## 使用方法

### 基本命令

```bash
# 显示帮助信息（显示可用的提供商）
ccc --help

# 使用当前提供商运行
ccc

# 切换到并使用特定提供商运行
ccc kimi

# 将参数传递给 Claude Code
ccc kimi --help
ccc kimi /path/to/project

# 如果未设置 current_provider，则使用第一个提供商
ccc
```

### 环境变量

- `CCC_CONFIG_DIR`：覆盖配置目录（默认：`~/.claude/`）

调试时非常有用：
```bash
CCC_CONFIG_DIR=./tmp ccc kimi
```

### 提供商切换原理

1. `ccc` 读取 `~/.claude/ccc.json` 配置
2. 将所选提供商的设置与基础设置模板深度合并
3. 将合并的配置写入 `~/.claude/settings-{provider}.json`
4. 更新 `ccc.json` 中的 `current_provider` 字段
5. 执行 `claude --settings ~/.claude/settings-{provider}.json [additional-args...]`

配置合并是递归的，因此像 `env` 和 `permissions` 这样的嵌套对象会被正确合并。

每个提供商都有自己的设置文件（如 `settings-kimi.json`、`settings-glm.json`），让你可以轻松查看和管理不同的配置。

## 命令行参考

```
用法: ccc [provider] [args...]

Claude Code 配置切换器

命令：
  ccc              使用当前提供商（如果未设置则使用第一个提供商）
  ccc <provider>   切换到指定提供商并运行 Claude Code
  ccc --help       显示此帮助信息（显示可用的提供商）

环境变量：
  CCC_CONFIG_DIR   覆盖配置目录（默认：~/.claude/）

示例：
  ccc              使用当前提供商运行 Claude Code
  ccc kimi         切换到 'kimi' 提供商并运行 Claude Code
  ccc glm          切换到 'glm' 提供商并运行 Claude Code
  ccc m2           切换到 'm2'（MiniMax）提供商并运行 Claude Code
  ccc kimi --help  切换到 'kimi' 并将 --help 传递给 Claude Code
```
