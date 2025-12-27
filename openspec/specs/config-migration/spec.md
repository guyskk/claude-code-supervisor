# config-migration Specification

## Purpose
TBD - created by archiving change add-migration-tests. Update Purpose after archive.
## Requirements
### Requirement: 自动检测旧配置

系统 SHALL 能够检测 `~/.claude/settings.json` 文件是否存在，以便决定是否提供配置迁移选项。

#### Scenario: settings.json 存在
- **WHEN** 用户运行 `ccc` 或 `ccc <provider>` 命令
- **AND** `~/.claude/ccc.json` 文件不存在
- **AND** `~/.claude/settings.json` 文件存在
- **THEN** 系统应当检测到旧配置存在，返回 true

#### Scenario: settings.json 不存在
- **WHEN** 用户运行 `ccc` 或 `ccc <provider>` 命令
- **AND** `~/.claude/ccc.json` 文件不存在
- **AND** `~/.claude/settings.json` 文件不存在
- **THEN** 系统应当检测到旧配置不存在，返回 false

### Requirement: 用户交互式迁移确认

当检测到旧配置存在时，系统 SHALL 提示用户并等待确认，而不是自动执行迁移。

#### Scenario: 用户接受迁移
- **WHEN** 系统提示 "Would you like to create ccc config from existing settings? [y/N]"
- **AND** 用户输入 "y" 或 "yes"（不区分大小写）
- **THEN** 系统应当返回 true，表示用户同意迁移

#### Scenario: 用户拒绝迁移
- **WHEN** 系统提示 "Would you like to create ccc config from existing settings? [y/N]"
- **AND** 用户输入 "n" 或 "no" 或其他任意字符
- **THEN** 系统应当返回 false，表示用户拒绝迁移
- **AND** 程序应当直接退出，不创建任何配置

#### Scenario: 输入读取失败
- **WHEN** 系统尝试读取用户输入
- **AND** 读取过程中发生错误（如 stdin 关闭）
- **THEN** 系统应当返回 false，默认拒绝迁移

### Requirement: 配置迁移算法

系统 SHALL 能够从 `settings.json` 迁移配置到 `ccc.json`，正确拆分 `env` 字段和其他配置。

#### Scenario: 标准迁移 - 包含 env 字段
- **GIVEN** settings.json 内容为:
  ```json
  {
    "permissions": { "allow": ["*"] },
    "alwaysThinkingEnabled": true,
    "env": {
      "ANTHROPIC_BASE_URL": "https://api.example.com",
      "ANTHROPIC_AUTH_TOKEN": "sk-xxx"
    }
  }
  ```
- **WHEN** 执行迁移操作
- **THEN** 应当创建 ccc.json，内容为:
  ```json
  {
    "settings": {
      "permissions": { "allow": ["*"] },
      "alwaysThinkingEnabled": true
    },
    "current_provider": "default",
    "providers": {
      "default": {
        "env": {
          "ANTHROPIC_BASE_URL": "https://api.example.com",
          "ANTHROPIC_AUTH_TOKEN": "sk-xxx"
        }
      }
    }
  }
  ```
- **AND** settings.json 应当保持不变（只读）

#### Scenario: 迁移 - 不包含 env 字段
- **GIVEN** settings.json 内容为:
  ```json
  {
    "permissions": { "allow": ["*"] },
    "alwaysThinkingEnabled": true
  }
  ```
- **WHEN** 执行迁移操作
- **THEN** 应当创建 ccc.json，内容为:
  ```json
  {
    "settings": {
      "permissions": { "allow": ["*"] },
      "alwaysThinkingEnabled": true
    },
    "current_provider": "default",
    "providers": {}
  }
  ```

#### Scenario: 空配置迁移
- **GIVEN** settings.json 内容为: `{}`
- **WHEN** 执行迁移操作
- **THEN** 应当创建 ccc.json，内容为:
  ```json
  {
    "settings": {},
    "current_provider": "default",
    "providers": {}
  }
  ```

#### Scenario: settings.json 读取失败
- **GIVEN** settings.json 文件不存在或无读取权限
- **WHEN** 执行迁移操作
- **THEN** 应当返回错误，错误信息包含 "failed to read settings file"
- **AND** 不应当创建 ccc.json 文件

#### Scenario: settings.json 格式错误
- **GIVEN** settings.json 内容不是有效的 JSON 格式
- **WHEN** 执行迁移操作
- **THEN** 应当返回错误，错误信息包含 "failed to parse settings file"
- **AND** 不应当创建 ccc.json 文件

### Requirement: 迁移流程集成

配置迁移 SHALL 无缝集成到 `ccc` 命令的启动流程中，当配置文件缺失时自动触发。

#### Scenario: 首次运行时自动迁移
- **WHEN** 用户首次运行 `ccc` 命令
- **AND** `~/.claude/ccc.json` 不存在
- **AND** `~/.claude/settings.json` 存在
- **THEN** 系统应当提示用户确认迁移
- **AND** 如果用户确认，执行迁移并继续运行
- **AND** 如果用户拒绝，退出程序并显示帮助信息

#### Scenario: 迁移成功后继续运行
- **GIVEN** 用户确认迁移
- **WHEN** 迁移成功完成
- **THEN** 系统应当显示 "Created ccc config with 'default' provider from existing settings."
- **AND** 重新加载 ccc.json 配置
- **AND** 使用迁移后的配置继续执行用户命令

#### Scenario: 迁移失败时退出
- **GIVEN** 用户确认迁移
- **WHEN** 迁移过程中发生错误
- **THEN** 系统应当输出错误信息到 stderr
- **AND** 退出程序，返回非零退出码

### Requirement: 测试覆盖

配置迁移功能 SHALL 有完善的单元测试和集成测试，覆盖所有正常场景和边缘情况。

#### Scenario: 单元测试覆盖
- **GIVEN** 配置迁移相关的所有函数
- **WHEN** 运行 `go test -cover ./...`
- **THEN** 测试覆盖率应当 ≥90%
- **AND** 所有测试应当通过

#### Scenario: 竞态条件检测
- **WHEN** 运行 `go test -race ./...`
- **THEN** 不应当检测到任何数据竞争
- **AND** 所有测试应当通过

#### Scenario: 测试隔离
- **GIVEN** 任意测试用例
- **WHEN** 测试执行
- **THEN** 不应当修改用户的真实配置文件（`~/.claude/`）
- **AND** 应当使用独立的临时目录进行测试

#### Scenario: 集成测试 - 完整迁移流程
- **GIVEN** 准备好的 settings.json 测试文件
- **WHEN** 模拟用户接受迁移并执行完整流程
- **THEN** 应当验证生成的 ccc.json 结构正确
- **AND** 应当验证 settings.json 未被修改
- **AND** 应当验证可以正常加载迁移后的配置

