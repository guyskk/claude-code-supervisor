# 实现方案：Supervisor Hook 支持 AskUserQuestion 工具调用审查

**分支**: `002-askuserquestion-hook` | **日期**: 2026-01-20 | **规格**: [spec.md](./spec.md)
**输入**: 来自 `/specs/002-askuserquestion-hook/spec.md` 的功能规格

## 特别说明：使用中文

**本文档必须使用中文编写。**

1. 所有技术描述、架构决策、实现细节必须使用中文。
2. 代码示例中的注释必须使用中文。
3. 变量名、函数名等标识符使用英文，但说明文字使用中文。

## 摘要

本功能扩展 Supervisor Hook 机制，使其不仅能在任务结束时（Stop 事件）进行审查，也能在任务执行过程中对关键交互（AskUserQuestion 工具调用）进行质量控制。

**主要需求**:
1. 在 Claude Code 配置中添加 PreToolUse hook，匹配 AskUserQuestion 工具
2. 扩展 hook 输入解析，支持 `tool_name` 和 `hook_event_name` 字段
3. 根据 `allow_stop` 决定返回 "allow" 或 "deny" 决策
4. 在 `permissionDecisionReason` 字段中填写 feedback 内容
5. 在 PreToolUse hook 触发时增加迭代计数

**技术方案摘要**:
- 扩展 `HookInput` 结构支持所有 hook 事件类型
- 添加 `HookOutput` 结构根据事件类型返回不同格式
- 在 `provider.go` 中添加 PreToolUse hook 配置
- 复用现有的 `ccc supervisor-hook` 命令
- 保持向后兼容，Stop hook 继续使用现有格式

## 技术上下文

**语言/版本**: Go 1.23
**主要依赖**:
- `github.com/schlunsen/claude-agent-sdk-go` - Claude Agent SDK
- 标准库：`encoding/json`, `fmt`, `os`, `log/slog`
- 项目内部包：`internal/config`, `internal/supervisor`, `internal/llmparser`, `internal/prettyjson`

**存储**:
- 配置文件：`~/.claude/settings.json` (Claude Code hooks 配置)
- 状态文件：`~/.claude/ccc/supervisor-{session_id}.json` (Supervisor 状态)
- Prompt 文件：`~/.claude/SUPERVISOR.md` 或内置 default prompt

**测试**: `go test` (单元测试), `go test -race` (竞态检测), `go test -tags=integration` (集成测试)

**目标平台**: CLI 工具，支持 darwin-amd64, darwin-arm64, linux-amd64, linux-arm64

**项目类型**: single (单一 Go 可执行文件)

**性能目标**:
- Hook 响应时间 < 30 秒
- 内存占用 < 50MB (hook 进程)
- 不影响 Claude Code 正常使用体验

**约束**:
- 单二进制分发（静态链接）
- 向后兼容（现有 Stop hook 功能不受影响）
- 跨平台支持（所有目标平台）

**规模/范围**: < 500 行新增代码，主要在 `internal/cli/hook.go` 和 `internal/provider/provider.go`

## 宪章检查

*门禁：必须在第 0 阶段研究前通过。第 1 阶段设计后再次检查。*

### ccc 项目宪章合规检查

- [x] **原则一：单二进制分发** - 最终产物是单一静态链接二进制文件
- [x] **原则二：代码质量标准** - 符合 gofmt、go vet 要求
- [x] **原则三：测试规范** - 包含单元测试和竞态检测
- [x] **原则四：向后兼容** - 配置格式变更保持兼容
- [x] **原则五：跨平台支持** - 支持 darwin/linux, amd64/arm64
- [x] **原则六：错误处理与可观测性** - 错误明确且可操作

### 复杂度跟踪

本功能无需任何宪章违规，所有实现都遵循现有原则和模式。

## 项目结构

### 文档组织（本功能）

```text
specs/002-askuserquestion-hook/
├── plan.md              # 本文件
├── research.md          # 技术研究结果
├── data-model.md        # 数据模型定义
├── quickstart.md        # 快速入门指南
├── contracts/           # API 契约
│   └── hook-input-output.md
├── checklists/          # 质量检查清单
│   └── requirements.md
└── tasks.md             # 任务分解（由 /speckit.tasks 生成）
```

