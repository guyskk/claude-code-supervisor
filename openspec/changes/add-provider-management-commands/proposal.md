# Change: 添加提供商管理子命令

## Why

当前用户如果想添加新的 API 提供商配置（如新的 AI 服务商），必须手动编辑 `~/.claude/ccc.json` 文件。这对不熟悉 JSON 格式的普通用户来说不够友好，容易出错（如JSON格式错误、字段名拼写错误等）。

通过提供一组 `ccc provider` 子命令，用户可以通过命令行直接管理提供商配置，提升用户体验，降低使用门槛。

## What Changes

添加以下提供商管理子命令：

- `ccc provider list` - 列出所有已配置的提供商及其基本信息
- `ccc provider add <name>` - 添加新的提供商（交互式引导用户输入配置）
- `ccc provider remove <name>` - 删除指定的提供商
- `ccc provider show <name>` - 显示指定提供商的详细配置
- `ccc provider set <name> <key> <value>` - 设置提供商的环境变量配置项

**用户体验改进**：
- 所有配置操作通过命令行完成，无需手动编辑 JSON
- `add` 命令提供交互式引导，自动验证必填字段
- 提供清晰的错误提示和帮助信息
- 自动保存和验证配置文件

## Impact

**影响的 specs**：
- 新增 `provider-management` capability

**影响的代码**：
- `main.go` - 添加子命令解析逻辑和provider管理函数
- `main_test.go` - 添加新功能的单元测试

**向后兼容性**：
- ✅ 完全向后兼容，不影响现有的 `ccc <provider>` 切换功能
- ✅ 不修改现有配置文件格式
- ✅ 现有用户无需任何迁移操作
