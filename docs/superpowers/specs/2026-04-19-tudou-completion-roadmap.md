# tudou 完成路线图 — 整体设计

- **日期**：2026-04-19
- **作者**：RenaLio + Claude 协作
- **状态**：Draft，待 M1 详细 plan

---

## 1. 范围与目标

### 1.1 使用场景

- **B 类场景**：个人 + 少数信任者（家人/朋友）
- 自己是唯一管理员；为朋友手动建用户 + 发 Token
- 不涉及陌生人注册、合规审计、计费

### 1.2 终极验收标准

> 你和少数信任者可以用任意 OpenAI / Claude 协议的客户端（Claude Code、Cline、Cursor、curl）稳定打进网关，走配置好的渠道组，返回正确响应，并在后台看到请求日志和用量统计。

### 1.3 In-Scope

| 能力 | 所属里程碑 |
|---|---|
| `/v1/chat/completions`、`/v1/messages`、`/v1/responses` 三协议入口 | M1 |
| SSE 流式透传 + Usage 异步回填 | M1 |
| Token 鉴权中间件（给 LLM 调用方） | M1 |
| 管理端 JWT 鉴权补齐到所有 CRUD 路由 | M1 |
| RequestLog 异步写入（底座已有） | M1 |
| 多 Provider 平台适配（OpenAI / Claude / OpenAI-compatible，复用现有 `base.Client`） | M1 |
| `FetchModel` 实装（替换硬编码桩为调 `base.Client.FetchModels`） | M1 |
| 前端请求日志页 | M1 |
| 前端：渠道分组 / 系统配置 / Token 管理 / Playground | M2 |
| 统计聚合 worker + Dashboard 接真数据 | M3 |
| Docker、密钥外置、部署文档、关键路径单测 | M4 |

### 1.4 Out-of-Scope（明确不做）

- ❌ `internal/models/temp_ignore/` 全部：**OAuth2、邀请码、操作审计日志**
- ❌ 多管理员 / 角色分层
- ❌ 限流 / 计费 / 分账
- ❌ Swagger 自动生成（手写 API 文档）
- ❌ i18n
- ❌ MySQL/PostgreSQL 性能调优

### 1.5 既有资产盘点

- **Provider 层** `pkg/provider/platforms/base/base.go` 已实现 OpenAI / Claude / OpenAI-Responses 三种格式 + 格式互转
- **LoadBalancer** `internal/loadbalancer/` 动态选 endpoint、评分插件、随机扰动完整
- **RequestLog service/repo/异步队列** 完整，缺被调用方
- **统计 Repo**（`channel_stats`、`user_usage_daily`/`hourly`、`token_stats`）存在，缺 worker
- **JWT 中间件** `RequireAuth` 存在，只挂在 `/self` 和 `/token`

---

## 2. 里程碑规划

### 2.1 依赖图

```
M1 (独立，可单独 ship)
 ├→ M2 (只依赖 M1 后端已稳定)
 ├→ M3 (可与 M2 并行)
 └→ M4 (最后做)
```

M2 和 M3 可并行或互换顺序；M1 是所有后续的前置。

### 2.2 M1 — MVP 最小可自用

**验收**：可以用 `claude` CLI 或 `curl` 以 Claude 协议打进来，拿到流式响应，`/api/v1/request-log` 能查到记录。

**后端交付**：

1. Relay 路由 `POST /v1/chat/completions`、`/v1/messages`、`/v1/responses`
2. Token 鉴权中间件 `RequireToken`
3. `RelayHandler` + `RelayService.Forward`：LB 选 endpoint → `base.Client.Execute()` 透传 → 流式 / 非流式分支 → RequestLog 异步写入
4. `ClientRegistry` 懒加载缓存 `*base.Client`
5. `RequireAuth` 补齐到 `/channel`、`/channel-group`、`/model`、`/system-config`、`/stats`
6. `FetchModel` 实装
7. `/api/v1/request-log` 查询接口 + handler

