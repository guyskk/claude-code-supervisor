# 重构设计文档

## 上下文

ccc 项目当前是一个 467 行的单文件应用。虽然简单，但随着功能增加，已出现可维护性和可测试性问题。需要在保持单二进制分发和简单性的前提下，改进项目结构。

## 目标 / 非目标

### 目标
- 提高代码可维护性：清晰的模块边界和单一职责
- 提高可测试性：所有核心功能可独立测试
- 提高类型安全：用强类型替代 `map[string]interface{}`
- 保持向后兼容：用户接口和配置格式不变
- 保持单二进制：所有代码静态链接到一个可执行文件

### 非目标
- 不引入外部依赖（仅使用标准库）
- 不改变 CLI 接口
- 不改变配置文件格式
- 不过度设计（避免不必要的抽象）

## 决策

### 1. 项目结构

```
.
├── main.go                 # 入口点，仅负责启动
├── config/
│   └── config.go          # 配置类型定义和加载/保存
├── migration/
│   └── migration.go       # 旧配置迁移逻辑
├── provider/
│   └── provider.go        # 提供商切换和配置合并
└── cli/
    └── cli.go             # 命令行解析和执行
```

**理由**：
- 每个包职责单一，易于理解和维护
- 包之间依赖单向：main → cli → provider/config/migration
- 测试可以针对每个包独立进行

**备选方案**：
- 使用 `internal/` 目录：考虑到项目很小，额外层级不必要
- 保持单文件：当前已 467 行，继续增长会难以维护

### 2. 类型安全配置

```go
// Env 环境变量配置
type Env map[string]string

// ProviderConfig 提供商配置
type ProviderConfig struct {
    Env Env `json:"env,omitempty"`
}

// Settings Claude 设置
type Settings struct {
    Permissions         *Permissions         `json:"permissions,omitempty"`
    AlwaysThinkingEnabled bool               `json:"alwaysThinkingEnabled,omitempty"`
    Env                  Env                 `json:"env,omitempty"`
}

// Config ccc 配置文件
type Config struct {
    Settings        Settings                 `json:"settings"`
    CurrentProvider string                   `json:"current_provider"`
    Providers       map[string]ProviderConfig `json:"providers"`
}
```

**理由**：
- 编译时类型检查，减少运行时错误
- IDE 自动完成和重构支持
- JSON 反序列化时自动验证类型

**备选方案**：
- 继续使用 `map[string]interface{}`：类型不安全，容易出错
- 使用第三方库如 mapstructure：增加依赖，不必要

### 3. 错误处理

定义错误类型：

```go
// Error 类型
type Error struct {
    Op  string // 操作名称
    Err error  // 底层错误
}

func (e *Error) Error() string {
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error {
    return e.Err
}
```

**理由**：
- 统一的错误格式
- 保留错误链用于调试
- 可以添加操作上下文

### 4. 测试策略

- **单元测试**：每个包的独立函数测试
- **集成测试**：完整流程测试
- **测试辅助**：共享的测试工具函数

```go
// internal/testutil/testutil.go
package testutil

func TempDir(t *testing.T) string
func WriteConfig(t *testing.T, dir string, cfg *config.Config)
func ReadConfig(t *testing.T, path string) *config.Config
```

## 风险 / 权衡

| 风险 | 缓解措施 |
|------|----------|
| 重构引入 bug | 保持现有测试，增量重构 |
| 接口不兼容 | 严格的兼容性测试 |
| 过度设计 | 遵循 YAGNI 原则，按需添加 |

## 迁移计划

### 阶段 1：创建新的包结构（不破坏现有代码）
1. 创建 `config` 包，定义类型
2. 创建 `migration` 包，迁移现有迁移逻辑
3. 创建 `provider` 包，迁移提供商逻辑
4. 创建 `cli` 包，迁移命令行逻辑

### 阶段 2：更新 main.go
1. 替换为调用新包的接口
2. 保持所有现有功能

### 阶段 3：测试和验证
1. 运行所有现有测试
2. 添加新的测试覆盖
3. 手动测试所有功能

### 回滚计划
- 使用 git，每个阶段独立 commit
- 如果出现问题，可以随时回滚到上一个可工作状态

## Open Questions

无 - 重构范围清晰，不需要额外的设计决策。
