# 实施任务清单

## 1. 代码重构（提高可测试性）

- [x] 1.1 添加 `getUserInputFunc` 全局变量
  - 位置: main.go 顶部（在 `getClaudeDirFunc` 之后）
  - 功能: 封装用户输入读取逻辑，便于测试时模拟
  - 预计修改: 约 10 行

- [x] 1.2 重构 `promptUserForMigration()` 函数
  - 修改 main.go:111-124
  - 使用 `getUserInputFunc` 替代直接读取 `os.Stdin`
  - 保持现有业务逻辑不变
  - 预计修改: 约 5 行

## 2. 单元测试实现

- [x] 2.1 创建 `main_test.go` 文件
  - 添加测试辅助函数（临时目录设置、配置清理）
  - 预计新增: 约 50 行

- [x] 2.2 实现 `TestCheckExistingSettings`
  - 测试场景: settings.json 存在/不存在
  - 使用 `t.TempDir()` 创建隔离测试环境
  - 预计新增: 约 40 行

- [x] 2.3 实现 `TestPromptUserForMigration`
  - 测试场景: 用户输入 y/yes/n/no/其他
  - 测试场景: 输入读取失败
  - 使用表驱动测试（table-driven tests）
  - 预计新增: 约 60 行

- [x] 2.4 实现 `TestMigrateFromSettings`
  - 测试场景: 标准迁移（包含 env）
  - 测试场景: 无 env 字段
  - 测试场景: 空配置 `{}`
  - 测试场景: settings.json 读取失败
  - 测试场景: settings.json 格式错误
  - 使用表驱动测试
  - 验证生成的 ccc.json 结构正确性
  - 验证 settings.json 未被修改
  - 预计新增: 约 150 行

## 3. 集成测试实现

- [x] 3.1 实现 `TestMigrationFlowAccept`
  - 测试完整流程: 检测 → 提示 → 接受 → 迁移 → 加载
  - 准备测试数据: 创建 settings.json
  - 模拟用户接受迁移
  - 验证 ccc.json 正确生成
  - 验证可以正常加载迁移后的配置
  - 预计新增: 约 80 行

- [x] 3.2 实现 `TestMigrationFlowReject`
  - 测试拒绝迁移流程
  - 验证不创建 ccc.json
  - 预计新增: 约 40 行

- [x] 3.3 实现 `TestMigrationFlowErrors`
  - 测试迁移失败场景（格式错误、权限问题）
  - 验证错误处理和错误消息
  - 预计新增: 约 50 行

## 4. 测试质量验证

- [x] 4.1 运行测试并检查覆盖率
  - 命令: `go test -cover ./...`
  - 验证: 迁移功能相关代码覆盖率 ≥90%
  - 结果: checkExistingSettings 100%, promptUserForMigration 100%, migrateFromSettings 95.5%

- [x] 4.2 运行竞态检测
  - 命令: `go test -race ./...`
  - 验证: 无数据竞争警告
  - 结果: 通过，无竞态条件

- [x] 4.3 运行代码格式检查
  - 命令: `gofmt -l .`
  - 验证: 无格式问题
  - 结果: 通过

- [x] 4.4 运行静态检查
  - 命令: `go vet ./...`
  - 验证: 无警告
  - 结果: 通过

## 5. 文档更新

- [x] 5.1 在代码中添加必要的注释
  - 为 `getUserInputFunc` 添加文档注释
  - 更新 `promptUserForMigration()` 的文档注释
  - 确保测试函数有清晰的注释说明

## 验收标准

完成所有任务后，满足以下条件：

- ✅ 所有测试通过: `go test ./...`
- ✅ 迁移功能覆盖率 ≥90%: checkExistingSettings 100%, promptUserForMigration 100%, migrateFromSettings 95.5%
- ✅ 无竞态条件: `go test -race ./...`
- ✅ 代码格式正确: `gofmt -l .` 无输出
- ✅ 静态检查通过: `go vet ./...` 无警告
- ✅ 现有功能不受影响

## 实际完成情况

- 代码重构: 15 行修改 ✅
- 测试代码: 约 523 行新增 ✅
- 文档注释: 已包含在代码中 ✅
- 总计: 约 538 行新增/修改 ✅

