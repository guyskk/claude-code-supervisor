# 讨论：完全移除 Supervisor 功能

> 日期：2026-05-19
> 状态：讨论中

## 背景

用户认为 supervisor 功能已完全被 Claude Code 自身的 goal 和 loop 功能取代，不再需要 ccc 项目中的 supervisor 功能。需要全面清理。

## 需求理解

**目标**：完全移除 supervisor 相关的所有代码、配置、测试、文档和依赖。

**保留**：Provider 切换功能（这是 ccc 的另一核心价值，独立于 supervisor）。

## 探索发现

### 项目双核心功能

ccc 有两个核心功能：
1. **Supervisor Mode** -- Agent 停止时自动审查工作质量（要移除）
2. **Provider Switching** -- 多 API 供应商之间切换（保留）

### Supervisor 功能涉及的范围

#### 1. 需要整个删除的包/目录

| 路径 | 原因 |
|------|------|
| `internal/supervisor/` (state.go, logger.go, output.go + 测试) | supervisor 核心运行时 |
| `internal/llmparser/` (parser.go + 测试) | 仅 supervisor hook 使用的容错 JSON 解析 |
| `cmd/test-sdk/` | SDK 诊断工具 |
| `cmd/test-sdk-auto-compact/` | SDK compact 测试 |
| `cmd/test-sdk-structured/` | SDK structured output 测试 |

#### 2. 需要删除的单个文件

| 路径 | 原因 |
|------|------|
| `internal/cli/hook.go` | supervisor hook 核心逻辑 |
| `internal/cli/supervisor_prompt_default.md` | 嵌入的默认审查 prompt |
| `internal/cli/supervisor_mode_test.go` | supervisor mode 测试 |
| `internal/cli/hook_test.go` | hook 测试 |
| `internal/config/supervisor.go` | SupervisorConfig 定义 |
| `internal/config/supervisor_test.go` | supervisor 配置测试 |
| `internal/config/supervisor_integration_test.go` | supervisor 集成测试 |

#### 3. 需要删除的文档

| 路径 | 原因 |
|------|------|
| `docs/supervisor-mode-improvement-proposal.md` | supervisor 改进方案 |
| `docs/supervisor-planning-integration.md` | supervisor 规划集成 |
| `docs/LLM_JSON_PARSER_IMPLEMENTATION_PLAN.md` | LLM 解析器实现计划 |
| `docs/STRUCTURED_OUTPUTS_IMPLEMENTATION_PLAN.md` | 结构化输出实现计划 |
| `docs/WORKING_STATE_PATTERN.md` | 工作状态模式 |
| `docs/working-state-discuss.md` | 工作状态讨论 |

#### 4. 需要修改的文件

| 路径 | 修改内容 |
|------|----------|
| `internal/cli/cli.go` | 移除 supervisor-hook、supervisor-mode 子命令；移除 --debug 标志（仅用于 supervisor 日志）；简化 help 文本 |
| `internal/cli/exec.go` | 移除 supervisor ID 生成（UUID）、日志文件打开、supervisor 相关环境变量 |
| `internal/config/config.go` | 移除 `EnsureStopHook()` 函数；从 Config 结构体移除 `Supervisor` 字段 |
| `internal/provider/provider.go` | 移除 Stop Hook 注入、移除 slash command 文件创建（supervisor.md、supervisoroff.md） |
| `go.mod` / `go.sum` | 移除 claude-agent-sdk-go、uuid、jsonrepair、gojsonschema 依赖 |
| `README.md` / `README-CN.md` | 移除 supervisor 相关文档 |
| `CHANGELOG.md` | 移除 supervisor 相关条目 |
| `CLAUDE.md` | 更新项目描述 |
| `check.sh` | 检查是否需要更新 |
| `.github/workflows/*` | 检查是否需要更新 |

#### 5. 可移除的 Go 依赖

| 依赖 | 原因 |
|------|------|
| `github.com/schlunsen/claude-agent-sdk-go` | 仅 supervisor hook 使用 SDK |
| `github.com/google/uuid` | 仅生成 supervisor session ID |
| `github.com/kaptinlin/jsonrepair` | 仅 llmparser 使用 |
| `github.com/xeipuuv/gojsonschema` | 仅 llmparser 使用 |
| `github.com/twpayne/go-expect` | PTY 交互测试（需确认是否有非 supervisor 用途） |

