# provider-management Spec Delta

## ADDED Requirements

### Requirement: 列出所有提供商

系统 SHALL 支持 `ccc provider list` 命令，列出所有已配置的提供商及其基本信息。

#### Scenario: 列出提供商（有多个）
- **GIVEN** ccc.json 配置了 3 个提供商：kimi, glm, m2
- **AND** 当前提供商是 kimi
- **WHEN** 用户执行 `ccc provider list`
- **THEN** 输出应包含所有提供商名称
- **AND** 当前提供商应有明确标记（如 `*` 或 `(current)`）
- **AND** 显示每个提供商的 BASE_URL 和 MODEL 信息
- **AND** 退出码为 0

#### Scenario: 列出提供商（无配置）
- **GIVEN** ccc.json 的 providers 为空对象
- **WHEN** 用户执行 `ccc provider list`
- **THEN** 输出提示信息 "No providers configured"
- **AND** 退出码为 0

#### Scenario: 配置文件不存在
- **GIVEN** ccc.json 文件不存在
- **WHEN** 用户执行 `ccc provider list`
- **THEN** 输出错误信息到 stderr
- **AND** 提示用户运行 `ccc --help` 或首次配置
- **AND** 退出码为非零

### Requirement: 添加新提供商（交互式）

系统 SHALL 支持 `ccc provider add <name>` 命令，通过交互式引导用户添加新的提供商配置。

#### Scenario: 成功添加提供商
- **GIVEN** ccc.json 存在且有效
- **AND** 提供商名称 "openai" 不存在
- **WHEN** 用户执行 `ccc provider add openai`
- **AND** 按提示输入 BASE_URL: "https://api.openai.com/v1"
- **AND** 输入 AUTH_TOKEN: "sk-xxx"
- **AND** 输入 MODEL: "gpt-4"
- **AND** 输入 SMALL_FAST_MODEL: "gpt-3.5-turbo"（或留空）
- **THEN** 新提供商应被添加到 ccc.json 的 providers 中
- **AND** 输出成功消息 "Provider 'openai' added successfully"
- **AND** 退出码为 0

#### Scenario: 提供商名称已存在
- **GIVEN** ccc.json 已配置提供商 "kimi"
- **WHEN** 用户执行 `ccc provider add kimi`
- **THEN** 输出错误信息 "Provider 'kimi' already exists"
- **AND** 提示使用 `ccc provider set` 修改配置或 `ccc provider remove` 删除后重新添加
- **AND** 退出码为非零
- **AND** 配置文件不应被修改

#### Scenario: 输入验证失败 - 无效 URL
- **WHEN** 用户添加提供商时输入的 BASE_URL 不是有效的 HTTPS URL（如 "http://api.test.com" 或 "invalid"）
- **THEN** 提示错误 "BASE_URL must be a valid HTTPS URL"
- **AND** 要求用户重新输入
- **AND** 最多允许 3 次重试

#### Scenario: 输入验证失败 - 空值
- **WHEN** 用户对必填字段（BASE_URL, AUTH_TOKEN, MODEL）输入空值
- **THEN** 提示错误 "This field is required"
- **AND** 要求用户重新输入

#### Scenario: 用户中断输入
- **WHEN** 用户在交互过程中按 Ctrl+C 中断
- **THEN** 输出 "Operation cancelled"
- **AND** 配置文件不应被修改
- **AND** 退出码为非零

### Requirement: 添加新提供商（非交互式）

系统 SHALL 支持通过命令行参数一次性指定提供商配置，适用于脚本自动化场景。

#### Scenario: 非交互式添加成功
- **WHEN** 用户执行 `ccc provider add openai --base-url=https://api.openai.com/v1 --token=sk-xxx --model=gpt-4`
- **THEN** 新提供商应被添加到配置
- **AND** SMALL_FAST_MODEL 应默认使用与 MODEL 相同的值
- **AND** 输出成功消息
- **AND** 退出码为 0

#### Scenario: 非交互式参数不完整
- **WHEN** 用户执行 `ccc provider add openai --base-url=https://api.openai.com/v1`（缺少 token 和 model）
- **THEN** 输出错误 "Missing required flags: --token, --model"
- **AND** 提示使用 `ccc provider add openai --help` 查看用法
- **AND** 退出码为非零

### Requirement: 删除提供商

系统 SHALL 支持 `ccc provider remove <name>` 命令，删除指定的提供商配置。

#### Scenario: 成功删除提供商
- **GIVEN** ccc.json 配置了提供商 "kimi"
- **AND** 当前提供商不是 "kimi"
- **WHEN** 用户执行 `ccc provider remove kimi`
- **THEN** 提供商 "kimi" 应从 ccc.json 的 providers 中移除
- **AND** 对应的 settings-kimi.json 文件应被删除（如果存在）
- **AND** 输出成功消息 "Provider 'kimi' removed successfully"
- **AND** 退出码为 0