### 源代码组织（仓库根目录）

```text
cmd/ccc/              # 主入口
├── main.go

internal/             # 私有应用代码
├── cli/              # CLI 命令处理（主要修改）
│   ├── cli.go
│   ├── hook.go       # [修改] 扩展输入输出格式
│   └── hook_test.go  # [新增/修改] 测试
├── config/           # 配置管理
├── provider/         # 提供商切换逻辑（主要修改）
│   └── provider.go   # [修改] 添加 PreToolUse hook 配置
└── supervisor/       # Supervisor 模式
    ├── state.go
    ├── output.go
    └── logger.go
```

**结构决策**: 使用现有的单一 Go 项目结构，主要修改 `internal/cli/hook.go` 和 `internal/provider/provider.go` 两个文件。

## 实现阶段

### 第 -1 阶段：预实现门禁

> **重要：在开始任何实现工作前必须通过此阶段**

#### 宪章合规门禁

- [x] 所有 6 条核心原则已检查
- [x] 如有违规，已在"复杂度跟踪"表中记录理由

#### 技术决策门禁

- [x] 技术栈已确定（语言、依赖）
- [x] 项目结构已定义
- [x] 数据模型已设计（data-model.md）
- [x] API 契约已定义（contracts/hook-input-output.md）

---

### 第 0 阶段：技术研究

**目标**: 调研技术选项，收集实现所需信息

**输出**: `research.md`

**研究内容**:
- [x] 可用的 Go 标准库和第三方包
- [x] Claude Code PreToolUse hook 的输入输出格式
- [x] 如何区分不同的 hook 事件类型
- [x] 向后兼容性策略
- [x] 测试框架和 Mock 工具

**状态**: 已完成

---

### 第 1 阶段：架构设计

**目标**: 定义数据模型、API 契约和实现细节

**输出**: `data-model.md`、`contracts/`、`quickstart.md`

**数据模型** (data-model.md):
- [x] 定义核心数据结构（HookInput, HookOutput, HookSpecificOutput）
- [x] 定义配置格式（PreToolUse hook 配置）
- [x] 定义错误处理契约

**API 契约** (contracts/hook-input-output.md):
- [x] 输入契约（Stop 和 PreToolUse 事件）
- [x] 输出契约（不同事件的输出格式）
- [x] 错误处理规范
- [x] 测试用例定义

**快速入门** (quickstart.md):
- [x] 关键验证场景
- [x] 测试检查清单
- [x] 调试技巧
- [x] 常见问题排查

**状态**: 已完成

---

### 第 2 阶段：任务分解

**目标**: 将设计转化为可执行的任务列表

**输出**: `tasks.md` (由 `/speckit.tasks` 命令生成)

> **注意**: 第 2 阶段不在此方案中完成，由独立的 `/speckit.tasks` 命令处理

---

## 实施文件创建顺序

> **重要：按照此顺序创建文件以确保质量**

