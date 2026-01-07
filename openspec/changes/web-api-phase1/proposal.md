# Proposal: web-api-phase1

## 概述

分阶段构建 Web API 能力，从本地 API 开始，逐步演进到分布式架构。每个阶段独立交付价值，根据用户反馈决定是否继续。

## 设计原则

**关键**: 保持模块化，让每个阶段都能独立交付价值。

```
阶段 1: 本地 API        →  单机管理
         ↓ (如果有需求)
阶段 2: 远程 API        → 远程管理单机
         ↓ (如果有需求)
阶段 3: NAT 穿透        → 管理内网设备
         ↓ (如果有需求)
阶段 4: 中心服务器      → 多设备管理
         ↓ (如果有需求)
阶段 5: Web UI          → 完整产品
```

## Phase 1: 本地 Web API (单机管理)

### 目标

在本地启动 HTTP 服务器，通过 API 控制 ccc，支持本地脚本自动化。

### 价值

1. **脚本集成**: Shell/Python 脚本可以通过 API 切换提供商
2. **IDE 集成**: 编辑器插件可以通过 API 调用 ccc
3. **自动化测试**: CI/CD 可以通过 API 验证配置

### 实现

```go
// internal/web/local_server.go
type LocalServer struct {
    addr   string
    config *Config
}

// 启动本地服务器
func StartLocalServer(addr string) error {
    server := &LocalServer{
        addr:   addr,
        config: config.Load(),
    }

    router := gin.Default()

    // API 端点
    router.GET("/api/v1/providers", server.listProviders)
    router.GET("/api/v1/providers/:name", server.getProvider)
    router.POST("/api/v1/providers/:name/switch", server.switchProvider)
    router.GET("/api/v1/status", server.getStatus)

    return router.Run(addr)
}
```

### API 示例

```bash
# 启动服务器
ccc serve --listen-addr=:8080

# 列出提供商
curl http://localhost:8080/api/v1/providers

# 切换提供商
curl -X POST http://localhost:8080/api/v1/providers/kimi/switch

# 获取状态
curl http://localhost:8080/api/v1/status
```

### 交付物

- `ccc serve` 命令
- REST API 端点
- 基础文档

### 成功指标

- 用户可以通过 API 切换提供商
- 文档清晰，示例可运行
- 有用户实际使用并提供反馈

---

## Phase 2: 远程 API (可选，基于反馈)

### 目标

支持远程访问，通过网络管理单机上的 ccc。

### 价值

1. **远程控制**: 在其他机器上控制本机 ccc
2. **团队协作**: 多人共享一台开发机
3. **CI/CD 集成**: CI 服务器可以远程触发构建

### 实现

```go
// 添加认证和 TLS
type RemoteServerConfig struct {
    Addr     string
    AuthToken string
    TLS       bool
    CertFile  string
    KeyFile   string
}

// 中间件：简单 Token 认证
func authMiddleware(token string) gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.GetHeader("Authorization") != "Bearer " + token {
            c.AbortWithStatus(401)
            return
        }
        c.Next()
    }
}
```

### API 示例

```bash
# 启动远程服务器
ccc serve --listen-addr=0.0.0.0:8080 --auth-token=secret

# 远程访问
curl -H "Authorization: Bearer secret" \
     http://192.168.1.100:8080/api/v1/providers/kimi/switch
```

### 安全考虑

1. **Token 认证**: 简单 Bearer Token
2. **TLS 支持**: 可选 HTTPS
3. **IP 白名单**: 可配置允许的 IP

### 成功指标

- 有用户反馈需要远程访问
- 安全机制经过验证
- 有实际使用场景

---

## Phase 3: NAT 穿透 (可选，基于反馈)

### 目标

使用内网穿透技术，让中心服务器能访问内网中的 worker 设备。

### 架构选择

**推荐方案**: frp (Fast Reverse Proxy)

