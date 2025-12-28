# 实现任务清单

## 1. 核心验证功能实现
- [ ] 1.1 创建 `internal/validate` 包
- [ ] 1.2 添加 `ValidationResult` 和 `ValidationSummary` 类型
- [ ] 1.3 实现 `validateProvider()` 函数：验证单个提供商配置
  - 检查环境变量完整性
  - 验证 Base URL 格式
  - 测试 API 连通性
- [ ] 1.4 实现 `validateAllProviders()` 函数：验证所有提供商
  - 遍历所有提供商配置
  - 生成汇总报告
- [ ] 1.5 实现 `testAPIConnection()` 函数：测试 API 连通性
- [ ] 1.6 实现 `printValidationResult()` 函数：格式化输出验证结果
- [ ] 1.7 实现 `Run()` 函数：处理验证命令流程

## 2. CLI 集成
- [ ] 2.1 扩展 `Command` 结构体支持 validate 命令
- [ ] 2.2 添加 `ValidateCommand` 结构体
- [ ] 2.3 更新 `Parse()` 函数解析 validate 命令
- [ ] 2.4 在 `Run()` 函数中添加 validate 命令处理
- [ ] 2.5 更新 `ShowHelp()` 添加 validate 命令说明

## 3. 测试
- [ ] 3.1 添加 `validateProvider()` 函数单元测试
  - 测试有效配置
  - 测试缺失环境变量
  - 测试无效 URL 格式
  - 测试提供商不存在
- [ ] 3.2 添加 `validateAllProviders()` 函数单元测试
  - 测试多个提供商
  - 测试空配置
- [ ] 3.3 添加 `ValidationResult` 字段验证测试
- [ ] 3.4 确保 `go test -race ./...` 通过

## 4. 文档
- [ ] 4.1 更新 README.md：添加 validate 命令使用说明
- [ ] 4.2 更新 README-CN.md：添加 validate 命令使用说明

## 5. 验证
- [ ] 5.1 运行 `openspec validate add-config-validation --strict`
- [ ] 5.2 运行 `go test ./...` 确保所有测试通过
- [ ] 5.3 运行 `go vet ./...` 确保静态检查通过
- [ ] 5.4 运行 `gofmt -l .` 确保代码格式正确
