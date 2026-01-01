## 1. 实现 ClearEnvInSettings 函数

- [x] 1.1 在 `internal/config/config.go` 中添加 `GetSettingsJSONPath()` 函数
- [x] 1.2 实现 `ClearEnvInSettings()` 函数
  - 读取 `~/.claude/settings.json`
  - 检查是否存在 `env` 字段
  - 如果存在，将 `env` 设置为空对象 `{}`
  - 写回文件
  - 返回是否清空了 env（bool）和可能的错误

## 2. 修改 provider.Switch 逻辑

- [x] 2.1 在 `provider.Switch()` 中，先写入 `settings-{provider}.json`
- [x] 2.2 然后调用 `config.ClearEnvInSettings()`
- [x] 2.3 如果清空了 env，输出提示信息："Cleared env field in settings.json to prevent configuration pollution"

## 3. 测试

- [x] 3.1 为 `ClearEnvInSettings()` 添加单元测试
- [x] 3.2 更新 `provider.Switch()` 的测试以验证清空逻辑
- [x] 3.3 运行完整测试套件确保无回归

## 4. 代码质量检查

- [x] 4.1 运行 `go test ./... -race` 验证无竞态条件
- [x] 4.2 运行 `gofmt -l .` 检查格式
- [x] 4.3 运行 `go vet ./...` 进行静态检查
