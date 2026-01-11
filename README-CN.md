# ccc - Claude Code 监督器

[English](README.md) | [中文文档](README-CN.md)

## 为什么选择 ccc？

`ccc` 是一个增强 Claude Code 的命令行工具，提供两大核心功能：

1. **Supervisor 模式**: ⭐ 自动任务审查，确保高质量、可交付的成果
2. **无缝提供商切换**: 一条命令在 Kimi、GLM、MiniMax 等提供商之间切换

**优于 `ralph-claude-code`**：

- Supervisor 模式使用 Stop Hook 触发的审查机制配合严格的六步框架，显著提高任务完成度和质量。
- 与 ralph 基于信号的退出检测不同，ccc 的 Supervisor 会 Fork 完整的会话上下文来评估实际工作质量。
- 这有效防止了 AI 声称"完成"但结果质量差、仍有很多问题的虚假完成情况。

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
  "supervisor": {
    "enabled": true
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
> **Tokens 消耗**：`supervisor.enabled` 开启 Supervisor 模式能显著提高任务完成度，但也会显著增加 Tokens 消耗量。

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

## Supervisor 模式（推荐）

Supervisor 模式是 `ccc` 最有价值的特性。它会在 Agent 每次停止后自动审查工作质量，如果未完成则提供反馈让 Agent 继续执行。

### 启用 Supervisor 模式

**默认方式（配置文件）**：在 `ccc.json` 中设置 `supervisor.enabled: true`（参见上方快速开始）。

**临时覆盖**：使用 `CCC_SUPERVISOR` 环境变量临时覆盖配置：

```bash
# 强制启用（即使配置中 enabled = false）
export CCC_SUPERVISOR=1
ccc

# 强制禁用（即使配置中 enabled = true）
export CCC_SUPERVISOR=0
ccc
```

### 工作原理

1. Agent 完成任务并停止，触发 Claude Code 的 Stop Hook
2. Supervisor（一个 Claude 实例）执行严格的审查工作
3. 如果工作未完成或质量不佳，Supervisor 提供反馈
4. Agent 根据反馈继续工作
5. 重复直到 Supervisor 确认工作完成

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
  "supervisor": {
    "enabled": true,
    "max_iterations": 20,
    "timeout_seconds": 600
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
| `supervisor`       | Supervisor 模式配置（可选）           |
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

### Supervisor 配置

| 字段              | 说明                             | 默认值  |
| ----------------- | -------------------------------- | ------- |
| `enabled`         | 启用 Supervisor 模式             | `false` |
| `max_iterations`  | 当前会话强制停止前的最大迭代次数 | `20`    |
| `timeout_seconds` | 每次 supervisor 调用的超时时间   | `600`   |

可通过 `CCC_SUPERVISOR=1` 环境变量覆盖 Supervisor 模式设置。

### 自定义 Supervisor 提示词

创建 `~/.claude/SUPERVISOR.md` 来自定义 Supervisor 提示词。此文件会使用你自己的指令覆盖默认的审查行为。

### 环境变量

| 变量             | 说明                                           |
| ---------------- | ---------------------------------------------- |
| `CCC_CONFIG_DIR` | 覆盖配置目录（默认：`~/.claude/`）             |
| `CCC_SUPERVISOR` | 启用 Supervisor 模式（`"1"` 启用，`"0"` 禁用） |

```bash
# 使用自定义配置目录调试
CCC_CONFIG_DIR=./tmp ccc glm

# 启用 Supervisor 模式
export CCC_SUPERVISOR=1
ccc glm
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
