# 实现任务清单

## 1. 核心验证功能实现
- [ ] 1.1 添加 `validateProvider` 函数：验证单个提供商配置
  - 检查环境变量完整性
  - 验证 Base URL 格式
  - 测试 API 连通性
- [ ] 1.2 添加 `validateAllProviders` 函数：验证所有提供商
  - 遍历所有提供商配置
  - 生成汇总报告
- [ ] 1.3 添加 `runValidation` 函数：处理验证命令
  - 解析命令行参数（--all, --provider）
  - 格式化输出验证结果

## 2. 命令行集成
- [ ] 2.1 在 `main` 函数中添加 `validate` 命令处理
  - 添加 `--validate` 参数检测
  - 支持 `ccc validate` 和 `ccc validate --all` 两种模式
- [ ] 2.2 更新 `showHelp` 函数：添加 validate 命令说明

## 3. 测试
- [ ] 3.1 添加单元测试：`validateProvider` 函数测试
  - 测试有效配置
  - 测试缺失环境变量
  - 测试无效 URL 格式
- [ ] 3.2 添加单元测试：`validateAllProviders` 函数测试
  - 测试多个提供商
  - 测试空配置
- [ ] 3.3 添加集成测试：完整验证流程测试
  - 测试命令行调用
  - 测试输出格式

## 4. 文档
- [ ] 4.1 更新 README.md：添加 validate 命令使用说明
- [ ] 4.2 更新帮助信息：添加验证命令的说明和示例
