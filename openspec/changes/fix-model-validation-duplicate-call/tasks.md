# Tasks: 简化 API 验证逻辑，移除回退机制

## Tasks

1. **[代码] 简化 `testAPIConnection` 函数**
   - 如果 `model` 为空：
     - 调用 `fetchAvailableModels()` 获取模型列表
     - **成功 → 直接返回 "ok"**
     - **失败 → 返回错误**
   - 如果 `model` 不为空：
     - 直接用配置的 model 调用 `/v1/messages`
     - **成功 → 返回 "ok"**
     - **失败 → 返回错误**（移除所有回退逻辑）

2. **[代码] 移除 `testModelsEndpoint` 函数**
   - 删除整个函数（不再需要回退机制）

3. **[代码] 移除回退相关的条件和逻辑**
   - 移除 `strictAccessErrors` 列表
   - 移除回退到 `testModelsEndpoint()` 的调用
   - 移除相关的错误模式匹配逻辑

4. **[测试] 更新 `TestTestAPIConnection` 测试用例**
   - 更新或删除依赖回退机制的测试
   - 添加新测试验证简化后的行为
   - 确保测试覆盖：
     - 无 model 配置 → 调用 /v1/models 成功 → 返回 "ok"
     - 无 model 配置 → 调用 /v1/models 失败 → 返回错误
     - 有 model 配置 → 调用 /v1/messages 成功 → 返回 "ok"
     - 有 model 配置 → 调用 /v1/messages 失败 → 返回错误（不回退）

5. **[测试] 移除 `TestIsAPIStatusOK` 和 `TestFormatAPIStatus` 中不需要的测试**
   - 移除对 "ok (...)" 格式的测试（不再使用）

6. **[验证] 运行完整测试套件**
   - `go test ./internal/validate/... -v`
   - `go test ./...`
   - 确保所有测试通过

7. **[验证] 手动测试真实 provider**
   - 测试 provider "88"（无 model，/v1/models 成功）
   - 测试 provider "ikun"（无 model，/v1/models 成功）
   - 测试有 model 配置的 provider
   - 确认 `/v1/models` 只被调用一次
   - 确认不再有回退行为

## Dependencies

- 任务 1-3 是代码修改，必须按顺序完成
- 任务 4-5 是测试更新，可以在代码修改完成后并行执行
- 任务 6-7 是验证，在所有修改完成后执行
