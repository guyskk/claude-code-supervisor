# error-handling Specification Delta

## Purpose

定义统一的错误处理规范，确保错误信息清晰、一致、可操作。

## ADDED Requirements

### Requirement: 错误类型分类

系统 SHALL 定义错误类型枚举用于错误分类。

#### Scenario: 错误类型定义
- **WHEN** 查询错误类型
- **THEN** 应当包含以下类型：
  - `ErrTypeConfig`: 配置错误
  - `ErrTypeNetwork`: 网络错误
  - `ErrTypeProcess`: 进程错误
  - `ErrTypeValidation`: 验证错误
  - `ErrTypeTimeout`: 超时错误

### Requirement: 应用错误结构

系统 SHALL 定义 `AppError` 结构体用于统一错误表示。

#### Scenario: AppError 结构
- **GIVEN** 创建一个 `AppError`
- **THEN** 应当包含：
  - `Type`: 错误类型（ErrorType）
  - `Code`: 错误码（string）
  - `Message`: 用户可见的错误消息
  - `Cause`: 底层错误（error，可选）
  - `Context`: 额外上下文信息（map[string]interface{}，可选）

#### Scenario: 错误格式化输出
- **GIVEN** 一个 `AppError` 包含 Code="CCC_CLAUDE_NOT_FOUND" 和 Context={"path": "/usr/bin/claude"}
- **WHEN** 调用 `Error()` 方法
- **THEN** 输出应当包含：`[CCC_CLAUDE_NOT_FOUND] path=/usr/bin/claude ...`

### Requirement: 预定义错误码

系统 SHALL 定义常用错误码常量。

#### Scenario: 配置相关错误码
- **WHEN** 查询配置错误码
- **THEN** 应当包含：
  - `CCC_CONFIG_INVALID`: 配置文件格式无效
  - `CCC_CONFIG_NOT_FOUND`: 配置文件不存在
  - `CCC_CONFIG_READ_FAILED`: 配置文件读取失败
  - `CCC_CONFIG_PARSE_FAILED`: 配置文件解析失败

#### Scenario: 提供商相关错误码
- **WHEN** 查询提供商错误码
- **THEN** 应当包含：
  - `CCC_PROVIDER_NOT_FOUND`: 指定的提供商不存在
  - `CCC_PROVIDER_INVALID`: 提供商配置无效

#### Scenario: Claude 相关错误码
- **WHEN** 查询 Claude 错误码
- **THEN** 应当包含：
  - `CCC_CLAUDE_NOT_FOUND`: claude 命令未找到
  - `CCC_CLAUDE_START_FAILED`: claude 启动失败
  - `CCC_CLAUDE_EXIT_ABNORMALLY`: claude 异常退出

#### Scenario: Supervisor 相关错误码
- **WHEN** 查询 Supervisor 错误码
- **THEN** 应当包含：
  - `CCC_SUPERVISOR_TIMEOUT`: Supervisor 调用超时
  - `CCC_SUPERVISOR_MAX_ITERATIONS`: 达到最大迭代次数
  - `CCC_SUPERVISOR_PARSE_FAILED`: Supervisor 输出解析失败

### Requirement: 错误创建和包装

系统 SHALL 提供函数创建和包装错误。

#### Scenario: 创建新错误
- **GIVEN** 需要创建一个配置错误
- **WHEN** 调用 `errors.NewError(ErrTypeConfig, "CCC_CONFIG_INVALID", "配置文件格式无效", err)`
- **THEN** 应当返回 `*AppError`
- **AND** `Type` 应当为 `ErrTypeConfig`
- **AND** `Code` 应当为 `"CCC_CONFIG_INVALID"`

#### Scenario: 包装错误
- **GIVEN** 已有一个错误 `err`
- **WHEN** 调用 `errors.Wrap(err, "CCC_CLAUDE_START_FAILED", "启动 claude 失败")`
- **THEN** 应当返回 `*AppError`
- **AND** `Cause` 应当指向原始错误 `err`
- **AND** 调用 `errors.Unwrap()` 应当能获取原始错误

#### Scenario: 添加上下文
- **GIVEN** 已有一个 `AppError`
- **WHEN** 调用 `e.WithContext("path", "/path/to/file")`
- **THEN** 应当返回新的 `AppError`
- **AND** `Context` 应当包含 `{"path": "/path/to/file"}`

### Requirement: 错误日志记录

系统 SHALL 在记录错误时包含完整的错误信息。

#### Scenario: 记录错误到日志
- **GIVEN** 一个 `AppError` 包含完整的上下文
- **WHEN** 使用 `logger.Error()` 记录
- **THEN** 日志应当包含：
  - 错误码
  - 错误消息
  - 所有上下文字段
  - 原始错误（如果存在）

#### Scenario: 用户友好的错误消息
- **GIVEN** 一个内部错误需要展示给用户
- **WHEN** 格式化错误消息
- **THEN** 应当：
  - 使用清晰的语言描述问题
  - 提供解决建议（如果可能）
  - 避免暴露技术细节

### Requirement: 错误处理最佳实践

代码 SHALL 遵循错误处理最佳实践。

#### Scenario: 错误包装
- **GIVEN** 函数调用返回错误
- **WHEN** 返回错误给调用者
- **THEN** 应当使用 `errors.Wrap()` 添加上下文
- **AND** 不应当丢弃原始错误

#### Scenario: 错误检查
- **GIVEN** 函数可能返回错误
- **WHEN** 调用该函数
- **THEN** 应当立即检查错误
- **AND** 不应当忽略错误

#### Scenario: 错误传播
- **GIVEN** 底层函数返回错误
- **WHEN** 上层函数无法处理该错误
- **THEN** 应当包装后向上传播
- **AND** 应当添加当前层级的上下文

### Requirement: 可恢复错误处理

系统 SHALL 对某些可恢复错误进行自动处理。

#### Scenario: 网络错误重试
- **GIVEN** API 调用返回网络错误
- **WHEN** 错误类型为 `ErrTypeNetwork`
- **THEN** 系统可以自动重试
- **AND** 重试次数不应超过 3 次

#### Scenario: 配置错误不重试
- **GIVEN** 配置文件解析失败
- **WHEN** 错误类型为 `ErrTypeConfig`
- **THEN** 不应当重试
- **AND** 应当立即返回错误给用户

## Examples

### 示例 1: 创建配置错误

```go
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, errors.NewError(
                errors.ErrTypeConfig,
                "CCC_CONFIG_NOT_FOUND",
                "配置文件不存在",
                err,
            ).With("path", path)
        }
        return nil, errors.Wrap(
            err,
            "CCC_CONFIG_READ_FAILED",
            "读取配置文件失败",
        ).With("path", path)
    }

    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, errors.NewError(
            errors.ErrTypeConfig,
            "CCC_CONFIG_PARSE_FAILED",
            "配置文件格式无效",
            err,
        ).With("path", path)
    }

    return &cfg, nil
}
```

### 示例 2: 使用错误

```go
cfg, err := config.Load("/path/to/ccc.json")
if err != nil {
    var appErr *errors.AppError
    if errors.As(err, &appErr) {
        switch appErr.Code {
        case "CCC_CONFIG_NOT_FOUND":
            logger.Error("配置文件不存在", "path", appErr.Context["path"])
            // 创建默认配置
        case "CCC_CONFIG_PARSE_FAILED":
            logger.Error("配置文件格式无效", "details", appErr.Message)
            // 显示配置文件示例
        default:
            logger.Error("未知错误", "code", appErr.Code)
        }
    }
    return err
}
```