1. **contracts/** - 首先定义 API 契约和接口
2. **测试文件** - 按以下顺序创建：
   - `internal/cli/hook_test.go` - 单元测试（测试输入解析、输出转换）
   - 集成测试（测试完整的 hook 流程）
3. **源代码文件** - 创建使测试通过的实现：
   - `internal/cli/hook.go` - 扩展输入输出格式
   - `internal/provider/provider.go` - 添加 PreToolUse hook 配置

**理由**: 测试先行确保 API 设计可用，实现符合需求。

## 实现细节

### 1. 数据结构定义

**位置**: `internal/cli/hook.go`

```go
// HookInput 支持所有 hook 事件类型
type HookInput struct {
    SessionID      string          `json:"session_id"`
    StopHookActive bool            `json:"stop_hook_active,omitempty"`
    HookEventName  string          `json:"hook_event_name,omitempty"`
    ToolName       string          `json:"tool_name,omitempty"`
    ToolInput      json.RawMessage `json:"tool_input,omitempty"`
    ToolUseID      string          `json:"tool_use_id,omitempty"`
    TranscriptPath string          `json:"transcript_path,omitempty"`
    CWD            string          `json:"cwd,omitempty"`
    PermissionMode string          `json:"permission_mode,omitempty"`
}

// HookOutput 根据事件类型返回不同格式
type HookOutput struct {
    Decision            *string             `json:"decision,omitempty"`
    Reason              string              `json:"reason,omitempty"`
    HookSpecificOutput  *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// HookSpecificOutput 表示 PreToolUse hook 的特定输出
type HookSpecificOutput struct {
    HookEventName            string `json:"hookEventName"`
    PermissionDecision       string `json:"permissionDecision"`
    PermissionDecisionReason string `json:"permissionDecisionReason"`
}

// 向后兼容：保留旧类型
type StopHookInput = HookInput
```

### 2. 输出转换函数

**位置**: `internal/cli/hook.go`

```go
// SupervisorResultToHookOutput 将内部审查结果转换为 hook 输出
func SupervisorResultToHookOutput(result *SupervisorResult, eventType string) *HookOutput {
    if eventType == "PreToolUse" {
        decision := "allow"
        if !result.AllowStop {
            decision = "deny"
        }
        return &HookOutput{
            HookSpecificOutput: &HookSpecificOutput{
                HookEventName:            "PreToolUse",
                PermissionDecision:       decision,
                PermissionDecisionReason: result.Feedback,
            },
        }
    }

    // Stop 事件（默认）
    if !result.AllowStop {
        decision := "block"
        return &HookOutput{
            Decision: &decision,
            Reason:   result.Feedback,
        }
    }

    return &HookOutput{
        Reason: result.Feedback,
    }
}
```

### 3. 扩展输入解析

**位置**: `internal/cli/hook.go`，修改 `RunSupervisorHook` 函数

```go
// 使用新的 HookInput 结构（向后兼容）
var input HookInput
if err := decoder.Decode(&input); err != nil {
    return fmt.Errorf("failed to parse stdin JSON: %w", err)
}

// 识别事件类型
eventType := input.HookEventName
if eventType == "" {
    eventType = "Stop" // 默认为 Stop 事件
}
```

### 4. 扩展输出格式

**位置**: `internal/cli/hook.go`，修改输出逻辑

```go
// 转换结果为 hook 输出
output := SupervisorResultToHookOutput(result, eventType)

// 输出 JSON
outputJSON, err := json.MarshalIndent(output, "", "  ")
if err != nil {
    return fmt.Errorf("failed to marshal hook output: %w", err)
}
fmt.Println(string(outputJSON))
```

### 5. 添加 PreToolUse hook 配置

**位置**: `internal/provider/provider.go`，修改 `SwitchWithHook` 函数

```go
// Create hooks configuration
hooks := map[string]interface{}{
    "Stop": []map[string]interface{}{
        {
            "hooks": []map[string]interface{}{
                {
                    "type":    "command",
                    "command": hookCommand,
                    "timeout": 600,
                },
            },
        },
    },
    "PreToolUse": []map[string]interface{}{
        {
            "matcher": "AskUserQuestion",
            "hooks": []map[string]interface{}{
                {
                    "type":    "command",
                    "command": hookCommand,
                    "timeout": 600,
                },
            },
        },
    },
}
```

### 6. 迭代计数一致性

**位置**: `internal/cli/hook.go`，`RunSupervisorHook` 函数

确保所有事件类型都增加迭代计数（当前实现已在 SDK 调用前增加计数，无需修改）。

## 测试策略

### 单元测试

1. **测试输入解析**：
   - Stop 事件输入解析
   - PreToolUse 事件输入解析
   - 缺少 `hook_event_name` 字段时默认为 Stop

2. **测试输出转换**：
   - SupervisorResult → Stop 事件输出
   - SupervisorResult → PreToolUse 事件输出
   - allow_stop=true → "allow" 决策
   - allow_stop=false → "deny" 决策

3. **测试向后兼容**：
   - 旧格式输入能正确解析
   - 旧格式输出保持不变

### 集成测试

1. **测试完整 hook 流程**：
   - 模拟 PreToolUse hook 调用
   - 验证输出格式正确

2. **测试配置生成**：
   - 验证生成的 hooks 配置包含 PreToolUse

### 端到端测试

1. **手动测试**：
   - 启用 Supervisor 模式
   - 触发 AskUserQuestion 调用
   - 验证审查行为

## 复杂度跟踪

本功能无需任何宪章违规，所有实现都遵循现有原则和模式。