#### Scenario: 删除当前正在使用的提供商
- **GIVEN** ccc.json 的 current_provider 是 "kimi"
- **WHEN** 用户执行 `ccc provider remove kimi`
- **THEN** 输出警告 "Cannot remove the current provider 'kimi'"
- **AND** 提示用户先切换到其他提供商
- **AND** 提供商不应被删除
- **AND** 退出码为非零

#### Scenario: 删除不存在的提供商
- **WHEN** 用户执行 `ccc provider remove nonexistent`
- **THEN** 输出错误 "Provider 'nonexistent' not found"
- **AND** 列出所有可用提供商
- **AND** 退出码为非零

#### Scenario: 删除最后一个提供商
- **GIVEN** ccc.json 只配置了一个提供商 "kimi"
- **WHEN** 用户执行 `ccc provider remove kimi`
- **THEN** 输出警告 "Cannot remove the last provider"
- **AND** 提示至少保留一个提供商
- **AND** 退出码为非零

### Requirement: 显示提供商详细配置

系统 SHALL 支持 `ccc provider show <name>` 命令，显示指定提供商的详细配置信息。

#### Scenario: 显示提供商配置
- **GIVEN** ccc.json 配置了提供商 "kimi"，包含 BASE_URL, AUTH_TOKEN, MODEL 等字段
- **WHEN** 用户执行 `ccc provider show kimi`
- **THEN** 输出应包含提供商名称
- **AND** 显示所有 env 配置项（格式化为易读的键值对）
- **AND** AUTH_TOKEN 应脱敏显示（如 "sk-***xxx" 只显示前3位和后3位）
- **AND** 退出码为 0

#### Scenario: 显示不存在的提供商
- **WHEN** 用户执行 `ccc provider show nonexistent`
- **THEN** 输出错误 "Provider 'nonexistent' not found"
- **AND** 列出所有可用提供商
- **AND** 退出码为非零

### Requirement: 设置提供商环境变量

系统 SHALL 支持 `ccc provider set <name> <key> <value>` 命令，修改指定提供商的环境变量配置。

#### Scenario: 成功设置环境变量
- **GIVEN** ccc.json 配置了提供商 "kimi"
- **WHEN** 用户执行 `ccc provider set kimi ANTHROPIC_MODEL kimi-k1.5`
- **THEN** 提供商 "kimi" 的 env.ANTHROPIC_MODEL 应更新为 "kimi-k1.5"
- **AND** 输出成功消息 "Provider 'kimi' updated: ANTHROPIC_MODEL=kimi-k1.5"
- **AND** 对应的 settings-kimi.json 应自动重新生成（如果当前提供商是 kimi）
- **AND** 退出码为 0

#### Scenario: 设置新的环境变量
- **GIVEN** 提供商 "kimi" 的 env 中不存在 "CUSTOM_VARIABLE"
- **WHEN** 用户执行 `ccc provider set kimi CUSTOM_VARIABLE value123`
- **THEN** 新的键值对应被添加到 env 中
- **AND** 输出成功消息
- **AND** 退出码为 0

#### Scenario: 设置不存在的提供商
- **WHEN** 用户执行 `ccc provider set nonexistent ANTHROPIC_MODEL test`
- **THEN** 输出错误 "Provider 'nonexistent' not found"
- **AND** 退出码为非零

#### Scenario: 验证关键字段
- **WHEN** 用户执行 `ccc provider set kimi ANTHROPIC_BASE_URL http://insecure.com`（非 HTTPS）
- **THEN** 输出错误 "ANTHROPIC_BASE_URL must be a valid HTTPS URL"
- **AND** 配置不应被修改
- **AND** 退出码为非零

### Requirement: 命令帮助信息

系统 SHALL 为所有 provider 子命令提供清晰的帮助信息。

#### Scenario: 显示 provider 子命令帮助
- **WHEN** 用户执行 `ccc provider` 或 `ccc provider --help`
- **THEN** 输出应包含所有可用子命令列表
- **AND** 每个子命令应有简短描述
- **AND** 包含使用示例
- **AND** 退出码为 0

#### Scenario: 显示特定子命令帮助
- **WHEN** 用户执行 `ccc provider add --help`
- **THEN** 输出应包含 `add` 命令的详细用法
- **AND** 列出所有可用的 flags（--base-url, --token, --model 等）
- **AND** 包含示例
- **AND** 退出码为 0

#### Scenario: 无效的子命令
- **WHEN** 用户执行 `ccc provider invalid-command`
- **THEN** 输出错误 "Unknown command: invalid-command"
- **AND** 提示使用 `ccc provider --help` 查看可用命令
- **AND** 退出码为非零
