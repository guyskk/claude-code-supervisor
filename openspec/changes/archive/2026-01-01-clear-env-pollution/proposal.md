# Change: 防止 settings.json 中的 env 配置污染提供商配置

## Why

Claude Code 启动时会按 `--setting-sources`（默认：user,project,local）顺序加载配置。其中 user 源会加载 `~/.claude/settings.json`，即使使用 `--settings` 参数指定了其他配置文件，settings.json 仍然会被合并。

这导致 ccc 切换提供商时，settings.json 中的 env 配置（如 API key、model 等）会被加载并污染提供商特定的配置，使得切换提供商后实际使用的配置与预期不符。

## What Changes

- 在切换提供商时，清空 `~/.claude/settings.json` 中的 `env` 字段
- 添加 `ClearEnvInSettings()` 辅助函数到 config 包
- 在 `provider.Switch()` 中调用清空函数，并在清空时输出提示信息
- 保持现有文件结构不变（仍使用 `settings-{provider}.json` + `--settings`）

## Impact

- Affected specs: `provider-management`, `core-config`
- Affected code:
  - `internal/config/config.go` - 新增 `ClearEnvInSettings()` 函数
  - `internal/provider/provider.go` - 修改 `Switch()` 函数调用清空逻辑
  - `internal/cli/cli.go` - 无需修改（保持 --settings 参数）
