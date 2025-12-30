# Proposal: 简化 API 验证逻辑，移除不必要的回退机制

## Why

当前 `testAPIConnection` 函数存在 `/v1/models` 端点被重复调用的问题，并且有复杂的回退逻辑：

**问题 1：重复调用**
- Provider 没有配置 model 时：
  1. 调用 `fetchAvailableModels()` 获取模型列表（**第1次 /v1/models**）
  2. 选择最佳模型
  3. 用选中的模型调用 `/v1/messages`
  4. 如果 `/v1/messages` 失败，回退调用 `testModelsEndpoint()`（**第2次 /v1/models**）

**问题 2：不必要的 /v1/messages 调用**
- 如果 `/v1/models` 成功获取了模型列表，说明 token 是有效的
- 此时再调用 `/v1/messages` 是多余的
- 对于有严格访问控制的 provider（如 "88"），`/v1/messages` 总是会失败

**问题 3：复杂的回退逻辑**
- 当 `/v1/messages` 失败时，需要判断错误类型决定是否回退
- 增加了代码复杂度和维护成本

## What Changes

**简化验证逻辑**：

1. **如果没有配置 model**：
   - 调用 `/v1/models` 获取模型列表
   - **成功 → 直接返回 "ok"**（token 有效，不需要再测试）
   - **失败 → 返回错误**

2. **如果配置了 model**：
   - 直接用配置的 model 调用 `/v1/messages`
   - **成功 → 返回 "ok"**
   - **失败 → 直接返回错误**（**移除回退机制**）

3. **移除 `testModelsEndpoint()` 函数**（不再需要回退）

4. **移除所有回退相关的代码和判断逻辑**

## Impact

- Affected code: `internal/validate/validate.go`
  - 简化 `testAPIConnection` 函数逻辑
  - 移除 `testModelsEndpoint` 函数
  - 移除回退相关的条件判断
- 代码量减少约 20 行
- API 调用减少：
  - 无 model 配置的 provider：从 2-3 次 API 调用减少到 1 次
  - 有 model 配置的 provider：保持 1 次 API 调用
- 用户体验改进：
  - 验证速度更快（减少网络请求）
  - 逻辑更清晰（没有复杂的回退）
  - 对于有严格访问控制的 provider（如 "88"），直接验证成功

## Edge Cases

1. **Provider "88"（无 model，有严格访问控制）**：
   - 调用 `/v1/models` → 成功 → 返回 "ok" ✅

2. **Provider "ikun"（无 model，正常）**：
   - 调用 `/v1/models` → 成功 → 返回 "ok" ✅

3. **Provider 有 model 配置但 token 无效**：
   - 调用 `/v1/messages` → 失败 → 返回错误 ❌
   - 用户看到具体的错误信息，可以修复配置

4. **Provider 有 model 配置且 token 有效**：
   - 调用 `/v1/messages` → 成功 → 返回 "ok" ✅