#### 6. 需要更新的 CLI 命令

移除后的命令结构：
```
ccc [--version] [-v]
ccc [--help] [-h]
ccc <provider> [claude_args...]      # 切换 provider 并启动 claude
ccc                                  # 使用当前 provider 启动 claude
ccc validate [--all] [provider]      # 验证配置
ccc patch [--reset]                  # 替换/恢复 claude 命令
```

移除的命令：
- `ccc supervisor-mode [on|off]` -- 已移除
- `ccc supervisor-hook [--session-id ID]` -- 已移除
- `ccc [--debug]` -- debug 标志仅用于显示 supervisor 日志路径

#### 7. 需要移除的环境变量

- `CCC_SUPERVISOR_ID` -- supervisor 会话 ID
- `CCC_SUPERVISOR_HOOK` -- 防递归标记

#### 8. 需要移除的文件系统路径

- `~/.claude/ccc/supervisor-{id}.json` -- 状态文件
- `~/.claude/ccc/supervisor-{id}.log` -- 日志文件
- `~/.claude/commands/supervisor.md` -- slash command
- `~/.claude/commands/supervisoroff.md` -- slash command
- `~/.claude/SUPERVISOR.md` -- 自定义 prompt

### 保留的功能（独立于 supervisor）

1. **Provider 切换** -- 核心功能
2. **配置管理** -- ccc.json/settings.json 读写
3. **迁移** -- 旧格式迁移
4. **验证** -- Provider 配置验证
5. **Pretty JSON** -- JSON 美化
6. **Patch** -- claude 命令替换

### 模块依赖变化

移除后的依赖图：
```
main.go
  └── cli
        ├── config          (配置加载/保存)
        ├── migration       (从旧格式迁移)
        ├── provider        (provider 切换)
        │     └── config    (provider 依赖 config)
        ├── validate        (配置验证)
        └── prettyjson      (美化 JSON)
```

## 讨论与决策

### Q1: settings.json 中残留的 Stop Hook 配置如何处理？
**决策**：将 `EnsureStopHook` 改为清理逻辑，在 SwitchWithHook 中主动清除 settings.json 中的 ccc Stop Hook 配置。

具体实现：
- `EnsureStopHook` 改名为 `RemoveStopHook`，功能从"注入 hook"变为"移除 hook"
- 清理 settings.json 中 `hooks.Stop` 里 command 包含 `ccc supervisor-hook` 的条目
- 如果清理后 hooks 为空 map，移除整个 hooks 字段
- 清理 `disableAllHooks` 和 `allowManagedHooksOnly` 字段（这两个是 supervisor 专用）

### Q2: slash command 文件（supervisor.md、supervisoroff.md）是否清理？
**决策**：清理。在 SwitchWithHook 中主动删除 `~/.claude/commands/supervisor.md` 和 `~/.claude/commands/supervisoroff.md`。

### Q3: cmd/ 目录下的 SDK 测试工具是否全部删除？
**决策**：全部删除。它们都是为 supervisor SDK 功能服务的。

### Q4: --debug 标志如何处理？
**决策**：移除。之前只通过环境变量控制 debug，不需要 CLI 参数。移除 Command.Debug 字段、Parse() 中的 --debug 扫描逻辑、exec.go 中的相关判断。

### Q5: docs/ 下的 supervisor 相关文档如何处理？
**决策**：封存不动，保留在仓库中作为历史记录。

### Q6: ccc.json 中的 supervisor 配置字段如何处理？
**决策**：从 Config 结构体中移除 Supervisor 字段。Go JSON 解析自动忽略多余字段，旧的 ccc.json 中的 supervisor 配置不影响。Save 时也不会再写入。

## 最终方案

### 删除文件清单（共 ~20 个）

**整个删除的目录：**
- `internal/supervisor/` -- 6 个文件（state.go, logger.go, output.go + 3 个测试）
- `internal/llmparser/` -- 2 个文件（parser.go + parser_test.go）
- `cmd/test-sdk/` -- 1 个文件
- `cmd/test-sdk-auto-compact/` -- 2 个文件
- `cmd/test-sdk-structured/` -- 1 个文件

