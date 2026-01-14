# Project Context

## Purpose

**Claude Code Config Switcher (ccc)** 是一个轻量级 CLI 工具，用于在多个 Claude Code API 提供商配置之间无缝切换。用户可以通过一条命令在 Kimi（月之暗面）、GLM（智谱 AI）、MiniMax 等提供商之间切换，无需手动编辑配置文件。

### 核心目标
- **简单易用**: 一条命令完成提供商切换
- **独立可执行**: 单一静态链接二进制文件，无运行时外部依赖
- **跨平台支持**: 支持 macOS 和 Linux（amd64/arm64）
- **安全可靠**: 配置深度合并，支持提供商特定覆盖

## Tech Stack

- **语言**: Go 1.21
- **构建系统**: GNU Bash (`build.sh`)，支持交叉编译
- **分发方式**: 单一静态二进制文件（所有依赖在构建时静态链接）
- **CI/CD**: GitHub Actions
- **支持平台**: darwin-amd64, darwin-arm64, linux-amd64, linux-arm64

## Project Conventions

### Code Style

- **格式化**: 使用 `gofmt`（CI 强制执行）
- **静态检查**: 使用 `go vet`（CI 强制执行）
- **命名规范**: 标准 Go 约定
  - 导出: `PascalCase`
  - 私有: `camelCase`
  - 常量: `PascalCase` 或 `UPPER_SNAKE_CASE`
- **文件结构**: 单文件应用 (`main.go`)
- **注释**: 导出函数/类型使用 Go doc 注释

### Architecture Patterns

- **单二进制分发**: 所有依赖静态链接到一个可执行文件中
- **配置驱动**: 基于 JSON 的配置 (`~/.claude/ccc.json`)
- **深度合并策略**: 提供商设置深度合并到基础模板
  - 基础 `settings` → 提供商 `env` → `settings-{provider}.json`
- **依赖注入**: 函数显式接受配置/参数
- **错误处理**: 显式错误返回并携带上下文，使用 `fmt.Errorf` 包装

### Testing Strategy

- **单元测试**: `go test ./...`
- **竞态检测**: `go test -race ./...`（CI 必需）
- **测试隔离**: 使用 `CCC_CONFIG_DIR` 测试，不影响用户配置
- **Mock**: 通过覆盖 `getClaudeDirFunc` 进行测试

### Git Workflow

- **主分支**: `main`
- **Pull Request**: 所有更改必须通过 PR
- **提交约定**: 使用 Conventional Commits 前缀
  - `feat:` - 新功能
  - `fix:` - Bug 修复
  - `docs:` - 文档
  - `ci:` - CI/CD 变更
  - `build:` - 构建系统变更
  - `refactor:` - 重构
- **CI 门禁**: Lint → Build → Test 必须全部通过

## Domain Context

### Claude Code 配置

Claude Code CLI 使用 `settings.json` 文件，结构如下：

```json
{
  "permissions": { "allow": [...], "defaultMode": "..." },
  "alwaysThinkingEnabled": true,
  "env": {
    "ANTHROPIC_BASE_URL": "...",
    "ANTHROPIC_AUTH_TOKEN": "...",
    "ANTHROPIC_MODEL": "..."
  }
}
```

### 提供商切换模型

1. **基础模板**: `ccc.json` 中的共享设置 → `settings`
2. **提供商覆盖**: `ccc.json` 中的提供商特定环境变量 → `providers.{name}`
3. **合并输出**: 深度合并结果写入 `~/.claude/settings-{provider}.json`
4. **执行**: `claude --settings ~/.claude/settings-{provider}.json`

### 支持的提供商

| 名称 | 服务 | Base URL |
|------|---------|----------|
| kimi | Moonshot（月之暗面） | https://api.moonshot.cn/anthropic |
| glm | 智谱 AI | https://open.bigmodel.cn/api/anthropic |
| m2 | MiniMax | https://api.minimaxi.com/anthropic |

## Important Constraints

- **单二进制分发**: 最终产物必须是独立的可执行文件，无需运行时外部依赖
- **向后兼容**: 配置文件格式变更应保持向后兼容
- **不修改 Claude Code**: 不能修改 Claude Code CLI 本身
- **静态链接**: 所有依赖必须静态链接（Go 默认行为）

## External Dependencies

### 运行时依赖
- **claude CLI**: 必须安装且在 `$PATH` 中可用
- **配置目录**: `~/.claude/`（或通过 `CCC_CONFIG_DIR` 指定）

### 构建依赖
- **Go 1.21+**: 用于构建二进制文件
- **git**: 用于版本注入（commit hash）

### API 提供商（用户配置）
- Moonshot (Kimi): https://api.moonshot.cn
- 智谱 AI (GLM): https://open.bigmodel.cn
- MiniMax: https://api.minimaxi.com