```
┌─────────────────────────────────────────────────┐
│           中心服务器 (公网 VPS)                    │
│  ┌──────────────┐                                │
│  │ frps (服务端)  │                                │
│  └───────┬──────┘                                │
│          │                                         │
│          │ 内网穿透                               │
│          ▼                                         │
│     Internet                                       │
└─────────────────────────────────────────────────┘
                          │
                          │ (内网穿透连接)
                          ▼
┌─────────────────────────────────────────────────┐
│           内网 Worker 设备                         │
│  ┌──────────────┐                                │
│  │ frpc (客户端)  │                                │
│  └───────┬──────┘                                │
│          │                                         │
│          ▼                                         │
│  ┌──────────────┐                                │
│  │  ccc serve   │  ← 暴露为 localhost:8080        │
│  └──────────────┘                                │
└─────────────────────────────────────────────────┘
```

### 实现方式

#### 3.1 Worker 端集成 frpc

```go
// internal/web/frp.go
type FRPClient struct {
    serverAddr string
    token      string
    localPort  int
}

// 启动时自动启动 frpc
func StartWithFRP(config *FRPConfig) error {
    // 1. 启动 ccc serve
    go StartLocalServer(":8080")

    // 2. 启动 frpc 连接中心服务器
    frpCmd := exec.Command("frpc",
        "-s", config.ServerAddr,
        "-t", config.Token,
        "-P", "worker1",  // 代理名称
        "-p", "localhost:8080",  // 本地端口
    )

    return frpCmd.Run()
}
```

#### 3.2 中心服务器集成 frps

```bash
# 中心服务器启动脚本
#!/bin/bash
# 启动 frps
frps -c /etc/frp/frps.toml

# 同时启动 API 网关
ccc-server --listen-addr=:8080
```

### 配置示例

```toml
# frps.toml (中心服务器)
bindAddr = "0.0.0.0"
bindPort = 7000

[auth]
token = "your-secret-token"

# Worker 配置在启动时动态生成
```

### 交付物

- Worker 端 frpc 集成
- 中心服务器端 frps 配置
- 连接管理文档

### 成功指标

- 内网设备可以注册到中心服务器
- 中心服务器可以调用 Worker API
- 有实际内网穿透需求

---

## Phase 4: 中心服务器 (可选，基于反馈)

### 目标

构建轻量级中心服务器，管理多个 Worker 设备，支持多设备协作。

### 核心功能

```
┌─────────────────────────────────────────────────┐
│              中心服务器 API                       │
│  ┌────────────────────────────────────────┐     │
│  │ 设备管理                                │     │
│  │  - 设备注册/上线                       │     │
│  │  - 心跳检测                             │     │
│  │  - 状态查询                             │     │
│  └────────────────────────────────────────┘     │
│  ┌────────────────────────────────────────┐     │
│  │ 任务分发                                │     │
│  │  - 向 Worker 发送任务                  │     │
│  │  - 收集结果                             │     │
│  │  - 任务队列                             │     │
│  └────────────────────────────────────────┘     │
└─────────────────────────────────────────────────┘
            │                    │
            │ 通过 frpc           │ 通过 frpc
            ▼                    ▼
┌───────────────┐      ┌───────────────┐
│ Worker A      │      │ Worker B      │
│ (内网设备)     │      │ (云端设备)     │
└───────────────┘      └───────────────┘
```

### 数据库选择

**推荐**: SQLite (单机) → PostgreSQL (生产)

```go
// 数据库抽象
type Storage interface {
    RegisterDevice(device *Device) error
    UpdateHeartbeat(deviceID string) error
    GetDevice(deviceID string) (*Device, error)
    ListDevices() ([]*Device, error)
}
```

### API 设计

```
POST   /api/v1/devices/register        设备注册
GET    /api/v1/devices                   列出设备
GET    /api/v1/devices/:id              设备详情
POST   /api/v1/devices/:id/heartbeat    心跳
POST   /api/v1/devices/:id/tasks         发送任务
GET    /api/v1/devices/:id/tasks/:id     任务状态
```