**删除的单个文件：**
- `internal/cli/hook.go` -- supervisor hook 逻辑
- `internal/cli/supervisor_prompt_default.md` -- 嵌入 prompt
- `internal/cli/supervisor_mode_test.go` -- mode 测试
- `internal/cli/hook_test.go` -- hook 测试
- `internal/config/supervisor.go` -- SupervisorConfig
- `internal/config/supervisor_test.go` -- 配置测试
- `internal/config/supervisor_integration_test.go` -- 集成测试

### 修改文件清单（共 ~8 个）

**`internal/cli/cli.go`：**
- 移除 `supervisor` 包导入
- 移除 Command 结构体中的 Debug、SupervisorHook、SupervisorHookOpts、SupervisorMode、SupervisorModeOpts 字段
- 移除 SupervisorHookCommand、SupervisorModeCommand 类型定义
- 移除 parseSupervisorHookArgs、parseSupervisorModeArgs 函数
- 移除 RunSupervisorMode 函数
- 移除 Parse() 中的 --debug 扫描、supervisor-hook 和 supervisor-mode 分支
- 移除 Run() 中的 supervisor-mode 和 supervisor-hook 分发
- 更新 ShowHelp() 移除 Supervisor Mode 部分
- 更新项目描述文案

**`internal/cli/exec.go`：**
- 移除 `uuid`、`time` 和 `supervisor` 包导入
- 移除 supervisorID 生成逻辑（UUID + CCC_SUPERVISOR_ID 环境变量）
- 移除日志文件打开和写入逻辑
- 移除所有 cmd.Debug 相关逻辑
- 保留 checkSettingsEnvConflict（这是 provider 功能）
- 保留 determineProvider（这是 provider 功能）

**`internal/config/config.go`：**
- 移除 `stopHookTimeout` 常量
- 从 Config 结构体移除 `Supervisor *SupervisorConfig` 字段
- 将 `EnsureStopHook()` 改为 `RemoveStopHook()`：清除 settings 中的 ccc Stop hook
  - 移除 hooks.Stop 中 command 包含 `ccc supervisor-hook` 的条目
  - 如果 hooks.Stop 为空则删除 Stop 字段
  - 如果整个 hooks 为空则删除 hooks 字段
  - 删除 `disableAllHooks` 和 `allowManagedHooksOnly` 字段

**`internal/provider/provider.go`：**
- 移除 `EnsureStopHook` 调用，改为调用 `RemoveStopHook`
- 移除 `createSupervisorCommandFiles` 函数
- 改为调用清理 slash command 文件的逻辑：删除 `~/.claude/commands/supervisor.md` 和 `supervisoroff.md`
- 更新 SwitchWithHook 函数注释

**`go.mod`：**
- 移除 `github.com/schlunsen/claude-agent-sdk-go` 直接依赖和 replace 指令
- 移除 `github.com/google/uuid` 直接依赖
- 移除 `github.com/kaptinlin/jsonrepair` 直接依赖
- 移除 `github.com/xeipuuv/gojsonschema` 直接依赖
- 移除 `github.com/twpayne/go-expect` 直接依赖（用于 PTY 交互测试）
- 运行 `go mod tidy` 清理间接依赖

**`README.md` / `README-CN.md`：**
- 移除 supervisor 相关的所有描述
- 更新项目定位为 "Claude Code Configuration Switcher"
- 更新功能列表、使用说明、配置说明

**`CHANGELOG.md`：**
- 添加移除 supervisor 功能的 changelog 条目

**`CLAUDE.md`：**
- 更新项目描述

### 移除后的命令结构

```
ccc [--version] [-v]
ccc [--help] [-h]
ccc <provider> [claude_args...]      # 切换 provider 并启动 claude
ccc                                  # 使用当前 provider 启动 claude
ccc validate [--all] [provider]      # 验证配置
ccc patch [--reset]                  # 替换/恢复 claude 命令
```

### 移除后的依赖图

```
main.go
  └── cli
        ├── config          (配置加载/保存)
        ├── migration       (从旧格式迁移)
        ├── provider        (provider 切换)
        │     └── config
        ├── validate        (配置验证)
        └── prettyjson      (美化 JSON)
```

### 移除后的 go.mod 直接依赖

仅保留：
- `github.com/stretchr/testify` -- 测试框架
