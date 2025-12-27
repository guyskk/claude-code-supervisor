# 实现任务清单

## 1. 核心功能实现

- [x] 1.1 实现命令行参数解析逻辑，识别 `provider` 子命令
- [x] 1.2 实现 `listProviders(config *Config)` 函数 - 列出所有提供商
- [x] 1.3 实现 `addProvider(config *Config, name string, flags map[string]string)` 函数 - 添加提供商（支持交互式和非交互式）
- [x] 1.4 实现 `removeProvider(config *Config, name string)` 函数 - 删除提供商
- [x] 1.5 实现 `showProvider(config *Config, name string)` 函数 - 显示提供商详细配置
- [x] 1.6 实现 `setProviderEnv(config *Config, name, key, value string)` 函数 - 设置环境变量

## 2. 辅助功能

- [x] 2.1 实现 `validateProviderConfig(providerConfig map[string]interface{})` - 验证提供商配置有效性
- [x] 2.2 实现 `promptForProviderConfig()` - 交互式引导用户输入配置
- [x] 2.3 实现 `maskToken(token string)` - 脱敏显示 AUTH_TOKEN
- [x] 2.4 实现 `deleteSettingsFile(providerName string)` - 删除对应的 settings-{provider}.json 文件

## 3. 输入验证

- [x] 3.1 验证 ANTHROPIC_BASE_URL 必须是有效的 HTTPS URL
- [x] 3.2 验证 ANTHROPIC_AUTH_TOKEN 非空
- [x] 3.3 验证 ANTHROPIC_MODEL 非空
- [x] 3.4 验证提供商名称格式（建议：小写字母、数字、连字符）

## 4. 错误处理

- [x] 4.1 处理提供商名称已存在的情况
- [x] 4.2 处理提供商不存在的情况
- [x] 4.3 处理删除当前提供商的情况
- [x] 4.4 处理删除最后一个提供商的情况
- [x] 4.5 处理配置文件读写失败的情况
- [x] 4.6 处理用户中断输入（Ctrl+C）的情况

## 5. 帮助信息

- [x] 5.1 实现 `showProviderHelp()` - 显示 provider 子命令总体帮助
- [x] 5.2 为每个子命令添加 `--help` 支持
- [x] 5.3 更新主 `showHelp()` 函数，添加 provider 子命令说明

## 6. 测试

- [x] 6.1 编写 `TestListProviders` - 测试列出提供商
- [x] 6.2 编写 `TestAddProvider` - 测试添加提供商（交互式和非交互式）
- [x] 6.3 编写 `TestRemoveProvider` - 测试删除提供商
- [x] 6.4 编写 `TestShowProvider` - 测试显示提供商配置
- [x] 6.5 编写 `TestSetProviderEnv` - 测试设置环境变量
- [x] 6.6 编写 `TestValidateProviderConfig` - 测试配置验证
- [x] 6.7 编写 `TestProviderErrors` - 测试各种错误场景
- [x] 6.8 确保测试覆盖率 ≥90%
- [x] 6.9 运行 `go test -race ./...` 确保无数据竞争

## 7. 文档更新

- [x] 7.1 更新 README.md - 添加 provider 子命令使用说明
- [x] 7.2 更新 README-CN.md - 添加中文使用说明
- [x] 7.3 添加使用示例到文档

## 8. 集成测试

- [x] 8.1 手动测试完整的添加→修改→删除流程
- [x] 8.2 测试与现有 `ccc <provider>` 切换功能的兼容性
- [x] 8.3 测试配置文件手动编辑和命令行管理混用的场景
- [x] 8.4 在 macOS, Linux 和 Windows 上测试（通过 CI）

## 9. CI/CD

- [x] 9.1 确保所有 CI 检查通过（lint, build, test）
- [x] 9.2 更新 GitHub Actions workflow（如有需要）
