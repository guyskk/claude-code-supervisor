# Change: 添加配置验证功能

## Why

用户在使用 ccc 切换提供商时，可能会遇到配置问题：
1. API key 已过期或无效
2. Base URL 配置错误
3. 环境变量缺失或格式错误
4. 切换提供商后无法判断配置是否有效

目前用户只有在实际调用 Claude Code 时才能发现配置错误，这降低了工具的可用性和用户体验。

## What Changes

- 添加 `ccc validate` 命令：验证当前配置或指定提供商的配置
- 添加 `ccc validate --all` 命令：验证所有提供商的配置
- 配置验证包括：
  - 配置文件格式验证（JSON 语法、必需字段）
  - 环境变量完整性检查（ANTHROPIC_BASE_URL、ANTHROPIC_AUTH_TOKEN）
  - API 连通性测试（简单的健康检查请求）
  - 模型名称验证（如果配置了 ANTHROPIC_MODEL）
- 添加详细的验证报告，包括：
  - 配置状态（有效/无效）
  - 具体错误信息（如果有）
  - 建议的修复方法

## Impact

- Affected specs: 新增 `config-validation` capability
- Affected code:
  - main.go: 添加 `validateProvider`、`validateAllProviders`、`runValidation` 函数
  - 新增验证逻辑（约 100-150 行代码）
- 不影响现有功能（向后兼容）
