# ccc - Claude Code 配置切换器

[English](README.md) | [中文文档](README-CN.md)

#### 推荐使用模型：

- **小米顶尖模型 MiMo V2.5-Pro 等，通过我的邀请码注册：双方各得 ¥10 API 体验金 + 首单 9 折。邀请码：2DZZC9。注册： https://platform.xiaomimimo.com?ref=2DZZC9 （注册后自动填入 · 体验金 40 天有效）**

- **智谱 Coding Plan，国内顶流编程大模型，20+主流工具全适配，性价比拉满！👉立即参与「拼好模」：https://www.bigmodel.cn/glm-coding?ic=S8PMS595T0**

## 为什么选择 ccc？

`ccc` 是一个为 Claude Code 提供无缝提供商切换的命令行工具。一条命令在 Kimi、GLM、MiniMax 等提供商之间切换。

## 快速开始

### 1. 安装

#### 选项 A：一键安装（Linux / macOS）

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]'); ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/'); curl -LO "https://github.com/guyskk/claude-code-config-switcher/releases/latest/download/ccc-${OS}-${ARCH}" && sudo install -m 755 "ccc-${OS}-${ARCH}" /usr/local/bin/ccc && rm "ccc-${OS}-${ARCH}" && ccc --version
```

#### 选项 B：从 [Releases](https://github.com/guyskk/claude-code-config-switcher/releases) 下载

下载适合你平台的二进制文件（`ccc-darwin-arm64`、`ccc-linux-amd64` 等）并安装到 `/usr/local/bin/`。

### 2. 配置

如果你已有 `~/.claude/settings.json`，首次运行 `ccc` 时会提示迁移并自动生成 ccc 配置 `~/.claude/ccc.json`。

你也可以自行创建配置文件，示例如下：

```json
{
  "settings": {
    "permissions": {
      "defaultMode": "bypassPermissions"
    }
  },
  "providers": {
    "glm": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "glm-4.7"
      }
    },
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "kimi-k2-thinking"
      }
    }
  }
}
```

> **安全警告**：`bypassPermissions` 允许 Claude Code 无需确认即可执行工具。仅在受信任的环境中使用。

### 3. 使用

```bash
# 查看帮助信息
ccc --help

# 切换到指定提供商并运行 Claude Code
ccc glm

# 使用当前提供商
ccc

# 传递任何 Claude Code 参数
ccc glm -p
```

### 4. 验证（可选）

验证提供商配置：

```bash
# 验证当前提供商
ccc validate

# 验证所有提供商
ccc validate --all
```

## 配置合并策略

运行 `ccc` 时，会读取你已有的 `settings.json` 并与 ccc.json 深度合并。优先级：**用户 `settings.json` > 提供商 > 基础 `settings`**。你手动编辑的配置、插件、hooks 都会被保留；提供商的环境变量通过命令行传递，不会写入 `settings.json`。

#### 环境变量处理

Claude Code 的 `settings.json` `env` 字段会**覆盖** ccc 启动 claude 时传入的环境变量。为确保 provider env 优先生效，ccc 通过 `--settings` CLI 参数传入 provider env，该参数优先级高于 `settings.json`。

这意味着：
- 你的 `settings.json` `env` 保持原样（ccc 不会修改它）
- provider env 通过 `--settings` 自动覆盖冲突的 key
- `settings.json` 中非冲突的 key 仍然正常工作

## Patch 命令：用 ccc 替代 `claude` 命令

通过替换系统中的 `claude` 命令，让任何调用 `claude` 的工具都使用配置了提供商的 `ccc` 命令。

```bash
# 用 ccc 替换 claude 命令（需要 sudo 权限）
sudo ccc patch

# 替换后，`claude` 命令现在会调用 ccc
claude --help    # 显示 ccc 的帮助信息

# 恢复原始 claude 命令
sudo ccc patch --reset
```

## 配置说明

配置文件位置，默认为：`~/.claude/ccc.json`

### 完整配置示例

```json
{
  "settings": {
    "permissions": {
      "defaultMode": "bypassPermissions"
    },
    "alwaysThinkingEnabled": true
  },
  "claude_args": ["--verbose"],
  "current_provider": "glm",
  "providers": {
    "glm": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "glm-4.7"
      }
    },
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "kimi-k2-thinking",
        "ANTHROPIC_SMALL_FAST_MODEL": "kimi-k2-0905-preview"
      }
    }
  }
}
```

### 配置字段说明

| 字段               | 说明                                  |
| ------------------ | ------------------------------------- |
| `settings`         | 所有提供商共享的 Claude Code 配置模板 |
| `claude_args`      | 固定传递给 Claude Code 的参数（可选） |
| `current_provider` | 当前使用的提供商（由 ccc 自动管理）   |
| `providers.{name}` | 提供商特定的 Claude Code 配置         |

### 提供商配置

每个提供商只需指定要覆盖的字段。常用字段：

| 字段                             | 说明               |
| -------------------------------- | ------------------ |
| `env.ANTHROPIC_BASE_URL`         | API 端点 URL       |
| `env.ANTHROPIC_AUTH_TOKEN`       | API 密钥/令牌      |
| `env.ANTHROPIC_MODEL`            | 使用的主模型       |
| `env.ANTHROPIC_SMALL_FAST_MODEL` | 快速任务使用的模型 |

**合并方式**：提供商设置与基础模板深度合并。提供商的 `env` 优先于 `settings.env`。

### 环境变量

| 变量             | 说明                                       |
| ---------------- | ------------------------------------------ |
| `CCC_CONFIG_DIR` | 覆盖配置目录（默认：`~/.claude/`）         |

```bash
# 使用自定义配置目录调试
CCC_CONFIG_DIR=./tmp ccc glm
```

## 从源码构建

```bash
# 构建所有平台
./build.sh --all

# 构建指定平台
./build.sh -p darwin-arm64,linux-amd64

# 自定义输出目录
./build.sh -o ./bin
```

**支持的平台：** `darwin-amd64`、`darwin-arm64`、`linux-amd64`、`linux-arm64`

## 开源许可证

MIT License - 详见 LICENSE 文件。
