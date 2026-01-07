# Proposal: integrate-claude-sdk

## 概述

集成 `schlunsen/claude-agent-sdk-go` 作为 CCC 的 Claude SDK 基础，替代当前的自定义实现，获得生产级的 SDK 功能和持续维护。

## 动机

### 当前问题

1. **维护负担**: 自定义实现需要持续维护 Claude API 变更
2. **功能缺失**: 缺少流式处理、钩子系统、权限控制等高级功能
3. **测试覆盖不足**: 自定义实现的测试覆盖有限
4. **重复造轮子**: 社区已有成熟的实现

### 社区方案分析

通过调研两个开源库：
- **connerohnesorge/claude-agent-sdk-go**: 功能简单，6 stars，维护较少
- **schlunsen/claude-agent-sdk-go**: 功能完整，9 stars，活跃维护，生产就绪

**选择**: `schlunsen/claude-agent-sdk-go`
- 零外部依赖（纯标准库）
- 完整的功能集（钩子、权限、MCP 支持）
- 优秀的文档和测试
- 活跃的开发和维护

## 变更内容

### 1. 集成方式

采用 **Fork + 定制** 策略：

```go
// Fork 仓库地址：github.com/guyskk/ccc-claude-sdk
// 基于：github.com/schlunsen/claude-agent-sdk-go

// ccc 定制版本特点：
// 1. 简化配置 - 针对 CLI 工具优化
// 2. 提供商切换 - 特定的 API 端点处理
// 3. 进程管理 - 集成到 ccc 的启动流程
```

### 2. 核心集成点

#### 2.1 替换 `internal/claude_agent_sdk`

当前自定义实现替换为基于 fork 的 SDK：

```go
// internal/claude_sdk/wrapper.go
package claude_sdk

import (
    "context"
    sdk "github.com/guyskk/ccc-claude-sdk"
)

// CCCClient 封装 SDK 为 CCC 优化的接口
type CCCClient struct {
    client *sdk.Client
    provider *ProviderConfig
}

// NewCCCClient 创建配置了特定提供商的客户端
func NewCCCClient(provider *ProviderConfig) (*CCCClient, error) {
    opts := sdk.NewClaudeAgentOptions().
        WithModel(provider.Model).
        WithBaseURL(provider.BaseURL)

    client, err := sdk.NewClient(context.Background(), opts)
    if err != nil {
        return nil, err
    }

    return &CCCClient{
        client: client,
        provider: provider,
    }, nil
}

// Query 执行一次性查询
func (c *CCCClient) Query(ctx context.Context, prompt string) (string, error) {
    messages, err := sdk.Query(ctx, prompt, sdk.QueryOptions{
        Model: c.provider.Model,
        BaseURL: c.provider.BaseURL,
    })
    if err != nil {
        return "", err
    }

    return extractContent(messages), nil
}
```

#### 2.2 保留 Provider 切换能力

SDK 作为底层，CCC 的 provider 切换逻辑在上层：

```go
// internal/provider/switch.go
func SwitchWithSDK(cfg *Config, providerName string) error {
    provider := cfg.Providers[providerName]

    // 创建配置了提供商的 SDK 客户端
    client, err := claude_sdk.NewCCCClient(provider)
    if err != nil {
        return err
    }

    // 使用 SDK 客户端执行
    result, err := client.Query(context.Background(), prompt)
    ...
}
```

#### 2.3 利用 SDK 高级功能

**钩子系统** - 用于 Supervisor Mode：

```go
// internal/supervisor/hooks.go
import sdk "github.com/guyskk/ccc-claude-sdk"

func setupSupervisorHooks(client *sdk.Client) {
    client.WithHook(sdk.HookEventPreToolUse, func(event sdk.HookEvent) {
        log.Debug("Tool about to be used", "tool", event.ToolName)
    })

    client.WithHook(sdk.HookEventPostToolUse, func(event sdk.HookEvent) {
        log.Debug("Tool completed", "tool", event.ToolName, "result", event.Result)
    });
}
```

**权限控制** - 工具使用权限管理：

```go
// internal/permissions/manager.go
func createPermissionCallback(allowedTools []string) sdk.CanUseToolFunc {
    return func(toolName string) bool {
        for _, allowed := range allowedTools {
            if allowed == toolName || allowed == "*" {
                return true
            }
        }
        return false
    }
}
```

### 3. 模块化设计

```
internal/
├── claude_sdk/           # SDK 封装层
│   ├── wrapper.go       # CCCClient
│   ├── provider.go      # 提供商配置
│   └── permissions.go   # 权限管理
├── cli/                 # CLI 入口
│   ├── exec.go          # 使用 SDK 执行
│   └── hook.go          # Supervisor hooks
└── supervisor/          # Supervisor 逻辑
    └── supervisor.go    # 使用 SDK 钩子
```

### 4. 依赖管理

**go.mod 更新**：

```go
module github.com/guyskk/ccc

go 1.21

require (
    github.com/guyskk/ccc-claude-sdk v0.1.0  // Fork 定制版本
)
```

**Fork 策略**：
1. Fork `schlunsen/claude-agent-sdk-go` 为 `guyskk/ccc-claude-sdk`
2. 针对 CCC 需求定制：
   - 移除不需要的功能（减少 binary 大小）
   - 添加提供商特定的优化
   - 集成 CCC 的日志和错误处理
3. 定期同步上游更新

### 5. 兼容性保证

**向后兼容**：
- 现有的 provider 切换功能保持不变
- CLI 接口保持不变
- Supervisor Mode 继续工作

**迁移路径**：
1. 零影响：新的 SDK 封装层与现有代码并行
2. 渐进式：逐步迁移到新 SDK
3. 可选回退：保留原有实现作为备份

## 实施计划

### Phase 1: Fork 和定制 (1-2 周)
1. Fork `schlunsen/claude-agent-sdk-go`
2. 创建 `internal/claude_sdk` 封装层
3. 编写单元测试
4. 验证基本功能

### Phase 2: 集成到 CLI (1-2 周)
1. 更新 `internal/cli/exec.go` 使用新 SDK
2. 更新 `internal/cli/hook.go` 使用 SDK 钩子
3. 端到端测试
4. 性能基准测试

### Phase 3: 高级功能 (1-2 周)
1. 集成权限控制
2. 利用流式处理
3. MCP 服务器支持（可选）
4. 文档更新

## 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| Fork 维护负担 | 中 | 与上游保持良好关系，贡献改进 |
| API 不兼容 | 低 | 封装层隔离，保持向后兼容 |
| 性能回退 | 低 | 基准测试对比 |
| License 问题 | 低 | 确认上游许可证兼容 |

## 开放问题

1. **Fork 维护**: 如何平衡上游同步和本地定制？
   - 建议：定期同步（每季度），贡献改进给上游

2. **功能裁剪**: 哪些功能可以移除以减少 binary 大小？
   - 建议：移除 MCP 服务器功能（初期不需要）
   - 移除图片处理（初期不需要）

3. **版本管理**: 如何管理 fork 版本与上游版本的关系？
   - 建议：使用语义化版本，记录上游 commit hash
