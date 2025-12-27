# 设计文档: 配置迁移测试

## Context

当前代码已实现配置迁移功能（main.go:104-178），但缺少测试。现有实现存在以下可测试性问题：

1. **用户交互依赖 stdin**: `promptUserForMigration()` 直接读取 `os.Stdin`，难以模拟用户输入
2. **文件系统硬编码**: 直接使用 `os.ReadFile`/`os.WriteFile`，测试需要操作真实文件
3. **全局状态依赖**: 通过 `getClaudeDir()` 获取配置路径，已有 `getClaudeDirFunc` 但未在所有场景应用

好消息：
- `getClaudeDirFunc` 机制已经存在（main.go:18），测试可以覆盖它
- 大部分逻辑已经模块化（三个独立函数）

## Goals / Non-Goals

### Goals
- 添加完整的单元测试覆盖（目标 >90% 覆盖率）
- 添加集成测试验证端到端迁移流程
- 测试边缘情况（空配置、格式错误、文件权限）
- 最小化代码重构，优先利用现有 `getClaudeDirFunc` 机制

### Non-Goals
- 不重构业务逻辑（迁移算法保持不变）
- 不修改公共 API 或函数签名
- 不引入外部测试框架（仅使用 Go 标准库）

## Decisions

### 决策 1: 利用现有的 `getClaudeDirFunc` 机制

**选择**: 在测试中覆盖 `getClaudeDirFunc`，指向临时测试目录

**理由**:
- 已有机制，无需额外重构
- 测试隔离性好，不影响用户配置
- 符合项目现有模式（参考 project.md 测试策略）

**实现**:
```go
func TestMigrateFromSettings(t *testing.T) {
    // 保存原始函数
    originalFunc := getClaudeDirFunc
    defer func() { getClaudeDirFunc = originalFunc }()

    // 创建临时目录
    tmpDir := t.TempDir()
    getClaudeDirFunc = func() string { return tmpDir }

    // 测试逻辑...
}
```

### 决策 2: 提取用户输入为可注入函数

**选择**: 添加一个可覆盖的 `getUserInputFunc` 变量（类似 `getClaudeDirFunc`）

**理由**:
- 最小化修改：只需将 `promptUserForMigration()` 中的读取逻辑提取
- 保持一致性：使用与 `getClaudeDirFunc` 相同的模式
- 易于测试：测试时注入模拟输入

**重构示例**:
```go
// 添加全局变量
var getUserInputFunc = func(prompt string) (string, error) {
    fmt.Print(prompt)
    reader := bufio.NewReader(os.Stdin)
    return reader.ReadString('\n')
}

// 修改 promptUserForMigration
func promptUserForMigration() bool {
    fmt.Println("ccc configuration not found.")
    fmt.Println("Found existing Claude configuration at: " + getSettingsPath(""))

    input, err := getUserInputFunc("Would you like to create ccc config from existing settings? [y/N] ")
    if err != nil {
        return false
    }

    input = strings.TrimSpace(strings.ToLower(input))
    return input == "y" || input == "yes"
}
```

### 决策 3: 分层测试策略

**层次 1: 单元测试**
- `TestCheckExistingSettings`: 测试文件存在性检查
- `TestPromptUserForMigration`: 测试用户输入解析（模拟输入）
- `TestMigrateFromSettings`: 测试迁移逻辑核心算法
  - 正常场景：有 env 和其他配置
  - 空 env：settings.json 没有 env 字段
  - 空配置：settings.json 为 `{}`
  - 格式错误：无效 JSON

**层次 2: 集成测试**
- `TestMigrationFlow`: 测试完整迁移流程
  - 准备 settings.json
  - 模拟用户接受迁移
  - 验证生成的 ccc.json 结构
  - 验证 settings.json 未被修改

**层次 3: 边缘情况测试**
- 文件权限问题（只读目录）
- 并发调用（多个进程同时迁移）
- 超大配置文件

### 决策 4: 测试数据组织

**选择**: 使用表驱动测试（table-driven tests）

**理由**:
- Go 社区最佳实践
- 易于添加新测试用例
- 减少重复代码

**示例**:
```go
func TestMigrateFromSettings(t *testing.T) {
    tests := []struct {
        name           string
        settingsJSON   string
        wantErr        bool
        wantSettings   map[string]interface{}
        wantProviders  map[string]map[string]interface{}
    }{
        {
            name: "正常迁移-包含env",
            settingsJSON: `{"permissions": {}, "env": {"ANTHROPIC_BASE_URL": "https://..."}}`,
            wantErr: false,
            wantSettings: map[string]interface{}{"permissions": map[string]interface{}{}},
            wantProviders: map[string]map[string]interface{}{
                "default": {"env": map[string]interface{}{"ANTHROPIC_BASE_URL": "https://..."}},
            },
        },
        {
            name: "无env字段",
            settingsJSON: `{"permissions": {}}`,
            wantErr: false,
            wantSettings: map[string]interface{}{"permissions": map[string]interface{}{}},
            wantProviders: map[string]map[string]interface{}{},
        },
        {
            name: "格式错误",
            settingsJSON: `{invalid json}`,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑...
        })
    }
}
```

## Alternatives Considered

### 备选方案 1: 引入 gomock/testify

**优点**: 更强大的 mock 能力，更丰富的断言
**缺点**: 增加外部依赖，违反项目约定（project.md: 单二进制分发）
**结论**: ❌ 拒绝，使用标准库足够

### 备选方案 2: 完全重构为接口驱动

**优点**: 更高的可测试性和灵活性
**缺点**: 大规模重构，引入复杂性，违反 YAGNI 原则
**结论**: ❌ 拒绝，当前简单函数注入已满足需求

### 备选方案 3: 使用 build tags 分离测试代码

**优点**: 测试代码不会打包到发布二进制
**缺点**: Go 测试文件（`_test.go`）本就不会打包，无需额外处理
**结论**: ⚠️ 不需要，Go 默认行为已满足

## Risks / Trade-offs

### 风险 1: 测试覆盖不完整

**风险**: 可能遗漏某些边缘情况
**缓解**:
- 使用 `go test -cover` 检查覆盖率（目标 >90%）
- Code review 时审查测试用例完整性
- 优先测试用户报告的真实场景

### 风险 2: 文件系统操作的不确定性

**风险**: 文件权限、磁盘空间等环境因素可能导致测试不稳定
**缓解**:
- 使用 `t.TempDir()` 确保隔离
- 测试失败时输出详细错误信息（包括文件路径、内容）
- 在 CI 中使用竞态检测（`-race` flag）

### Trade-off: 代码复杂度 vs 可测试性

**选择**: 仅添加 `getUserInputFunc`，不引入更多抽象
**理由**: 当前代码量小（437 行），过度抽象会降低可读性

## Migration Plan

无需迁移，纯测试添加。

### 实施步骤
1. 添加 `getUserInputFunc` 变量（main.go，约 10 行修改）
2. 重构 `promptUserForMigration()` 使用 `getUserInputFunc`（5 行修改）
3. 创建 `main_test.go` 文件
4. 实现单元测试（预计 200-300 行）
5. 实现集成测试（预计 100-150 行）
6. 运行 `go test -cover` 验证覆盖率
7. 运行 `go test -race` 检查竞态条件

### 验收标准
- ✅ 所有测试通过：`go test ./...`
- ✅ 覆盖率 >90%: `go test -cover`
- ✅ 无竞态条件：`go test -race ./...`
- ✅ CI 通过（GitHub Actions）

## Open Questions

无待解决问题，所有技术决策已明确。
