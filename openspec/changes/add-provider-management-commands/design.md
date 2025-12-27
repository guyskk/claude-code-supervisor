# 设计文档：提供商管理子命令

## Context

当前 `ccc` 工具只支持切换提供商（`ccc <provider>`），不支持通过命令行管理提供商配置。用户必须手动编辑 `~/.claude/ccc.json` 文件来添加、删除或修改提供商。

本设计旨在添加一组子命令，使用户能够通过命令行完成所有提供商管理操作。

## Goals / Non-Goals

### Goals
- 提供完整的提供商 CRUD 操作（添加、查看、修改、删除）
- 交互式引导用户添加新提供商
- 自动验证配置有效性
- 保持简单易用的命令行界面
- 100% 向后兼容现有功能

### Non-Goals
- 不支持批量导入/导出配置（可作为未来增强）
- 不支持提供商配置的版本控制（超出当前范围）
- 不实现提供商配置的自动发现或云端同步

## Decisions

### 1. 命令结构设计

采用 **子命令模式**：`ccc provider <action> [args...]`

**理由**：
- 清晰的层次结构，符合 CLI 最佳实践（参考 `git remote`, `docker container` 等）
- 易于扩展新功能
- 避免与现有的 `ccc <provider>` 切换命令冲突

**命令列表**：
```
ccc provider list                         # 列出所有提供商
ccc provider add <name>                   # 添加新提供商（交互式）
ccc provider remove <name>                # 删除提供商
ccc provider show <name>                  # 显示提供商详细配置
ccc provider set <name> <key> <value>     # 设置环境变量
```

**Alternatives considered**：
- ❌ 使用 flag 方式（如 `ccc --add-provider <name>`）- 不易扩展，不符合现代 CLI 习惯
- ❌ 使用独立命令（如 `ccc-provider`）- 增加用户认知负担，不够直观

### 2. 交互式 vs 非交互式

**决策**：`provider add` 命令支持**可选的交互式模式**

- **交互式**（默认）：`ccc provider add kimi` - 逐步提示用户输入 BASE_URL, AUTH_TOKEN, MODEL 等
- **非交互式**：`ccc provider add kimi --base-url=... --token=... --model=...` - 一次性指定所有参数

**理由**：
- 交互式对新手友好，降低学习曲线
- 非交互式支持脚本自动化
- 两种模式互不冲突，满足不同用户需求

### 3. 配置验证策略

**必填字段验证**：
- `ANTHROPIC_BASE_URL` - 必须是有效的 HTTPS URL
- `ANTHROPIC_AUTH_TOKEN` - 非空字符串
- `ANTHROPIC_MODEL` - 非空字符串

**可选字段**：
- `ANTHROPIC_SMALL_FAST_MODEL` - 如果不提供，默认与 `ANTHROPIC_MODEL` 相同

**验证时机**：
- `add` 命令执行时立即验证
- `set` 命令修改配置项时验证单个字段
- 保存到 `ccc.json` 前再次验证完整性

### 4. 错误处理

**错误场景**：
1. 提供商名称已存在 → 提示用户使用 `set` 修改或 `remove` 后重新添加
2. 提供商不存在 → 列出可用提供商列表
3. 配置验证失败 → 显示具体哪个字段有问题
4. JSON 文件损坏 → 建议用户检查 `~/.claude/ccc.json`

**错误输出**：
- 使用 stderr 输出错误信息
- 返回非零退出码
- 提供清晰的错误描述和修复建议

### 5. 实现方式

**单文件实现**：继续在 `main.go` 中实现所有功能

**理由**：
- 当前项目是单文件架构，保持一致性
- 总代码量预计增加 200-300 行，仍在可维护范围内
- 避免过早抽象

**函数组织**：
```go
// Provider management commands
func listProviders(config *Config) error
func addProvider(config *Config, name string, interactive bool) error
func removeProvider(config *Config, name string) error
func showProvider(config *Config, name string) error
func setProviderEnv(config *Config, name, key, value string) error

// Helper functions
func validateProviderConfig(providerConfig map[string]interface{}) error
func promptForProviderConfig() (map[string]interface{}, error)
```

## Risks / Trade-offs

### Risks

1. **用户习惯变化**
   - 风险：用户可能继续手动编辑 JSON 和使用新命令混用，导致困惑
   - 缓解：在文档中明确说明两种方式都支持，但推荐使用命令行

2. **配置文件竞争**
   - 风险：多个 `ccc` 实例同时修改配置文件可能导致数据丢失
   - 缓解：当前为单用户工具，并发风险较低；未来可考虑文件锁机制

### Trade-offs

1. **简单性 vs 功能性**
   - 选择：优先简单性，第一版只实现核心 CRUD 操作
   - 放弃：高级功能如批量操作、配置模板、配置验证 API 连通性等

2. **代码复杂度**
   - 增加：约 200-300 行新代码
   - 收益：显著提升用户体验，降低使用门槛

## Migration Plan

**不需要迁移** - 本功能完全向后兼容：
- 现有配置文件格式不变
- 现有命令行用法不变
- 用户可选择性地使用新命令

## Open Questions

无 - 设计已明确，可直接开始实现。
