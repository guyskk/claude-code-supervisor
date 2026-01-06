# Proposal: add-dynamic-reverse-proxy

## Summary

添加动态反向代理功能，允许 ccc 在本地启动一个反向代理服务器，Claude Code 通过该代理访问 API。用户可以通过 API 或 Web 界面动态切换使用的模型提供商，无需重启 Claude Code。

## Motivation

### 当前问题
1. **静态配置**: 当前 ccc 使用静态配置，切换提供商需要重新启动 Claude Code
2. **不灵活**: 无法在 Claude Code 运行时动态切换模型提供商
3. **手动配置**: 需要手动编辑 ccc.json 并重新执行 ccc 命令

### 优势
1. **动态切换**: 在 Claude Code 运行时通过 API 动态切换提供商
2. **统一入口**: 本地反向代理作为统一的 API 入口，简化配置
3. **无需重启**: 切换提供商不需要重启 Claude Code 进程
4. **可扩展**: 为未来的 Web 服务和中心服务器打下基础

## What Changes

### 新增功能
1. **反向代理服务器**
   - 用 Go 实现高性能 HTTP 反向代理
   - 支持动态目标切换
   - API Key 转发和验证

2. **配置支持**
   - 在 ccc.json 中添加 `proxy` 配置项
   - 配置代理监听地址、默认目标等
   - 支持启用/禁用代理模式

3. **管理 API**
   - `GET /providers` - 获取可用提供商列表
   - `GET /provider/current` - 获取当前提供商
   - `PUT /provider/current` - 切换当前提供商
   - `GET /health` - 健康检查

4. **Claude 集成**
   - 自动修改 settings.json 中的 ANTHROPIC_BASE_URL 指向本地代理
   - 保留原始提供商信息用于代理路由

### 配置示例

```json
{
  "proxy": {
    "enabled": true,
    "listen": "127.0.0.1:8080",
    "default_provider": "kimi",
    "api_key_header": "X-API-Key"
  }
}
```

## Impact

- **新增依赖**: HTTP 服务器库（使用 Go 标准库 net/http）
- **新增文件**: `internal/proxy/` 包
- **修改文件**:
  - `internal/config/config.go` - 添加 proxy 配置
  - `internal/provider/provider.go` - 支持代理模式
  - `internal/cli/exec.go` - 启动代理服务器

## Affected Specs

- `core-config` - 添加 proxy 配置支持
- 新增 `reverse-proxy` spec - 定义反向代理行为

## Open Questions

1. **代理生命周期**: 代理应该随 ccc 启动还是独立运行？
   - *决策*: 随 ccc 启动，作为后台进程运行

2. **端口冲突**: 如果端口被占用如何处理？
   - *决策*: 自动选择可用端口，输出实际端口到 stderr

3. **安全性**: 本地代理是否需要认证？
   - *决策*: 初期不需要，仅监听 127.0.0.1

4. **热重载**: 切换提供商时是否需要验证新提供商配置？
   - *决策*: 切换前验证 API 连接，失败则保持当前提供商
