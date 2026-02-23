# ccc - Claude Code 监督器

[English](README.md)

## 为什么选择 ccc？

`ccc` 是一个增强 Claude Code 的命令行工具，提供两大核心功能：

1. **Supervisor 模式**：自动任务审查，确保高质量可交付成果
2. **无缝提供商切换**：一条命令在 Kimi、GLM、MiniMax 等提供商之间切换

**优于 `ralph-claude-code`**：
- Supervisor 模式使用 Stop Hook 触发的审查机制，配合严格框架显著提高任务完成度和质量
- 与 ralph 基于信号的退出检测不同，ccc 的 Supervisor 会 fork 完整会话上下文来评估实际工作质量
- 这有效防止 AI 声称"完成"但结果质量差或仍有问题的虚假完成情况

## 快速开始

### 1. 安装

#### 选项 A：一键安装（Linux / macOS）

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]'); ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/'); curl -LO "https://github.com/guyskk/claude-code-supervisor/releases/latest/download/ccc-${OS}-${ARCH}" && sudo install -m 755 "ccc-${OS}-${ARCH}" /usr/local/bin/ccc && rm "ccc-${OS}-${ARCH}" && ccc --version
```

#### 选项 B：从 [Releases](https://github.com/guyskk/claude-code-supervisor/releases) 下载

下载适合你平台的二进制文件（`ccc-darwin-arm64`、`ccc-linux-amd64` 等）并安装到 `/usr/local/bin/`。

### 2. 配置

首次运行 `ccc` 时，如果已有 `~/.claude/settings.json`，会提示迁移并自动生成 `~/.claude/ccc.json`。

你也可以手动创建配置文件：

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
    }
  }
}
```

**安全警告**：`bypassPermissions` 允许 Claude Code 无需确认即可执行工具。仅在受信任的环境中使用。

### 3. 使用

```bash
# 查看帮助
ccc --help

# 切换到指定提供商并运行 Claude Code
ccc glm

# 使用当前提供商运行
ccc

# 传递任何 Claude Code 参数
ccc glm -p
```

### 4. 验证（可选）

验证你的提供商配置：

```bash
# 验证当前提供商
ccc validate

# 验证指定提供商
ccc validate glm

# 验证所有提供商
ccc validate --all
```

## Supervisor 模式（推荐）

Supervisor 模式是 `ccc` 最有价值的特性。它会在 Agent 每次停止后自动审查工作质量，如果未完成则提供反馈。

### 如何使用

1. 启动 `ccc`，与 Agent 确认需求和方案：

```bash
ccc
```

2. 使用斜杠命令启用 Supervisor 模式：

```
/supervisor 好，开始执行
```

3. Agent 执行任务，Supervisor 会在每次停止后自动审查
   - 如果工作未完成，Supervisor 提供反馈，Agent 继续执行
   - 重复直到 Supervisor 确认工作完成

### 工作原理

1. Agent 完成任务并停止，触发 Claude Code 的 Stop Hook
2. Supervisor（一个 Claude 实例）执行严格的审查工作
3. 如果工作未完成或质量不佳，Supervisor 提供反馈
4. Agent 根据反馈继续工作
5. 重复直到 Supervisor 确认工作完成

### 状态行显示

你可以在 Claude Code 中配置状态行来显示 Supervisor 模式状态：

```
/statusline 帮我配置一个状态行脚本，里面调用 `ccc supervisor-mode` 命令，这个命令会输出 on 或者 off，我希望显示成类似 ... | supervisor on 这样的效果。
```

## 替换命令：用 `ccc` 替换 `claude`

让 `ccc` 成为你默认的 Claude Code 命令。

```bash
# 用 ccc 替换 claude 命令（需要 sudo）
sudo ccc patch

# 替换后，`claude` 命令现在会调用 ccc
claude --help    # 显示 ccc 的帮助信息

# 恢复原始 claude 命令
sudo ccc patch --reset
```

## 配置说明

配置文件位置：`~/.claude/ccc.json`

### 配置字段

| 字段 | 说明 |
|------|------|
| `settings` | 所有提供商共享的 Claude Code 配置模板 |
| `providers.{name}` | 提供商特定的 Claude Code 配置 |

### 用户配置保护

**你的配置永远不会丢失！** ccc 会：
- 保留用户安装的插件
- 保留你手动编辑的 `settings.json`
- 保留你配置的其他 hooks

提供商的环境变量通过命令行传递，不会写入 `settings.json` 避免冲突。

## 开源许可证

MIT License - 详见 LICENSE 文件。
