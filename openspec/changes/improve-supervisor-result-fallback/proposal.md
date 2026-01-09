# Change: improve-supervisor-result-fallback

## Why

当前 Supervisor Mode 在解析 Supervisor 返回的 JSON 结果失败时会导致整个 hook 执行失败，进而中断 Agent 的工作流程。

这是不健壮的行为，因为：
1. LLM 可能返回不符合 JSON Schema 的内容
2. 即使使用了 `llmparser` 进行容错解析，仍然可能无法解析
3. 解析失败应该被视为"任务未完成"而不是"系统错误"

## What Changes

- **修改 `parseResultJSON` 函数** - 当 JSON 解析失败时，不返回错误
- **添加 fallback 逻辑** - 将原始 result 内容作为 feedback，设置 `allow_stop=false`
- **更新测试用例** - 覆盖解析失败的场景

## Impact

- **Affected specs**: `supervisor-hooks` - 添加新的场景要求
- **Affected code**:
  - `internal/cli/hook.go` - 修改 `parseResultJSON` 函数
  - `internal/cli/hook_test.go` - 添加测试用例