**前端交付**：

1. `api/request-log.ts`
2. `views/RequestLogsView.vue`：分页表格 + 状态过滤 + 详情抽屉

### 2.3 M2 — 前端补全

**验收**：所有现有后端功能都有对应 UI。

1. `ChannelGroupsView.vue`（CRUD + 成员渠道挂载）
2. `SystemConfigView.vue` + `api/system-config.ts`
3. `PlaygroundView.vue`（自身 /v1 流式对话测试）
4. `TokensView.vue` 补"绑定渠道组"字段
5. 路由/菜单/auth store 调整

### 2.4 M3 — 统计聚合

**验收**：Dashboard 显示真实 tokens / 成本 / 成功率。

1. `repository/aggregation_task_repo.go` + `service/aggregation_task_service.go`
2. 定时 worker：
   - 小时聚合：`request_log` → `user_usage_hourly_stats`、`channel_stats`、`token_stats`
   - 日聚合：`hourly → daily`
3. 基于 `AggregationTask` 的崩溃恢复（status / start_id / end_id）
4. 健康检查端点 `/health/aggregation`

### 2.5 M4 — 生产化

**验收**：能一键部署到 VPS，重启数据不丢，关键路径有回归测试。

1. JWT secret 从 env 读
2. `Dockerfile` + `docker-compose.yml`（app + nginx + volume）
3. 关键路径测试：Token 中间件、RelayHandler non-stream happy path、LB select、RequestLog prepare
4. 补流式 + failover 集成测试（httptest mock 上游）
5. `docs/deploy.md`、`docs/api.md`

---

## 3. M1 模块边界与数据流

### 3.1 改动清单

| 类型 | 位置 | 说明 |
|---|---|---|
| 新增中间件 | `internal/middleware/token_auth.go` | `RequireToken`：解析 Bearer → 查 Token → 校验 → 注入 `TokenID/UserID/GroupID` |
| 新增 Handler | `internal/handler/relay_handler.go` | 注册 `/v1/chat/completions`、`/v1/messages`、`/v1/responses`；挂 `RequireToken` |
| 扩展 Service | `internal/service/relay.go` | 保留 `FetchModel`（换成调 `base.Client.FetchModels`）；新增 `Forward(ctx, format, body)` |
| 新增 Registry | `internal/relay/client_registry.go` | `sync.Map[channelID]*base.Client`，channel 更新时 invalidate |
| 扩展 DTO | `api/v1/relay.go` | 三种格式的请求/响应 marker 结构 |
| 新增 RequestLog 路由 | `internal/handler/request_log_handler.go` | 管理端查询日志，挂 `RequireAuth` |
| 补齐管理端鉴权 | 5 个 handler 加 `r.Use(middleware.RequireAuth(...))` | 一行 × 5 |
| 路由注册 | `internal/router/router.go` + `internal/start/init.go` | relay 分组挂在 `/v1/*`，不走 `/api/v1` |

### 3.2 数据流

```
Client  ──POST /v1/chat/completions──▶  Gin
                                        │
                                        ▼
                           RequireToken middleware
                           • 解析 Bearer sk-xxx
                           • 查 Token + 校验 + 注入 ctx
                                        │
                                        ▼
                              RelayHandler.Forward
                           • 读 body（缓存为 []byte）→ 提取 model, stream
                           • 构造 loadbalancer.Request{GroupID, Model, Strategy}
                                        │
                                        ▼
                             LoadBalancer.Select
                           → []*Result（按分+随机扰动排序）
                                        │
                                        ▼
                      for each candidate (failover loop):
                           • ClientRegistry.Get(channel.ID) → *base.Client
                           • client.Execute(ctx, req, metricsCallback)
                                │
                    ┌───────────┴────────────┐
                    ▼                        ▼
               非流式路径                 流式路径
          resp.Body → client          io.Pipe + Flusher
                                       逐 chunk 转发
                                        │
                                        ▼
                               metricsCallback
                           • 拼 *models.RequestLog
                           • requestLogService.CreateAsync
```