### 成功指标

- 有多个设备需要管理
- 用户反馈需要设备管理功能
- 中心服务器稳定运行

---

## Phase 5: Web UI (可选，基于反馈)

### 目标

基于现有 API 构建 Web UI，复用开源项目代码。

### 复用策略

**推荐**: Fork `opcode` 或 `Manus` 进行定制

```
项目选择：

opcode:
✅ 基础 UI 完整
✅ 使用 React + TypeScript
✅ 有完整的聊天界面
❌ 但需要大量定制

Manus:
✅ 产品成熟
✅ 功能完整
❌ 闭源，无法复用代码

策略：
1. Fork opcode 作为基础
2. 定制 API 调用层
3. 添加 CCC 特有功能
4. 贡献改进回上游
```

### 最小 UI 功能

```
Phase 5.1: 基础界面
- 提供商列表
- 提供商切换
- 配置验证

Phase 5.2: 设备管理
- 设备列表
- 任务管理
- 状态监控

Phase 5.3: 聊天界面 (如果需要)
- 类似 Manus 的对话界面
- 集成流式输出
- 工具调用可视化
```

### 成功指标

- 有用户反馈需要图形界面
- UI 比命令行有明显优势
- 有用户实际使用

---

## 实施决策树

```
是否需要本地 API？
├─ 否 → 维持现状
└─ 是 → 实施 Phase 1
        │
        ├─ 用户是否需要远程访问？
        │   ├─ 否 → 停止在 Phase 1
        │   └─ 是 → 实施 Phase 2
        │         │
        │         ├─ 是否有内网设备？
        │         │   ├─ 否 → 停止在 Phase 2
        │         │   └─ 是 → 实施 Phase 3
        │         │         │
        │         │         ├─ 是否有多个设备？
        │         │         │   ├─ 否 → 停止在 Phase 3
        │         │         │   └─ 是 → 实施 Phase 4
        │         │         │         │
        │         │         │         └─ 是否需要图形界面？
        │         │         │           ├─ 否 → 停止在 Phase 4
        │         │         │           └─ 是 → 实施 Phase 5
```

## 风险管理

### 阶段性风险

| 阶段 | 风险 | 缓解 |
|------|------|------|
| Phase 1 | 无用户使用 | 发布后收集反馈再决定 |
| Phase 2 | 安全问题 | 强调可选功能，默认关闭 |
| Phase 3 | frp 稳定性 | 使用成熟的 frp 项目 |
| Phase 4 | 复杂度高 | 充分测试，保持简单 |
| Phase 5 | 开发成本大 | Fork 现有项目，避免从零开始 |

### 停止准则

任何阶段如果：
1. 没有用户需求
2. 维护成本超过收益
3. 出现更好的替代方案

**果断停止**，聚焦核心价值。

## 与原方案对比

| 方面 | 原提案 | 新方案 |
|------|--------|--------|
| 架构 | 一次性设计完整 | 分阶段演进 |
| 复杂度 | 高 | 从低到高 |
| 价值交付 | 最后才能用 | 每阶段独立价值 |
| 风险 | 高 | 低（可随时停止） |
| 维护成本 | 高 | 渐进式投入 |

## 开放问题

1. **frp 替代方案**: 是否考虑其他 NAT 穿透方案？
   - ZeroTier：虚拟网络，更强大但也更复杂
   - ngrok：商业方案，有免费额度
   - 自实现：基于 WebSocket 的隧道

2. **认证方案**: Phase 2 使用什么认证？
   - 简单 Token：初期够用
   - JWT：更标准，但复杂度增加
   - OAuth：最完整，但过度设计

3. **数据存储**: Phase 4 使用什么数据库？
   - SQLite：零配置，适合单机
   - PostgreSQL：生产级，适合多用户
   - 建议：先用 SQLite，有需求再迁移
