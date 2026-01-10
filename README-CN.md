# Claude Code 配置切换器

[English](README.md)

**一条命令在多个 Claude Code 提供商（Kimi、GLM、MiniMax 等）之间切换。**

---

## 为什么选择 ccc？

`ccc` 是一个增强 Claude Code 的命令行工具，提供两大核心功能：

1. **无缝提供商切换** - 一条命令在 Kimi、GLM、MiniMax 等提供商之间切换
2. **Supervisor 模式** - 自动任务审查和迭代，确保高质量、可交付的成果

与 `ralph-claude-code` 不同，Supervisor 模式使用严格的六步审查框架，能发现"只问不做"、"只计划不执行"、"缺少集成测试"等常见问题。

---

## 快速开始（5 分钟）

### 1. 安装

```bash
# Linux / macOS（自动检测平台）
OS=$(uname -s | tr '[:upper:]' '[:lower:]'); ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/'); curl -LO "https://github.com/guyskk/claude-code-config-switcher/releases/latest/download/ccc-${OS}-${ARCH}" && sudo install -m 755 "ccc-${OS}-${ARCH}" /usr/local/bin/ccc && rm "ccc-${OS}-${ARCH}" && ccc --version
```

### 2. 配置

创建 `~/.claude/ccc.json`：

```json
{
  "settings": {
    "permissions": {
      "allow": ["Edit", "MultiEdit", "Write", "WebFetch", "WebSearch"],
      "defaultMode": "acceptEdits"
    }
  },
  "providers": {
    "kimi": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "kimi-k2-thinking"
      }
    },
    "glm": {
      "env": {
        "ANTHROPIC_BASE_URL": "https://open.bigmodel.cn/api/anthropic",
        "ANTHROPIC_AUTH_TOKEN": "YOUR_API_KEY_HERE",
        "ANTHROPIC_MODEL": "glm-4.7"
      }
    }
  }
}
```

> **注意**：这是快速上手的最小化配置。完整配置选项（包括高级设置）请参阅下方的[配置](#配置)章节。

### 3. 使用

```bash
# 切换到指定提供商并运行 Claude Code
ccc kimi

# 使用当前提供商
ccc

# 传递任何 Claude Code 参数
ccc glm --help
ccc kimi /path/to/project
```

---

## Supervisor 模式（推荐）

Supervisor 模式是 `ccc` 最有价值的特性。它会在 Agent 每次停止后自动审查工作质量，如果未完成则提供反馈。

### 启用 Supervisor 模式

```bash
export CCC_SUPERVISOR=1
ccc kimi
```

### 工作原理

1. Agent 完成任务并尝试停止
2. Supervisor（一个 Claude 实例）使用严格的六步框架审查工作
3. 如果工作未完成或质量不佳，Supervisor 提供反馈
4. Agent 根据反馈继续工作
5. 重复直到 Supervisor 确认工作完成

### Supervisor 审查内容

Supervisor 使用综合审查框架：

| 步骤 | 检查内容 |
|------|---------|
| 1 | 理解用户原始需求 |
| 2 | 验证实际执行了工作（不只是提问/计划） |
| 3 | 检查常见陷阱（只问不做、测试循环、虚假完成） |
| 4 | 评估代码质量（无 TODO、有自我审查、有测试） |
| 5 | 确保可交付性（集成测试、可部署） |
| 6 | 提供建设性反馈 |

### 核心优势

- **捕获"只问不做"** - 识别只提问不执行的 Agent
- **要求自我审查** - 代码必须经过 Agent 自身审查
- **要求集成测试** - 不接受"应该可以"，必须验证
- **防止过早停止** - Agent 必须迭代直到质量达标
- **最多 20 次迭代** - 防止无限循环

### 示例输出

```
[supervisor] starting supervisor review
[supervisor] iteration count: 1/20
[supervisor] supervisor review completed
[supervisor] work not satisfactory, agent will continue
[supervisor] feedback: 代码中有 TODO 注释。请完成所有待办事项并添加集成测试后再停止。
```

### 日志

Supervisor 日志保存在 `~/.claude/ccc/supervisor-{id}.log` 供调试使用。

---

## 核心功能

### 提供商切换

```bash
# 切换到指定提供商
ccc kimi    # 切换到 Kimi（月之暗面）
ccc glm     # 切换到 GLM（智谱 AI）
ccc m2      # 切换到 MiniMax

# 使用当前提供商（或第一个可用）
ccc

# 查看可用提供商
ccc --help
```

### 配置验证

```bash
# 验证当前提供商
ccc validate

# 验证指定提供商
ccc validate kimi

# 验证所有提供商（并行检查）
ccc validate --all
```

输出示例：
```
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
    API connection: HTTP 503: Service unavailable

All providers valid (1 with API warnings)
```

### 替代 claude 命令

`ccc` 可以完全替代你的工作流中的 `claude` 命令：

```bash
# 替代: claude --help
ccc --help

# 替代: claude /path/to/project
ccc /path/to/project

# 替代: claude --debug --verbose
ccc --debug --verbose
```

所有参数都会原样传递给 Claude Code。

---

## 配置

### 配置文件位置

默认：`~/.claude/ccc.json`
通过环境变量覆盖：`CCC_CONFIG_DIR`

### 配置结构

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
  "claude_args": ["--verbose", "--debug"],
  "current_provider": "kimi",
  "providers": {
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

| 字段 | 说明 |
|------|------|
| `settings` | 所有提供商共享的基础模板 |
| `claude_args` | 固定传递给 Claude Code 的参数（可选） |
| `current_provider` | 最后使用的提供商（自动更新） |
| `providers` | 提供商特定配置 |

**合并方式**：提供商设置与基础模板深度合并。提供商的 `env` 优先于 `settings.env`。

### 自动迁移

如果你已有 `~/.claude/settings.json`，`ccc` 可以在首次运行时自动迁移：

```bash
ccc

# 提示："Would you like to create ccc config from existing settings? [y/N]"
# 按 'y' 确认迁移
```

迁移行为：
- `settings.json` 中的 `env` 字段 → `providers.default.env`
- 其他字段 → `settings`（基础模板）

---

## 环境变量

| 变量 | 说明 |
|------|------|
| `CCC_CONFIG_DIR` | 覆盖配置目录（默认：`~/.claude/`） |
| `CCC_SUPERVISOR` | 启用 Supervisor 模式（`"1"` 启用，`"0"` 禁用） |

```bash
# 使用自定义配置目录调试
CCC_CONFIG_DIR=./tmp ccc kimi

# 启用 Supervisor 模式
export CCC_SUPERVISOR=1
ccc kimi
```

---

## 高级用法

### Supervisor 配置

可以在 `~/.claude/ccc-supervisor.json` 中配置 Supervisor 行为（可选）：

```json
{
  "enabled": true,
  "max_iterations": 20,
  "timeout_seconds": 600
}
```

### 自定义 Supervisor 提示词

创建 `~/.claude/SUPERVISOR.md` 来自定义 Supervisor 提示词。默认提示词参见 `internal/cli/supervisor_prompt_default.md`。

---

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

---

## 许可证

MIT License - 详见 LICENSE 文件。