### 3.3 关键设计决策

| # | 决策 | 选择 |
|---|---|---|
| D1 | relay 路由前缀 | **`/v1/*`**（OpenAI/Claude 客户端原生兼容） |
| D2 | Client 缓存 | **懒加载 sync.Map**，channel 更新/删除时显式失效 |
| D3 | Failover 触发 | **5xx / 网络错误 / 429 限流** 触发，最多 3 次；其余 4xx 直接返回 |
| D4 | Token 额度 | **M1 只记 usage 不扣减**；额度靠渠道 `price_rate` 限制 |
| D5 | Body 重读 | **读完缓存 `[]byte`**，给 model 解析也给 Provider |
| D6 | ctx 取消传播 | 客户端断开 → `c.Request.Context()` Done → 上游 HTTP 自动取消 |

### 3.4 假设（开 plan 前验证）

- Token 模型有 `GroupID` 字段（绑定渠道组）；若无需在 M1 加
- `base.Client.Execute` 流式下直接把 `types.Response.Body` 作为 reader 返回
- `MetricsCallback` 流式结束时触发一次携带完整 usage

---

## 4. 跨切面关注点

### 4.1 错误处理分类

| 类别 | 示例 | 对客户端响应 | 写 RequestLog |
|---|---|---|---|
| 客户端错误 | 400 / 401 / 404 / 422 | 原样返回 4xx | 写 |
| 上游错误 | 5xx / 网络超时 / **429 限流** | failover，全失败返 502 | 写（每次尝试一条） |
| 网关自身错误 | LB 0 endpoint、Token 过期、序列化错 | 500 + trace_id | 写 |

客户端错误**不重试**；上游错误**最多 3 次 failover**。

### 4.2 日志 / Tracing

- `X-Request-Id` 响应头回写（relay 路径也挂 RequestID 中间件）
- RequestLog 字段：`request_id / token_id / user_id / channel_id / model / upstream_model / status / latency_ms / usage.*`
- 日志级别：请求进出 → info；failover → warn；网关错误 → error；客户端错误 → debug

### 4.3 配置与环境

- 敏感值从 env 读（M4 强制）：`JWT_SECRET`、上游 API Key
- 配置文件只放默认值 + 非敏感
- 运行时可改的放 `system_config` 表

### 4.4 Context 语义

- `ctx.Request.Context()` = 客户端连接生命周期
- RequestLog 异步写入：`context.WithoutCancel(ctx)`（已实现 ✓）
- MetricsCallback 同样：`context.WithoutCancel(ctx)`
- LB Select 用原 ctx

### 4.5 并发与资源

- **HTTP Client 池**：全局单例 `*http.Client`（`pkg/httpclient/`），所有 `base.Client` 共享 Transport
- **ClientRegistry**：`sync.Map[channelID]*base.Client`
- **流式并发**：每流一 goroutine，ctx cancel 兜底，M1 不做全局限流

### 4.6 测试策略

| 里程碑 | 范围 |
|---|---|
| M1 | Token 中间件、LB select、RelayHandler 非流式 happy path |
| M2 | 前端组件级（可选） |
| M3 | 聚合 worker 幂等性 + 跨小时/跨天边界 |
| M4 | 补流式 + failover 集成测试（httptest mock 上游） |

不做：全量 e2e、前端 E2E、压测。

### 4.7 向后兼容

- `/v1/chat/completions`、`/v1/messages`、`/v1/responses` **原格式透传**
- 错误响应**伪装成上游原生格式**（OpenAI `{error:{...}}`、Claude `{type:"error",error:{...}}`），避免 SDK 崩溃

---

## 5. 后续流程

1. 本 spec 由用户 review
2. 用户确认后进入 **M1 writing-plans**，产出 `docs/superpowers/plans/2026-04-19-M1-relay-core.md`
3. M1 实施完成后再分别为 M2 / M3 / M4 开 plan
