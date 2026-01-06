# Design: add-dynamic-reverse-proxy

## Context

Claude Code Config Switcher (ccc) 当前使用静态配置模式，用户切换提供商需要重新启动 Claude Code。为了实现动态切换提供商，需要在本地运行一个反向代理服务器，作为 Claude Code 和实际 API 提供商之间的中间层。

### Goals
1. 支持在 Claude Code 运行时动态切换 API 提供商
2. 提供简单的 REST API 用于管理当前提供商
3. 保持与现有配置系统的兼容性
4. 最小化性能开销

### Non-Goals
1. 不实现用户认证（仅监听 localhost）
2. 不实现负载均衡或故障转移
3. 不修改 Claude Code CLI 本身
4. 不实现代理的高可用特性

## Architecture

### 组件结构

```
┌─────────────────┐
│  Claude Code    │
│                 │
│ ANTHROPIC_      │
│ BASE_URL:       │
│ http://local    │
│ host:8080       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  CCC Proxy      │
│  (Go HTTP)      │
│                 │
│  /v1/messages  │────┐
│  /v1/models    │    │
│  Management API│    │
└────────┬────────┘    │
         │             │
         ▼             ▼
    ┌─────────────────────┐
    │ Provider Switcher   │
    │ (Current: kimi)     │
    └─────────┬───────────┘
              │
     ┌────────┼────────┐
     ▼        ▼        ▼
  kimi     glm      m2
```

### 数据流

1. **启动流程**
   ```
   ccc kimi → 读取配置 → 启动代理 → 修改settings.json指向代理 → 启动claude
   ```

2. **API 请求流程**
   ```
   Claude Code → Proxy (localhost:8080) → 实际 Provider API
   ```

3. **切换提供商流程**
   ```
   API PUT /provider/current → 验证新提供商 → 更新内部状态 → 后续请求转发到新提供商
   ```

## Decisions

### 1. 代理服务器实现

**决策**: 使用 Go 标准库 `net/http` 实现反向代理

**理由**:
- Go 标准库性能优秀，无外部依赖
- `httputil.ReverseProxy` 提供开箱即用的反向代理功能
- 符合项目的单二进制分发原则

**实现要点**:
```go
type ProxyServer struct {
    config      *ProxyConfig
    providers   *ProviderRegistry
    current     atomic.Value // string (provider name)
    httpServer  *http.Server
    reverseProxy *httputil.ReverseProxy
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // API routes
    if strings.HasPrefix(r.URL.Path, "/api/") {
        p.handleAPI(w, r)
        return
    }
    // Proxy to current provider
    p.reverseProxy.ServeHTTP(w, r)
}
```

### 2. 当前提供商状态管理

**决策**: 使用 `atomic.Value` 存储当前提供商名称

**理由**:
- 线程安全，无需额外锁
- 支持高并发读取
- 简单高效

### 3. Claude Code 配置修改

**决策**: 代理启动时自动修改 settings.json

**方式**:
1. 读取合并后的 settings
2. 将 `ANTHROPIC_BASE_URL` 改为 `http://127.0.0.1:8080`
3. 保存原始提供商到代理配置中
4. 写入 settings.json

### 4. API Key 转发

**决策**: 代理从请求中提取 API Key 并转发到目标提供商

**实现**:
- 从环境变量或配置中读取提供商的 API Key
- 转发请求时添加 `Authorization: Bearer {key}` 头
- 不暴露原始 API Key 给客户端

## API Specification

### GET /api/providers
获取所有可用提供商列表

**Response**:
```json
{
  "providers": [
    {
      "name": "kimi",
      "base_url": "https://api.moonshot.cn/anthropic",
      "model": "kimi-k2-thinking"
    }
  ]
}
```

### GET /api/provider/current
获取当前提供商

**Response**:
```json
{
  "name": "kimi",
  "base_url": "https://api.moonshot.cn/anthropic"
}
```

### PUT /api/provider/current
切换提供商

**Request**:
```json
{
  "name": "glm"
}
```

**Response**:
```json
{
  "success": true,
  "name": "glm",
  "base_url": "https://open.bigmodel.cn/api/anthropic"
}
```

**Error Response** (400):
```json
{
  "success": false,
  "error": "Provider 'xxx' not found"
}
```

### GET /api/health
健康检查

**Response**:
```json
{
  "status": "ok",
  "current_provider": "kimi"
}
```

## Risks / Trade-offs

### 性能开销
- **风险**: 增加一跳网络延迟
- **缓解**: 本地代理延迟 < 1ms，可忽略不计

### 端口占用
- **风险**: 默认端口被占用
- **缓解**: 支持配置端口，自动选择可用端口

### 兼容性
- **风险**: Claude Code API 调用可能有特殊行为
- **缓解**: 完整转发请求头和响应，保持透明代理

## Migration Plan

### 阶段 1: 核心代理功能
- 实现基本反向代理
- 支持动态切换提供商
- 添加管理 API

### 阶段 2: 配置集成
- 添加 proxy 配置到 ccc.json
- 自动修改 settings.json
- ccc 命令行集成

### 阶段 3: 高级功能
- WebSocket 支持（如果 Claude Code 使用）
- 请求/响应日志
- 指标统计

## Open Questions

1. **代理端口冲突处理**: 如果端口被占用是否自动选择其他端口？
   - *倾向*: 是，输出实际端口到 stderr

2. **Claude Code 重启时行为**: 代理是否需要持久化状态？
   - *倾向*: 不需要，每次 ccc 启动都是新会话

3. **多个 Claude 实例**: 是否支持同时运行多个代理？
   - *倾向*: 不支持，一个端口一个实例
