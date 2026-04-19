# M1 — Relay Core 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 tudou 能通过 OpenAI/Claude 协议（含 SSE 流式）把客户端请求转发到上游渠道，并写入请求日志，管理端能查询日志。

**Architecture:** 在现有 Gin 路由上挂 `/v1/*` relay 路由；用新增 `RequireToken` 中间件解析 Token → 注入 ctx；`RelayService.Forward` 调 `LoadBalancer.Select` 选 endpoint → 用缓存的 `*base.Client`（`ClientRegistry`）执行 → 流式通过 `StandardStream.Recv()` 透传给客户端 → 通过 `MetricsCallback` 回填 `RequestLog` 异步写入。管理端所有路由补 `RequireAuth`；启动时把 channel/group 从 DB 灌进 `loadbalancer.Registry`，CRUD 时同步。

**Tech Stack:** Go 1.26、Gin、GORM、Wire（DI，需重新生成）、zap、goccy/go-json；前端 Vue 3 + Pinia + TanStack Query + Reka UI + UnoCSS。

---

## 上下文与前置约束

### 关键既有资产（不要重写）

- `pkg/provider/platforms/base/base.go::Client.Execute(ctx, *types.Request, MetricsCallback)` — 已支持三种格式 + 格式转换 + 流式
- `internal/service/request_log_service.go::CreateAsync` — 已实现异步队列落库，用 `context.WithoutCancel`
- `internal/loadbalancer/loadbalancer.go::DynamicLoadBalancer.Select` — 已支持 group→channel→model 筛选、策略排序、随机扰动
- `internal/loadbalancer/registry.go::Registry.ReloadChannel / ReloadGroup` — 已定义，**当前未被调用**（本 plan 补上）
- `internal/service/token_service.go::TokenService.GetAvailableByToken` — 可直接用于 Token 鉴权
- `internal/middleware/auth.go::RequireAuth` — 已实现 JWT 校验
- `internal/middleware/trace.go::RequestID` — 已实现 X-Request-Id
- `pkg/httpclient/httpclient.go::GetDefaultClient` — 全局共享 HTTP 连接池

### 约定

- 所有新 Go 文件包含 package 声明，与目录名一致
- 所有测试文件名 `_test.go`，package 以 `_test` 后缀（外部测试包）以避免循环依赖
- 所有 commit 使用中文 commit message，遵循现有风格（`feat/fix/refactor(<scope>): <说明>`）
- 所有 commit 带 `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>` 尾部
- 每个 Task 结束都要 `go build ./...` 和（若有测试）`go test ./...` 全绿后才 commit

### 验证命令速查

- `go build ./...` — 全工程编译
- `go test ./internal/... -run 'Test<name>' -v` — 跑单个测试
- `go test ./... -short` — 跑所有短测试
- `cd cmd/server/wire && go run -mod=mod github.com/google/wire/cmd/wire` — 重新生成 wire_gen.go

---

## Task 1 — Token 鉴权中间件

**Files:**
- Create: `internal/middleware/token_auth.go`
- Create: `internal/middleware/token_auth_test.go`
- Modify: `internal/constants/constants.go`（若缺 `TokenIdKey/GroupIdKey` 常量则补）

- [ ] **Step 1: 读 `internal/constants/` 下所有文件，确认有没有 `TokenIdKey`、`GroupIdKey`、`UserIdKey`。若某个缺失，补到 `constants.go`**

若需要新增，在 `internal/constants/constants.go` 追加：

```go
func TokenIdKey() string { return "token_id" }
func GroupIdKey() string { return "group_id" }
```

（`UserIdKey` 和 `ClaimsKey` 已有，不要覆盖。）

- [ ] **Step 2: 写失败测试**

创建 `internal/middleware/token_auth_test.go`：

```go
package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/middleware"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type fakeTokenLookup struct {
	token *v1.TokenWithRelationsResponse
	err   error
}

func (f *fakeTokenLookup) GetAvailableByToken(_ context.Context, _ string) (*v1.TokenWithRelationsResponse, error) {
	return f.token, f.err
}

func newTokenResp(id, userID, groupID int64, status models.TokenStatus) *v1.TokenWithRelationsResponse {
	return &v1.TokenWithRelationsResponse{
		TokenResponse: v1.TokenResponse{
			ID:     id,
			UserID: userID,
			GroupID: groupID,
			Status: status,
		},
	}
}

func runMiddleware(t *testing.T, lookup middleware.TokenLookup, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var capturedTokenID, capturedUserID, capturedGroupID int64

	r := gin.New()
	r.Use(middleware.RequireToken(lookup))
	r.GET("/ping", func(c *gin.Context) {
		if v, ok := c.Get(constants.TokenIdKey()); ok {
			if id, ok := v.(int64); ok {
				capturedTokenID = id
			}
		}
		if v, ok := c.Get(constants.UserIdKey()); ok {
			if id, ok := v.(int64); ok {
				capturedUserID = id
			}
		}
		if v, ok := c.Get(constants.GroupIdKey()); ok {
			if id, ok := v.(int64); ok {
				capturedGroupID = id
			}
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	t.Logf("captured: tokenID=%d userID=%d groupID=%d", capturedTokenID, capturedUserID, capturedGroupID)
	return w
}

func TestRequireToken_MissingHeader(t *testing.T) {
	w := runMiddleware(t, &fakeTokenLookup{}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_WrongScheme(t *testing.T) {
	w := runMiddleware(t, &fakeTokenLookup{}, "Basic abc")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_LookupNotFound(t *testing.T) {
	lookup := &fakeTokenLookup{err: gorm.ErrRecordNotFound}
	w := runMiddleware(t, lookup, "Bearer sk-does-not-exist")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_LookupError(t *testing.T) {
	lookup := &fakeTokenLookup{err: errors.New("db boom")}
	w := runMiddleware(t, lookup, "Bearer sk-abc")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestRequireToken_Disabled(t *testing.T) {
	lookup := &fakeTokenLookup{token: newTokenResp(1, 2, 3, models.TokenStatusDisabled)}
	w := runMiddleware(t, lookup, "Bearer sk-disabled")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_Success(t *testing.T) {
	lookup := &fakeTokenLookup{token: newTokenResp(100, 200, 300, models.TokenStatusEnabled)}
	w := runMiddleware(t, lookup, "Bearer sk-ok")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

```
go test ./internal/middleware/ -run TestRequireToken -v
```

Expected: 编译失败或所有子测试 FAIL，因为 `middleware.RequireToken` 和 `middleware.TokenLookup` 尚未定义。

- [ ] **Step 4: 实现中间件**

创建 `internal/middleware/token_auth.go`：

```go
package middleware

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TokenLookup 抽象出 Token 查询能力，便于测试注入。
// 生产实现由 service.TokenService 提供。
type TokenLookup interface {
	GetAvailableByToken(ctx context.Context, token string) (*v1.TokenWithRelationsResponse, error)
}

const bearerPrefix = "Bearer "

// RequireToken 解析 Authorization: Bearer <token>，查 Token，注入 ctx。
// 失败时直接 abort；成功时向 ctx 注入 token_id / user_id / group_id。
func RequireToken(lookup TokenLookup) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := strings.TrimSpace(ctx.GetHeader("Authorization"))
		if authHeader == "" {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("missing Authorization header"), nil)
			ctx.Abort()
			return
		}
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("Authorization must use Bearer scheme"), nil)
			ctx.Abort()
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
		if tokenStr == "" {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("empty token"), nil)
			ctx.Abort()
			return
		}

		token, err := lookup.GetAvailableByToken(ctx.Request.Context(), tokenStr)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("token not found or not available"), nil)
				ctx.Abort()
				return
			}
			v1.Fail(ctx, v1.ErrInternalServerError.WithMessage(err.Error()), nil)
			ctx.Abort()
			return
		}
		if token == nil {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("token not available"), nil)
			ctx.Abort()
			return
		}
		if token.Status != models.TokenStatusEnabled {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("token is "+string(token.Status)), nil)
			ctx.Abort()
			return
		}

		ctx.Set(constants.TokenIdKey(), token.ID)
		ctx.Set(constants.UserIdKey(), token.UserID)
		ctx.Set(constants.GroupIdKey(), token.GroupID)

		reqCtx := context.WithValue(ctx.Request.Context(), constants.TokenIdKey(), token.ID)
		reqCtx = context.WithValue(reqCtx, constants.UserIdKey(), token.UserID)
		reqCtx = context.WithValue(reqCtx, constants.GroupIdKey(), token.GroupID)
		ctx.Request = ctx.Request.WithContext(reqCtx)

		ctx.Next()
	}
}
```

- [ ] **Step 5: 运行测试确认通过**

```
go test ./internal/middleware/ -run TestRequireToken -v
```

Expected: 全绿。

- [ ] **Step 6: 确认全工程编译**

```
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/middleware/token_auth.go internal/middleware/token_auth_test.go internal/constants/
git commit -m "$(cat <<'EOF'
feat(middleware): 新增 Token 鉴权中间件 RequireToken

解析 Authorization: Bearer <token>，通过 TokenLookup 查询可用 Token，
将 token_id / user_id / group_id 注入 gin ctx 与 request context。
配套 6 个单元测试覆盖缺 header、wrong scheme、not found、db error、disabled、success。

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2 — Client Registry（缓存 `*base.Client`）

**Files:**
- Create: `internal/relay/client_registry.go`
- Create: `internal/relay/client_registry_test.go`

- [ ] **Step 1: 写失败测试**

创建 `internal/relay/client_registry_test.go`：

```go
package relay_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/relay"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func newChannel(id int64, updated time.Time) *models.Channel {
	return &models.Channel{
		ID:         id,
		Name:       "test",
		BaseURL:    "https://api.example.com",
		APIKey:     "sk-test",
		Type:       models.ChannelTypeOpenAI,
		UpdatedAt:  updated,
	}
}

func TestClientRegistry_GetCreatesAndCaches(t *testing.T) {
	reg := relay.NewClientRegistry(&http.Client{Timeout: 1 * time.Second})

	ch := newChannel(1, time.Now())
	c1 := reg.Get(ch, []types.Ability{types.AbilityChat})
	if c1 == nil {
		t.Fatal("expected non-nil client")
	}
	c2 := reg.Get(ch, []types.Ability{types.AbilityChat})
	if c1 != c2 {
		t.Fatal("expected cached client to be reused")
	}
}

func TestClientRegistry_InvalidateOnUpdatedAtChange(t *testing.T) {
	reg := relay.NewClientRegistry(&http.Client{Timeout: 1 * time.Second})

	t1 := time.Now()
	t2 := t1.Add(1 * time.Second)

	ch1 := newChannel(1, t1)
	ch2 := newChannel(1, t2)
	ch2.APIKey = "sk-rotated"

	c1 := reg.Get(ch1, []types.Ability{types.AbilityChat})
	c2 := reg.Get(ch2, []types.Ability{types.AbilityChat})
	if c1 == c2 {
		t.Fatal("expected a new client after UpdatedAt change")
	}
}

func TestClientRegistry_InvalidateExplicit(t *testing.T) {
	reg := relay.NewClientRegistry(&http.Client{Timeout: 1 * time.Second})

	ch := newChannel(1, time.Now())
	c1 := reg.Get(ch, []types.Ability{types.AbilityChat})
	reg.Invalidate(1)
	c2 := reg.Get(ch, []types.Ability{types.AbilityChat})
	if c1 == c2 {
		t.Fatal("expected a new client after explicit invalidate")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```
go test ./internal/relay/ -v
```

Expected: 编译失败，`relay` 包和 `ClientRegistry` 不存在。

- [ ] **Step 3: 实现 ClientRegistry**

创建 `internal/relay/client_registry.go`：

```go
package relay

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

// ClientRegistry 按 channel.ID 缓存 base.Client，感知 channel.UpdatedAt 变化自动失效。
type ClientRegistry struct {
	httpC *http.Client
	mu    sync.RWMutex
	m     map[int64]*cachedClient
}

type cachedClient struct {
	client    *base.Client
	updatedAt time.Time
}

// NewClientRegistry 构造 Registry；httpC 必须非 nil。
func NewClientRegistry(httpC *http.Client) *ClientRegistry {
	return &ClientRegistry{
		httpC: httpC,
		m:     make(map[int64]*cachedClient),
	}
}

// Get 获取或创建对应 channel 的 Client；当 channel.UpdatedAt 比缓存新时强制重建。
func (r *ClientRegistry) Get(ch *models.Channel, abilities []types.Ability) *base.Client {
	if ch == nil {
		return nil
	}
	r.mu.RLock()
	cached := r.m[ch.ID]
	r.mu.RUnlock()
	if cached != nil && !ch.UpdatedAt.After(cached.updatedAt) {
		return cached.client
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// double check
	if cached, ok := r.m[ch.ID]; ok && !ch.UpdatedAt.After(cached.updatedAt) {
		return cached.client
	}
	id := strconv.FormatInt(ch.ID, 10)
	client := base.NewClient(r.httpC, ch.BaseURL, ch.APIKey, id, abilities)
	r.m[ch.ID] = &cachedClient{client: client, updatedAt: ch.UpdatedAt}
	return client
}

// Invalidate 显式失效指定 channel 的缓存。
func (r *ClientRegistry) Invalidate(channelID int64) {
	r.mu.Lock()
	delete(r.m, channelID)
	r.mu.Unlock()
}
```

- [ ] **Step 4: 运行测试确认通过**

```
go test ./internal/relay/ -v
```

Expected: 全绿。

- [ ] **Step 5: Commit**

```bash
git add internal/relay/
git commit -m "$(cat <<'EOF'
feat(relay): 新增 ClientRegistry 缓存 base.Client

按 channel.ID 缓存 *base.Client，共享全局 HTTP 连接池；
当 channel.UpdatedAt 比缓存更新时自动失效重建；
支持显式 Invalidate 以便 channel 删除/更新时手动触发。
配套 3 个单元测试。

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3 — Relay V1 DTO

**Files:**
- Modify: `api/v1/relay.go`

- [ ] **Step 1: 扩展 relay.go**

将 `api/v1/relay.go` 替换为：

```go
package v1

import (
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

// FetchModelRequest 后台"测试渠道，拉上游模型列表"的请求体。保留现有用途。
type FetchModelRequest struct {
	Type    models.ChannelType `json:"type" binding:"required"`
	BaseURL string             `json:"baseURL" binding:"required"`
	APIKey  string             `json:"apiKey" binding:"required"`
}

// RelayFormatOf 根据路径返回对应 provider Format；路径未命中返回空串。
func RelayFormatOf(path string) types.Format {
	switch path {
	case "/v1/chat/completions":
		return types.FormatChatCompletion
	case "/v1/messages":
		return types.FormatClaudeMessages
	case "/v1/responses":
		return types.FormatOpenAIResponses
	default:
		return ""
	}
}
```

- [ ] **Step 2: 编译确认**

```
go build ./...
```

Expected: PASS。

- [ ] **Step 3: Commit**

```bash
git add api/v1/relay.go
git commit -m "$(cat <<'EOF'
feat(api): 扩展 relay DTO，新增 RelayFormatOf 路径映射

保留 FetchModelRequest（后台渠道测试用途），新增
RelayFormatOf 把 HTTP 路径映射到 provider Format 常量。

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4 — RelayService.Forward（非流式路径 + Failover）

**Files:**
- Modify: `internal/service/relay.go`
- Create: `internal/service/relay_test.go`

- [ ] **Step 1: 写失败测试**

创建 `internal/service/relay_test.go`：

```go
package service_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

type fakeLB struct {
	results []*loadbalancer.Result
	err     error
}

func (f *fakeLB) Select(_ context.Context, _ *loadbalancer.Request, _ ...loadbalancer.ScorePlugin) ([]*loadbalancer.Result, error) {
	return f.results, f.err
}

type fakeCollector struct{}

func (f *fakeCollector) CollectMetrics(_ context.Context, _ *loadbalancer.ResultRecord) error {
	return nil
}

func (f *fakeCollector) IncConn(_ int64) {}

type fakeProvider struct {
	calls    int
	respSeq  []*types.Response
	errSeq   []error
	lastReq  *types.Request
}

func (f *fakeProvider) Execute(_ context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {
	f.lastReq = req
	idx := f.calls
	f.calls++
	if cb != nil {
		cb(&types.ResponseMetrics{Provider: "fake", Model: req.Model})
	}
	var resp *types.Response
	if idx < len(f.respSeq) {
		resp = f.respSeq[idx]
	}
	var err error
	if idx < len(f.errSeq) {
		err = f.errSeq[idx]
	}
	return resp, err
}

type fakeRequestLog struct {
	logs []*models.RequestLog
}

func (f *fakeRequestLog) CreateAsync(_ context.Context, log *models.RequestLog) error {
	f.logs = append(f.logs, log)
	return nil
}

func newResult(id int64, baseURL, key string) *loadbalancer.Result {
	return &loadbalancer.Result{
		UpstreamModel: "upstream-model",
		Channel: &models.Channel{
			ID:      id,
			Name:    "ch",
			BaseURL: baseURL,
			APIKey:  key,
		},
	}
}

func TestRelay_Forward_NoCandidate(t *testing.T) {
	lb := &fakeLB{err: loadbalancer.ErrNoAvailableChannel}
	s := service.NewRelayServiceForTest(lb, &fakeCollector{}, &fakeRequestLog{}, nil)

	meta := service.RelayMeta{Format: types.FormatChatCompletion, TokenID: 1, UserID: 2, GroupID: 3}
	_, err := s.Forward(context.Background(), meta, []byte(`{"model":"gpt-4o","stream":false}`), http.Header{})
	if !errors.Is(err, loadbalancer.ErrNoAvailableChannel) {
		t.Fatalf("expected ErrNoAvailableChannel, got %v", err)
	}
}

func TestRelay_Forward_NonStream_Success(t *testing.T) {
	lb := &fakeLB{results: []*loadbalancer.Result{newResult(1, "https://a", "k1")}}
	prov := &fakeProvider{
		respSeq: []*types.Response{
			{StatusCode: 200, Format: types.FormatChatCompletion, RawData: []byte(`{"ok":true}`)},
		},
	}
	logSvc := &fakeRequestLog{}
	s := service.NewRelayServiceForTest(lb, &fakeCollector{}, logSvc, func(*models.Channel) service.Provider { return prov })

	meta := service.RelayMeta{Format: types.FormatChatCompletion, TokenID: 1, UserID: 2, GroupID: 3}
	resp, err := s.Forward(context.Background(), meta, []byte(`{"model":"gpt-4o","stream":false}`), http.Header{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if prov.calls != 1 {
		t.Fatalf("expected 1 call, got %d", prov.calls)
	}
	if len(logSvc.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logSvc.logs))
	}
	got, _ := io.ReadAll(bytes.NewReader(resp.RawData))
	if string(got) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", string(got))
	}
}

func TestRelay_Forward_NonStream_FailoverOn5xx(t *testing.T) {
	lb := &fakeLB{results: []*loadbalancer.Result{
		newResult(1, "https://a", "k1"),
		newResult(2, "https://b", "k2"),
	}}
	prov := &fakeProvider{
		respSeq: []*types.Response{
			{StatusCode: 503, RawData: []byte(`upstream down`)},
			{StatusCode: 200, RawData: []byte(`{"ok":true}`)},
		},
	}
	logSvc := &fakeRequestLog{}
	s := service.NewRelayServiceForTest(lb, &fakeCollector{}, logSvc, func(*models.Channel) service.Provider { return prov })

	meta := service.RelayMeta{Format: types.FormatChatCompletion, TokenID: 1, UserID: 2, GroupID: 3}
	resp, err := s.Forward(context.Background(), meta, []byte(`{"model":"gpt-4o","stream":false}`), http.Header{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 on failover, got %d", resp.StatusCode)
	}
	if prov.calls != 2 {
		t.Fatalf("expected 2 calls after failover, got %d", prov.calls)
	}
	if len(logSvc.logs) != 2 {
		t.Fatalf("expected 2 logs (one fail, one success), got %d", len(logSvc.logs))
	}
}

func TestRelay_Forward_NonStream_No4xxFailover(t *testing.T) {
	lb := &fakeLB{results: []*loadbalancer.Result{
		newResult(1, "https://a", "k1"),
		newResult(2, "https://b", "k2"),
	}}
	prov := &fakeProvider{
		respSeq: []*types.Response{
			{StatusCode: 400, RawData: []byte(`{"error":"bad"}`)},
		},
	}
	logSvc := &fakeRequestLog{}
	s := service.NewRelayServiceForTest(lb, &fakeCollector{}, logSvc, func(*models.Channel) service.Provider { return prov })

	meta := service.RelayMeta{Format: types.FormatChatCompletion, TokenID: 1, UserID: 2, GroupID: 3}
	resp, _ := s.Forward(context.Background(), meta, []byte(`{"model":"gpt-4o","stream":false}`), http.Header{})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 passed through, got %d", resp.StatusCode)
	}
	if prov.calls != 1 {
		t.Fatalf("expected 1 call (no retry on 4xx), got %d", prov.calls)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```
go test ./internal/service/ -run 'TestRelay_' -v
```

Expected: 编译失败（需要的类型/函数都不存在）。

- [ ] **Step 3: 重写 `internal/service/relay.go`**

覆盖现有文件（保留 `FetchModel` 的接口签名以便 channel_handler 继续编译）：

```go
package service

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/relay"
	"github.com/RenaLio/tudou/pkg/httpclient"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/tidwall/gjson"
)

const maxFailoverAttempts = 3

// RelayMeta 由中间件/Handler 填好后传给 Service。
type RelayMeta struct {
	Format   types.Format
	TokenID  int64
	UserID   int64
	GroupID  int64
	Strategy models.LoadBalanceStrategy
	IP       string
	UA       string
	Path     string
}

// Provider 是 RelayService 执行一次调用所需的最小能力集。
type Provider interface {
	Execute(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error)
}

// RequestLogCreator 是 RelayService 写日志的最小接口（对应 service.RequestLogService.CreateAsync）。
type RequestLogCreator interface {
	CreateAsync(ctx context.Context, log *models.RequestLog) error
}

type RelayService interface {
	FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error)
	Forward(ctx context.Context, meta RelayMeta, body []byte, headers http.Header) (*types.Response, error)
}

type RelayServiceImpl struct {
	lb          loadbalancer.LoadBalancer
	collector   loadbalancer.MetricsCollector
	requestLogs RequestLogCreator
	registry    *relay.ClientRegistry
	// providerFor 允许测试注入假的 Provider 工厂；生产环境由 registry 产出。
	providerFor func(*models.Channel) Provider
	*Service
}

func NewRelayService(
	s *Service,
	lb loadbalancer.LoadBalancer,
	collector loadbalancer.MetricsCollector,
	requestLogs RequestLogService,
) RelayService {
	registry := relay.NewClientRegistry(httpclient.GetDefaultClient())
	impl := &RelayServiceImpl{
		lb:          lb,
		collector:   collector,
		requestLogs: requestLogs,
		registry:    registry,
		Service:     s,
	}
	impl.providerFor = func(ch *models.Channel) Provider {
		return registry.Get(ch, abilitiesFor(ch))
	}
	return impl
}

// NewRelayServiceForTest 只用于测试，允许注入假 lb / collector / logger / provider 工厂。
func NewRelayServiceForTest(
	lb loadbalancer.LoadBalancer,
	collector loadbalancer.MetricsCollector,
	requestLogs RequestLogCreator,
	providerFor func(*models.Channel) Provider,
) *RelayServiceImpl {
	return &RelayServiceImpl{
		lb:          lb,
		collector:   collector,
		requestLogs: requestLogs,
		providerFor: providerFor,
	}
}

func (s *RelayServiceImpl) FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error) {
	httpC := httpclient.GetDefaultClient()
	client := base.NewClient(httpC, strings.TrimRight(req.BaseURL, "/"), req.APIKey, "fetch-model", []types.Ability{types.AbilityChat})
	return client.Models()
}

func (s *RelayServiceImpl) Forward(ctx context.Context, meta RelayMeta, body []byte, headers http.Header) (*types.Response, error) {
	model, isStream := parseModelAndStream(body)
	if model == "" {
		return nil, errors.New("missing model in request body")
	}

	strategy := string(meta.Strategy)
	if strategy == "" {
		strategy = string(models.LoadBalanceStrategyPerformance)
	}
	candidates, err := s.lb.Select(ctx, &loadbalancer.Request{
		GroupID:  meta.GroupID,
		UserID:   meta.UserID,
		Model:    model,
		Strategy: strategy,
	})
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, loadbalancer.ErrNoAvailableChannel
	}

	limit := maxFailoverAttempts
	if len(candidates) < limit {
		limit = len(candidates)
	}
	var lastResp *types.Response
	var lastErr error

	for i := 0; i < limit; i++ {
		r := candidates[i]
		if r == nil || r.Channel == nil {
			continue
		}
		attemptStart := time.Now()

		provider := s.providerFor(r.Channel)
		if provider == nil {
			lastErr = errors.New("no provider for channel")
			continue
		}

		if s.collector != nil {
			s.collector.IncConn(r.Channel.ID)
		}

		var capturedMetrics *types.ResponseMetrics
		cb := func(m *types.ResponseMetrics) { capturedMetrics = m }

		providerReq := &types.Request{
			Model:    r.UpstreamModel,
			IsStream: isStream,
			Payload:  bytes.Clone(body),
			Format:   meta.Format,
			Headers:  cloneHeader(headers),
		}
		resp, err := provider.Execute(context.WithoutCancel(ctx), providerReq, cb)
		lastResp = resp
		lastErr = err

		s.writeRequestLog(ctx, meta, r, resp, err, capturedMetrics, attemptStart, model, isStream, body, headers)
		s.collectLBMetrics(ctx, r, resp, err, attemptStart)

		if !shouldFailover(resp, err) {
			return resp, err
		}
	}
	return lastResp, lastErr
}

func parseModelAndStream(body []byte) (string, bool) {
	model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	isStream := gjson.GetBytes(body, "stream").Bool()
	return model, isStream
}

// shouldFailover: 返回 true 表示"当前尝试失败且可以 failover"。
func shouldFailover(resp *types.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return true
	}
	switch {
	case resp.StatusCode >= 500:
		return true
	case resp.StatusCode == http.StatusTooManyRequests:
		return true
	}
	return false
}

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return http.Header{}
	}
	return h.Clone()
}

func (s *RelayServiceImpl) writeRequestLog(
	ctx context.Context,
	meta RelayMeta,
	r *loadbalancer.Result,
	resp *types.Response,
	execErr error,
	metrics *types.ResponseMetrics,
	start time.Time,
	model string,
	isStream bool,
	body []byte,
	headers http.Header,
) {
	if s.requestLogs == nil {
		return
	}
	status := models.RequestStatusFail
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
		if statusCode >= 200 && statusCode < 400 && execErr == nil {
			status = models.RequestStatusSuccess
		}
	}
	logEntry := &models.RequestLog{
		UserID:           meta.UserID,
		TokenID:          meta.TokenID,
		GroupID:          meta.GroupID,
		ChannelID:        r.Channel.ID,
		ChannelName:      r.Channel.Name,
		ChannelPriceRate: r.Channel.PriceRate,
		Model:            model,
		UpstreamModel:    r.UpstreamModel,
		Status:           status,
		IsStream:         isStream,
		TransferTime:     time.Since(start).Milliseconds(),
		Extra: models.RequestExtra{
			IP:          meta.IP,
			UserAgent:   meta.UA,
			RequestPath: meta.Path,
			Headers:     headersToMap(headers),
		},
		ProviderDetail: models.ProviderDetail{
			Provider:      string(r.Channel.Type),
			RequestFormat: string(meta.Format),
		},
		CreatedAt: time.Now(),
	}
	if metrics != nil {
		logEntry.InputToken = metrics.Usage.InputTokens
		logEntry.OutputToken = metrics.Usage.OutputTokens
		logEntry.CachedCreationInputTokens = metrics.Usage.CachedCreationInputTokens
		logEntry.CachedReadInputTokens = metrics.Usage.CachedReadInputTokens
		if metrics.TTFT > 0 {
			logEntry.TTFT = metrics.TTFT.Milliseconds()
		}
		if metrics.TransferTime > 0 {
			logEntry.TransferTime = metrics.TransferTime.Milliseconds()
		}
	}
	if execErr != nil {
		logEntry.ErrorMsg = execErr.Error()
	}
	_ = s.requestLogs.CreateAsync(context.WithoutCancel(ctx), logEntry)
}

func (s *RelayServiceImpl) collectLBMetrics(ctx context.Context, r *loadbalancer.Result, resp *types.Response, err error, start time.Time) {
	if s.collector == nil {
		return
	}
	record := &loadbalancer.ResultRecord{
		Model:         "",
		UpstreamModel: r.UpstreamModel,
		ChannelID:     r.Channel.ID,
		ChannelName:   r.Channel.Name,
		Duration:      time.Since(start).Milliseconds(),
	}
	if resp != nil {
		record.StatusCode = resp.StatusCode
	}
	if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 400 {
		record.Status = 1
	} else {
		record.Status = 2
	}
	_ = s.collector.CollectMetrics(context.WithoutCancel(ctx), record)
}

func headersToMap(h http.Header) map[string]string {
	if h == nil {
		return nil
	}
	m := make(map[string]string, len(h))
	for k, vs := range h {
		if len(vs) == 0 {
			continue
		}
		m[k] = vs[0]
	}
	return m
}

// abilitiesFor 根据 ChannelType 推断支持的 Ability 集合。
// M1 统一按 "三种格式全支持" 处理，具体 Channel 只要 base_url 对应即可。
func abilitiesFor(ch *models.Channel) []types.Ability {
	return []types.Ability{
		types.AbilityChat,
		types.AbilityChatCompletions,
		types.AbilityClaudeMessages,
		types.AbilityResponses,
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

```
go test ./internal/service/ -run 'TestRelay_' -v
```

Expected: 4 个子测试全绿。

- [ ] **Step 5: 编译检查**

```
go build ./...
```

若 `channel_handler.go` 因为 `NewRelayService` 签名变化而编译失败，需要在本任务内同时修它。当前签名是 `NewRelayService(s *Service, lb, collector, requestLogs)` — 新增了第 4 参数。check `cmd/server/wire/wire_gen.go` 中 `NewRelayService` 的调用位置，暂时先不动（Task 12 会一起重新生成）。

如果本步编译失败，仅因为缺 `RequestLogService` 作为参数，需要：
- 暂时保留旧签名 `NewRelayService(s, lb, collector)` 以维持 wire_gen 编译 → 改成工厂接收 optional logger

但按标准 TDD 流程：我们允许当前 wire_gen 暂时编译错误，因为 Task 12 就会修。**本 Task 的 `go build` 命令使用 `go build ./internal/... ./api/... ./pkg/...` 跳过 wire_gen**：

```
go build ./internal/... ./api/... ./pkg/...
```

Expected: PASS（wire_gen 可能暂时爆红，在 Task 12 重新生成）。

- [ ] **Step 6: Commit**

```bash
git add internal/service/relay.go internal/service/relay_test.go
git commit -m "$(cat <<'EOF'
feat(relay): 实现 RelayService.Forward 转发核心

- 保留 FetchModel，把硬编码桩替换为调用 base.Client.Models() 拉真实模型
- 新增 Forward：peek model/stream → LB.Select → Provider.Execute → 写 RequestLog
- 5xx / 429 / 网络错误触发 failover，最多尝试 3 次；4xx 原样返回
- MetricsCallback 把 usage/TTFT 回填到 RequestLog
- 引入 Provider / RequestLogCreator 接口便于测试
- 配套 4 个单元测试：无候选、非流式成功、5xx failover、4xx 不重试

wire_gen 因签名变更暂时爆红，Task 12 重新生成。

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5 — RelayHandler + `/v1` 路由

**Files:**
- Create: `internal/handler/relay_handler.go`

- [ ] **Step 1: 创建 RelayHandler**

新增 `internal/handler/relay_handler.go`：

```go
package handler

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/middleware"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/gin-gonic/gin"
)

type RelayHandler struct {
	*Handler
	RelayService service.RelayService
	TokenService service.TokenService
}

func NewRelayHandler(base *Handler, relaySvc service.RelayService, tokenSvc service.TokenService) *RelayHandler {
	return &RelayHandler{
		Handler:      base,
		RelayService: relaySvc,
		TokenService: tokenSvc,
	}
}

// RegisterRoutes 挂载在 /v1 根分组下，注意：不能复用 /api/v1。
func (h *RelayHandler) RegisterRoutes(r gin.IRouter) {
	g := r.Group("")
	g.Use(middleware.RequireToken(h.TokenService))
	g.POST("/chat/completions", h.forward)
	g.POST("/messages", h.forward)
	g.POST("/responses", h.forward)
}

func (h *RelayHandler) forward(ctx *gin.Context) {
	format := v1.RelayFormatOf(ctx.Request.URL.Path)
	if format == "" {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage("unknown relay path"), nil)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage(err.Error()), nil)
		return
	}
	defer ctx.Request.Body.Close()
	// 让下游想再读时也能拿到
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))

	meta := service.RelayMeta{
		Format:   format,
		TokenID:  getInt64FromCtx(ctx, constants.TokenIdKey()),
		UserID:   getInt64FromCtx(ctx, constants.UserIdKey()),
		GroupID:  getInt64FromCtx(ctx, constants.GroupIdKey()),
		Strategy: models.LoadBalanceStrategyPerformance,
		IP:       ctx.ClientIP(),
		UA:       ctx.Request.UserAgent(),
		Path:     ctx.Request.URL.Path,
	}

	resp, err := h.RelayService.Forward(ctx.Request.Context(), meta, body, filterForwardHeaders(ctx.Request.Header))
	if err != nil {
		if errors.Is(err, loadbalancer.ErrNoAvailableChannel) {
			v1.Fail(ctx, v1.ErrInternalServerError.WithMessage("no available channel for model"), nil)
			return
		}
		v1.Fail(ctx, v1.ErrInternalServerError.WithMessage(err.Error()), nil)
		return
	}
	if resp == nil {
		v1.Fail(ctx, v1.ErrInternalServerError.WithMessage("empty provider response"), nil)
		return
	}

	writeProviderResponse(ctx, resp)
}

func writeProviderResponse(ctx *gin.Context, resp *types.Response) {
	// 透传上游响应头中 content-* 和 openai/anthropic 特定 header
	for k, vs := range resp.Header {
		if shouldForwardHeader(k) {
			for _, v := range vs {
				ctx.Writer.Header().Add(k, v)
			}
		}
	}
	status := resp.StatusCode
	if status == 0 {
		status = http.StatusOK
	}

	if resp.IsStream && resp.Stream != nil {
		ctx.Writer.Header().Set("Content-Type", "text/event-stream")
		ctx.Writer.Header().Set("Cache-Control", "no-cache")
		ctx.Writer.Header().Set("Connection", "keep-alive")
		ctx.Writer.Header().Set("X-Accel-Buffering", "no")
		ctx.Writer.WriteHeader(status)
		pumpStream(ctx, resp)
		return
	}

	ctx.Data(status, headerContentType(resp.Header), resp.RawData)
}

func pumpStream(ctx *gin.Context, resp *types.Response) {
	flusher, _ := ctx.Writer.(http.Flusher)
	defer func() { _ = resp.Stream.Close() }()

	clientClosed := ctx.Request.Context().Done()
	for {
		select {
		case <-clientClosed:
			return
		default:
		}
		event, err := resp.Stream.Recv()
		if event != nil && len(event.Content) > 0 {
			if _, werr := ctx.Writer.Write(event.Content); werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			return
		}
		if event != nil && event.Finished {
			return
		}
	}
}

func shouldForwardHeader(k string) bool {
	lk := strings.ToLower(k)
	if strings.HasPrefix(lk, "content-") {
		return true
	}
	switch lk {
	case "openai-organization", "openai-processing-ms", "x-request-id",
		"anthropic-version", "anthropic-ratelimit-requests-limit",
		"anthropic-ratelimit-tokens-limit":
		return true
	}
	return false
}

func headerContentType(h http.Header) string {
	if h != nil {
		if ct := h.Get("Content-Type"); ct != "" {
			return ct
		}
	}
	return "application/json"
}

func filterForwardHeaders(src http.Header) http.Header {
	dst := http.Header{}
	for k, vs := range src {
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "host" || lk == "content-length" {
			continue
		}
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
	return dst
}

func getInt64FromCtx(ctx *gin.Context, key string) int64 {
	if v, ok := ctx.Get(key); ok {
		if i, ok := v.(int64); ok {
			return i
		}
	}
	return 0
}
```

- [ ] **Step 2: 编译检查**

```
go build ./internal/handler/
```

Expected: PASS。

- [ ] **Step 3: Commit**

```bash
git add internal/handler/relay_handler.go
git commit -m "$(cat <<'EOF'
feat(handler): 新增 RelayHandler 处理 /v1/chat/completions /messages /responses

- 挂载 RequireToken 中间件校验 Bearer Token
- 读完 body 缓存，提取 model/stream 参数
- 委托 RelayService.Forward 做真正的转发
- 流式：通过 resp.Stream.Recv() 循环逐 chunk 透传，支持 Flush
- 非流式：ctx.Data 原样返回 RawData
- 透传上游常用响应头（content-*, openai-*, anthropic-*）
- 过滤 Authorization/Host 等不应转发的请求头

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6 — RequestLog 查询接口

**Files:**
- Create: `api/v1/request_log.go`
- Create: `internal/handler/request_log_handler.go`

- [ ] **Step 1: 创建 v1 DTO**

创建 `api/v1/request_log.go`：

```go
package v1

import (
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type ListRequestLogsRequest struct {
	Page      int                   `form:"page"`
	PageSize  int                   `form:"pageSize"`
	OrderBy   string                `form:"orderBy"`
	UserID    string                `form:"userID"`
	TokenID   string                `form:"tokenID"`
	ChannelID string                `form:"channelID"`
	Model     string                `form:"model"`
	Status    models.RequestStatus  `form:"status"`
	StartAt   *time.Time            `form:"startAt"`
	EndAt     *time.Time            `form:"endAt"`
	RequestID string                `form:"requestID"`
}

type RequestLogResponse struct {
	*models.RequestLog
}
```

- [ ] **Step 2: 确认 `repository.RequestLogListOption` 字段**

```
rg "type RequestLogListOption" internal/repository/
```

若 option 已支持上述字段则继续；否则在本 Task 内先补。阅读 `internal/repository/request_log_repo.go`，在 `RequestLogListOption` 结构体中确保有：`Page`、`PageSize`、`OrderBy`、`UserID`、`TokenID`、`ChannelID`、`Model`、`Status`、`StartAt`、`EndAt`、`RequestID`。缺的字段加上（只需加字段 + 在 `List` 方法内对应 `Where`）。

- [ ] **Step 3: 创建 Handler**

创建 `internal/handler/request_log_handler.go`：

```go
package handler

import (
	"strconv"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/middleware"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/gin-gonic/gin"
)

type RequestLogHandler struct {
	*Handler
	Svc service.RequestLogService
}

func NewRequestLogHandler(base *Handler, svc service.RequestLogService) *RequestLogHandler {
	return &RequestLogHandler{Handler: base, Svc: svc}
}

func (h *RequestLogHandler) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/request-log")
	g.Use(middleware.RequireAuth(h.Service.JWT()))
	g.GET("", h.list)
	g.GET(":id", h.get)
}

func (h *RequestLogHandler) list(ctx *gin.Context) {
	var req v1.ListRequestLogsRequest
	if !h.BindQuery(ctx, &req) {
		return
	}
	opt := repository.RequestLogListOption{
		Page:      req.Page,
		PageSize:  req.PageSize,
		OrderBy:   req.OrderBy,
		Model:     req.Model,
		RequestID: req.RequestID,
		StartAt:   req.StartAt,
		EndAt:     req.EndAt,
	}
	if req.Status != "" {
		opt.Status = req.Status
	}
	if id, err := strconv.ParseInt(req.UserID, 10, 64); err == nil && id > 0 {
		opt.UserID = id
	}
	if id, err := strconv.ParseInt(req.TokenID, 10, 64); err == nil && id > 0 {
		opt.TokenID = id
	}
	if id, err := strconv.ParseInt(req.ChannelID, 10, 64); err == nil && id > 0 {
		opt.ChannelID = id
	}
	items, total, err := h.Svc.List(ctx.Request.Context(), opt)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	v1.Success(ctx, v1.ListResponse[*models.RequestLog]{
		Total:    total,
		Items:    items,
		Page:     int64(req.Page),
		PageSize: int64(req.PageSize),
	})
}

func (h *RequestLogHandler) get(ctx *gin.Context) {
	id, ok := h.ParseIDParam(ctx, "id")
	if !ok {
		return
	}
	log, err := h.Svc.GetByID(ctx.Request.Context(), id)
	if err != nil {
		HandleServiceError(ctx, err)
		return
	}
	if log == nil {
		HandleNotFound(ctx)
		return
	}
	v1.Success(ctx, log)
}
```

> Note：若上面的 `models.RequestLog` 在 handler 包里未 import，需要补 `"github.com/RenaLio/tudou/internal/models"`。

- [ ] **Step 4: 编译检查**

```
go build ./internal/handler/ ./api/v1/
```

Expected: PASS（repo 的 option 字段需一并补全才能过）。

- [ ] **Step 5: Commit**

```bash
git add api/v1/request_log.go internal/handler/request_log_handler.go internal/repository/request_log_repo.go
git commit -m "$(cat <<'EOF'
feat(request-log): 新增管理端请求日志查询接口

- v1 新增 ListRequestLogsRequest DTO
- 新增 RequestLogHandler 挂 /request-log 分页 + 详情接口
- RequireAuth 中间件保护
- 若缺字段则补全 repository.RequestLogListOption

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7 — 管理端鉴权补齐

**Files:**
- Modify: `internal/handler/channel_handler.go`
- Modify: `internal/handler/channel_group_handler.go`
- Modify: `internal/handler/model_handler.go`
- Modify: `internal/handler/system_config_handler.go`
- Modify: `internal/handler/stats_handler.go`

- [ ] **Step 1: channel_handler 加鉴权**

打开 `internal/handler/channel_handler.go`，在 `RegisterRoutes` 方法的 `channels := r.Group("/channel")` 之后、第一个 `channels.POST(...)` 之前插入：

```go
channels.Use(middleware.RequireAuth(h.Service.JWT()))
```

同时确保 `import` 段包含 `"github.com/RenaLio/tudou/internal/middleware"`。

- [ ] **Step 2: channel_group_handler 加鉴权**

同样在 `internal/handler/channel_group_handler.go` 的 `RegisterRoutes` 里，group 建立之后加 `g.Use(middleware.RequireAuth(h.Service.JWT()))`，import 补齐。

- [ ] **Step 3: model_handler 加鉴权**

`internal/handler/model_handler.go` 的 `RegisterRoutes` 同样处理。

- [ ] **Step 4: system_config_handler 加鉴权**

`internal/handler/system_config_handler.go` 的 `RegisterRoutes` 同样处理。

- [ ] **Step 5: stats_handler 加鉴权**

`internal/handler/stats_handler.go` 的 `RegisterRoutes` 同样处理。

- [ ] **Step 6: 编译**

```
go build ./internal/handler/
```

Expected: PASS。

- [ ] **Step 7: 手动验证**

临时起服务（wire 还没改，可能起不来，跳过这步，后续 Task 12 再整体起）。仅需 `go build ./...` 确认无编译错误。

- [ ] **Step 8: Commit**

```bash
git add internal/handler/channel_handler.go internal/handler/channel_group_handler.go internal/handler/model_handler.go internal/handler/system_config_handler.go internal/handler/stats_handler.go
git commit -m "$(cat <<'EOF'
feat(handler): 为所有管理端 CRUD 接口补齐 RequireAuth 中间件

此前仅 /self 和 /token 有 JWT 校验，
/channel、/channel-group、/model、/system-config、/stats 全部裸奔。
本次统一挂 middleware.RequireAuth，封堵后台接口。

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8 — LoadBalancer Registry 启动加载 + CRUD 同步

这是一个**必须修的现存 bug**：`Registry.ReloadChannel` 从未被调用，LB 永远空转。

**Files:**
- Modify: `internal/start/init.go`
- Modify: `internal/service/channel_service.go`
- Modify: `internal/service/channel_group_service.go`

- [ ] **Step 1: 启动时从 DB 灌 Registry**

打开 `internal/start/init.go`，把 `InitLBRegistry` 替换为能从 DB 加载的版本。先扩展签名接收 `repository.ChannelRepo` 和 `repository.ChannelGroupRepo`，然后在函数内 List 所有 channel / group 并 ReloadChannel / ReloadGroup：

```go
func InitLBRegistry(
	db *gorm.DB,
	channelRepo repository.ChannelRepo,
	groupRepo repository.ChannelGroupRepo,
) (loadbalancer.LoadBalancer, loadbalancer.MetricsCollector) {
	registry := loadbalancer.NewRegistry()

	ctx := context.Background()
	// 加载 channels
	channels, _, err := channelRepo.List(ctx, repository.ChannelListOption{Page: 1, PageSize: 10000})
	if err == nil {
		for _, ch := range channels {
			if ch != nil {
				registry.ReloadChannel(ch)
			}
		}
	}
	// 加载 groups
	groups, _, err := groupRepo.List(ctx, repository.ChannelGroupListOption{Page: 1, PageSize: 10000, PreloadChannels: true})
	if err == nil {
		for _, g := range groups {
			if g != nil {
				registry.ReloadGroup(g)
			}
		}
	}

	collector := loadbalancer.NewAsyncMetricsCollector(registry, 1024)
	lb := loadbalancer.NewDynamicLoadBalancer(registry)
	return lb, collector
}
```

> Note：`ChannelListOption` / `ChannelGroupListOption` 的实际字段名以 repo 为准。若 `PreloadChannels` 字段在 group option 中不存在，阅读 `internal/repository/channel_group_repo.go` 找到对应字段名（例如 `WithChannels` 或 `Preload`）并替换。

- [ ] **Step 2: 编译**

```
go build ./internal/start/
```

若因为 List 返回签名不对而失败，阅读对应 repo 方法签名调整。

- [ ] **Step 3: Channel CRUD 同步 Registry**

`internal/service/channel_service.go` 改造：

- 结构体 `channelService` 增加字段 `registry *loadbalancer.Registry`（以及可选 `clientReg *relay.ClientRegistry`）
- `NewChannelService` 签名加两个参数
- 在 `Create` / `Update` / `UpdateStatus` / `ReplaceGroups` 的尾部调用 `s.registry.ReloadChannel(channel)` + `s.clientReg.Invalidate(channel.ID)`
- 在 `Delete` 的尾部调用 `s.registry.UnregisterChannel(id)` + `s.clientReg.Invalidate(id)`

示例片段（Create 末尾）：

```go
latest, err := s.repo.GetByIDWithGroups(ctx, channel.ID)
if err != nil {
	return nil, err
}
if s.registry != nil {
	s.registry.ReloadChannel(latest)
}
if s.clientReg != nil {
	s.clientReg.Invalidate(latest.ID)
}
resp := toChannelResponse(latest)
return &resp, nil
```

- [ ] **Step 4: ChannelGroup CRUD 同步 Registry**

同样改造 `channel_group_service.go`：注入 `registry`，在 Create/Update/Delete 调 `registry.ReloadGroup` / `registry.UnregisterGroup`。

- [ ] **Step 5: 编译**

```
go build ./internal/...
```

若失败，按错误信息修复。

- [ ] **Step 6: Commit**

```bash
git add internal/start/init.go internal/service/channel_service.go internal/service/channel_group_service.go
git commit -m "$(cat <<'EOF'
fix(loadbalancer): 启动时从 DB 加载渠道和分组到 Registry，CRUD 时同步

此前 Registry.ReloadChannel / ReloadGroup 定义了但从未被调用，
导致 LoadBalancer.Select 永远返回空，所有转发请求必然失败。

- InitLBRegistry 启动时一次性灌入所有 channel/group
- ChannelService / ChannelGroupService 在 Create/Update/Delete 时
  调用 Registry.Reload* 或 Unregister*，并失效 ClientRegistry

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9 — Wire 重组并重新生成

**Files:**
- Modify: `cmd/server/wire/wire.go`
- Modify: `internal/router/deps.go`
- Modify: `cmd/server/wire/wire_gen.go`（由 wire 工具生成）

- [ ] **Step 1: 更新 router.Deps**

打开 `internal/router/deps.go`，添加新字段：

```go
type Deps struct {
	Conf                *config.Config
	Logger              *log.Logger
	ModelHandler        *handler.ModelHandler
	ChannelHandler      *handler.ChannelHandler
	ChannelGroupHandler *handler.ChannelGroupHandler
	TokenHandler        *handler.TokenHandler
	UserHandler         *handler.UserHandler
	SystemConfigHandler *handler.SystemConfigHandler
	StatsHandler        *handler.StatsHandler
	RelayHandler        *handler.RelayHandler
	RequestLogHandler   *handler.RequestLogHandler
}
```

- [ ] **Step 2: 更新 wire.go**

打开 `cmd/server/wire/wire.go`，在 `handlerSet` 内加入：

```go
handler.NewRelayHandler,
handler.NewRequestLogHandler,
```

在 `InitLBRegistry` 会自动从 depsSet 中被调用，但它签名变了（加了 channelRepo、groupRepo 参数）。确认 wire 能自动注入，这两个 repo 已在 `repositorySet` 里 → OK，不用改 wire_gen 手工调用。

若 `NewRelayService` 签名改了（加了 RequestLogService 参数），它也会被 wire 自动解析 → OK。

若 `NewChannelService` / `NewChannelGroupService` 加了 `*loadbalancer.Registry` / `*relay.ClientRegistry` 参数，需要在 `depsSet` 或 `serviceSet` 里 provide：

```go
var depsSet = wire.NewSet(
	jwt.NewJwt,
	sid.NewSid,
	loadbalancer.NewDynamicLoadBalancer,
	loadbalancer.NewAsyncMetricsCollector,
	loadbalancer.NewRegistry,     // 新增
	relay.NewClientRegistry,       // 新增（带 *http.Client 参数，需一起 provide）
	httpclient.GetDefaultClient,   // 新增
)
```

但目前 `InitLBRegistry` 把 Registry 包在返回值里。最小改动方案：**让 `InitLBRegistry` 额外返回 Registry 和 ClientRegistry**，在 wire.go 的 `wire.Build` 里用 `wire.FieldsOf` 或 `wire.StructProvider` 分解。

更直接：改签名为
```go
func InitLBRegistry(
	db *gorm.DB,
	channelRepo repository.ChannelRepo,
	groupRepo repository.ChannelGroupRepo,
) (loadbalancer.LoadBalancer, loadbalancer.MetricsCollector, *loadbalancer.Registry, *relay.ClientRegistry)
```

返回四个值，wire 自动从这里拿每一个。

- [ ] **Step 3: 更新 InitLBRegistry 返回签名**

同步修改 `internal/start/init.go`：

```go
func InitLBRegistry(
	db *gorm.DB,
	channelRepo repository.ChannelRepo,
	groupRepo repository.ChannelGroupRepo,
) (loadbalancer.LoadBalancer, loadbalancer.MetricsCollector, *loadbalancer.Registry, *relay.ClientRegistry) {
	registry := loadbalancer.NewRegistry()

	ctx := context.Background()
	channels, _, err := channelRepo.List(ctx, repository.ChannelListOption{Page: 1, PageSize: 10000})
	if err == nil {
		for _, ch := range channels {
			if ch != nil {
				registry.ReloadChannel(ch)
			}
		}
	}
	groups, _, err := groupRepo.List(ctx, repository.ChannelGroupListOption{Page: 1, PageSize: 10000, PreloadChannels: true})
	if err == nil {
		for _, g := range groups {
			if g != nil {
				registry.ReloadGroup(g)
			}
		}
	}

	collector := loadbalancer.NewAsyncMetricsCollector(registry, 1024)
	lb := loadbalancer.NewDynamicLoadBalancer(registry)
	clientReg := relay.NewClientRegistry(httpclient.GetDefaultClient())
	return lb, collector, registry, clientReg
}
```

- [ ] **Step 4: 重新生成 wire_gen.go**

```
cd cmd/server/wire && go run -mod=mod github.com/google/wire/cmd/wire ./...
```

Expected: 产出新的 `wire_gen.go`，无报错。若 wire 命令报错信息提示缺 provider，按提示在 `wire.go` 的各个 Set 中补全。

- [ ] **Step 5: 全工程编译**

```
go build ./...
```

Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add cmd/server/wire/ internal/router/deps.go internal/start/init.go
git commit -m "$(cat <<'EOF'
chore(wire): 接入 RelayHandler / RequestLogHandler 并重新生成

- router.Deps 新增 RelayHandler、RequestLogHandler 字段
- handlerSet 中 provide 对应构造函数
- InitLBRegistry 扩展返回 Registry + ClientRegistry 以便 channel/group
  service 注入
- 重新生成 wire_gen.go

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10 — 路由注册

**Files:**
- Modify: `internal/router/router.go`

- [ ] **Step 1: 挂载 /v1 + 验证新 handler**

打开 `internal/router/router.go`，在 `apiV1Group := engine.Group("/api/v1")` 的块之后、`return nil` 之前追加：

```go
// Relay 路由走 /v1 根分组（OpenAI/Claude 协议客户端默认 base_url 约定）
if deps.RelayHandler != nil {
	relayV1 := engine.Group("/v1")
	relayV1.Use(middleware.RequestID(deps.Logger))
	deps.RelayHandler.RegisterRoutes(relayV1)
}
```

同时在 `apiV1Group` 块内把 `RequestLogHandler` 也挂上：

```go
if deps.RequestLogHandler != nil {
	deps.RequestLogHandler.RegisterRoutes(apiV1Group)
}
```

并在开头的 nil 检查中加入：

```go
if deps.RelayHandler == nil {
	return errors.New("relay handler is nil")
}
if deps.RequestLogHandler == nil {
	return errors.New("request log handler is nil")
}
```

- [ ] **Step 2: 启动服务器本地冒烟**

```
go run ./cmd/server -conf config/config.yaml
```

Expected: 服务启动，不崩溃；日志里能看到"server start"。`Ctrl+C` 停掉。

- [ ] **Step 3: curl 探测路由存在**

重启后端，另开终端：

```
curl -i http://localhost:8080/v1/chat/completions
```

Expected: `401 Unauthorized`（因为缺 Token，正是期望）。如果是 `404` 则说明路由没注册，检查上面修改。

- [ ] **Step 4: Commit**

```bash
git add internal/router/router.go
git commit -m "$(cat <<'EOF'
feat(router): 注册 /v1 relay 路由与 /api/v1/request-log 管理路由

- /v1/* 作为 OpenAI/Claude 协议转发入口，挂 RequestID 中间件
- /api/v1/request-log 作为后台日志查询接口
- 增加 nil 检查避免启动时 panic 不清晰

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11 — 前端：RequestLog API 客户端

**Files:**
- Create: `web/src/api/request-log.ts`

- [ ] **Step 1: 读现有 api/ 客户端风格**

```
cat web/src/api/client.ts
cat web/src/api/stats.ts
```

观察现有用法（`http` 单例、`Response` 包装等），保持风格一致。

- [ ] **Step 2: 创建 api/request-log.ts**

```typescript
import { http } from './client'

export interface RequestLog {
  id: string
  requestID: string
  userID: string
  tokenID: string
  groupID: string
  channelID: string
  channelName: string
  channelPriceRate: number
  model: string
  upstreamModel: string
  inputToken: number
  outputToken: number
  cachedCreationInputTokens: number
  cachedReadInputTokens: number
  costMicros: number
  status: 'success' | 'fail'
  ttft: number
  transferTime: number
  errorCode?: string
  errorMsg?: string
  isStream: boolean
  extra?: {
    ip?: string
    userAgent?: string
    requestPath?: string
    headers?: Record<string, string>
  }
  providerDetail?: {
    provider?: string
    requestFormat?: string
    transFormat?: string
  }
  createdAt: string
}

export interface ListRequestLogsParams {
  page?: number
  pageSize?: number
  orderBy?: string
  userID?: string
  tokenID?: string
  channelID?: string
  model?: string
  status?: 'success' | 'fail'
  startAt?: string
  endAt?: string
  requestID?: string
}

export interface ListRequestLogsResponse {
  total: number
  items: RequestLog[]
  page: number
  pageSize: number
}

export function listRequestLogs(params: ListRequestLogsParams) {
  return http.get<ListRequestLogsResponse>('/request-log', { params })
}

export function getRequestLog(id: string) {
  return http.get<RequestLog>(`/request-log/${id}`)
}
```

- [ ] **Step 3: 前端类型检查**

```
cd web && bun run type-check
```

Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add web/src/api/request-log.ts
git commit -m "$(cat <<'EOF'
feat(web): 新增请求日志 API 客户端

对接 /api/v1/request-log GET/GET(:id) 接口，
提供 listRequestLogs 和 getRequestLog 两个调用。

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12 — 前端：请求日志视图

**Files:**
- Create: `web/src/views/RequestLogsView.vue`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/layouts/MainLayout.vue`

- [ ] **Step 1: 读现有 ChannelsView 或 TokensView 看风格**

```
cat web/src/views/ChannelsView.vue
```

注意数据加载、表格渲染、翻页、空态、骨架屏的模式，保持一致。

- [ ] **Step 2: 创建 RequestLogsView.vue**

最小可用版本（分页表格 + 状态过滤 + 点击行显示详情）。以下为骨架，复用项目现有表格组件样式：

```vue
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { listRequestLogs, type RequestLog, type ListRequestLogsParams } from '@/api/request-log'

const loading = ref(false)
const items = ref<RequestLog[]>([])
const total = ref(0)
const params = ref<ListRequestLogsParams>({ page: 1, pageSize: 20 })
const selected = ref<RequestLog | null>(null)

const totalPages = computed(() => Math.ceil(total.value / (params.value.pageSize || 20)))

async function load() {
  loading.value = true
  try {
    const res = await listRequestLogs(params.value)
    items.value = res.items ?? []
    total.value = res.total ?? 0
  } finally {
    loading.value = false
  }
}

function setStatus(v: 'success' | 'fail' | undefined) {
  params.value.status = v
  params.value.page = 1
  load()
}

function goPage(p: number) {
  if (p < 1 || p > totalPages.value) return
  params.value.page = p
  load()
}

function fmtMicros(v: number) {
  return `$${(v / 1_000_000).toFixed(6)}`
}

function fmtTime(iso: string) {
  return new Date(iso).toLocaleString()
}

onMounted(load)
</script>

<template>
  <div class="p-6">
    <div class="flex items-center gap-4 mb-4">
      <h1 class="text-2xl font-bold">请求日志</h1>
      <div class="flex gap-2">
        <button class="px-3 py-1 rounded border" :class="{ 'bg-primary text-white': !params.status }" @click="setStatus(undefined)">全部</button>
        <button class="px-3 py-1 rounded border" :class="{ 'bg-green-500 text-white': params.status === 'success' }" @click="setStatus('success')">成功</button>
        <button class="px-3 py-1 rounded border" :class="{ 'bg-red-500 text-white': params.status === 'fail' }" @click="setStatus('fail')">失败</button>
      </div>
    </div>

    <div v-if="loading" class="py-8 text-center text-gray-500">加载中...</div>
    <table v-else class="w-full border-collapse">
      <thead>
        <tr class="text-left border-b">
          <th class="p-2">时间</th>
          <th class="p-2">模型</th>
          <th class="p-2">渠道</th>
          <th class="p-2">状态</th>
          <th class="p-2">输入/输出</th>
          <th class="p-2">TTFT</th>
          <th class="p-2">耗时</th>
          <th class="p-2">成本</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in items" :key="row.id" class="border-b cursor-pointer hover:bg-gray-50" @click="selected = row">
          <td class="p-2">{{ fmtTime(row.createdAt) }}</td>
          <td class="p-2">{{ row.model }} → {{ row.upstreamModel }}</td>
          <td class="p-2">{{ row.channelName }}</td>
          <td class="p-2">
            <span :class="row.status === 'success' ? 'text-green-600' : 'text-red-600'">{{ row.status }}</span>
          </td>
          <td class="p-2">{{ row.inputToken }}/{{ row.outputToken }}</td>
          <td class="p-2">{{ row.ttft }}ms</td>
          <td class="p-2">{{ row.transferTime }}ms</td>
          <td class="p-2">{{ fmtMicros(row.costMicros) }}</td>
        </tr>
        <tr v-if="!items.length">
          <td class="p-4 text-center text-gray-400" colspan="8">暂无日志</td>
        </tr>
      </tbody>
    </table>

    <div class="flex justify-between items-center mt-4" v-if="totalPages > 1">
      <span class="text-sm text-gray-500">共 {{ total }} 条</span>
      <div class="flex gap-1">
        <button class="px-3 py-1 border rounded" @click="goPage((params.page ?? 1) - 1)">上一页</button>
        <span class="px-3 py-1">{{ params.page }} / {{ totalPages }}</span>
        <button class="px-3 py-1 border rounded" @click="goPage((params.page ?? 1) + 1)">下一页</button>
      </div>
    </div>

    <div v-if="selected" class="fixed inset-0 bg-black/50 flex justify-end z-50" @click.self="selected = null">
      <aside class="w-[520px] bg-white h-full p-6 overflow-auto">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-bold">日志详情</h2>
          <button @click="selected = null" class="text-gray-500">×</button>
        </div>
        <dl class="text-sm">
          <dt class="font-semibold">RequestID</dt><dd class="mb-2">{{ selected.requestID }}</dd>
          <dt class="font-semibold">Model</dt><dd class="mb-2">{{ selected.model }} → {{ selected.upstreamModel }}</dd>
          <dt class="font-semibold">Channel</dt><dd class="mb-2">{{ selected.channelName }}</dd>
          <dt class="font-semibold">Status</dt><dd class="mb-2">{{ selected.status }} <span v-if="selected.errorMsg">- {{ selected.errorMsg }}</span></dd>
          <dt class="font-semibold">Usage</dt><dd class="mb-2">in={{ selected.inputToken }} out={{ selected.outputToken }} cache-create={{ selected.cachedCreationInputTokens }} cache-read={{ selected.cachedReadInputTokens }}</dd>
          <dt class="font-semibold">Latency</dt><dd class="mb-2">TTFT={{ selected.ttft }}ms transfer={{ selected.transferTime }}ms</dd>
          <dt class="font-semibold">Stream</dt><dd class="mb-2">{{ selected.isStream ? '是' : '否' }}</dd>
          <dt class="font-semibold">Cost</dt><dd class="mb-2">{{ fmtMicros(selected.costMicros) }}</dd>
          <dt class="font-semibold">IP / UA</dt><dd class="mb-2">{{ selected.extra?.ip }} / {{ selected.extra?.userAgent }}</dd>
          <dt class="font-semibold">Created</dt><dd class="mb-2">{{ fmtTime(selected.createdAt) }}</dd>
        </dl>
      </aside>
    </div>
  </div>
</template>
```

- [ ] **Step 3: 路由 + 菜单**

`web/src/router/index.ts` 的 children 数组里添加：

```typescript
{
  path: 'request-logs',
  name: 'request-logs',
  component: () => import('@/views/RequestLogsView.vue'),
},
```

`web/src/layouts/MainLayout.vue` 里的菜单组里加一条（位置按现有风格）：

```html
<RouterLink to="/request-logs">请求日志</RouterLink>
```

（具体 DOM 结构按现有菜单样式模仿。）

- [ ] **Step 4: 类型检查 + dev 起前端**

```
cd web && bun run type-check
bun run dev
```

浏览器打开 http://localhost:5173，登录后点击"请求日志"菜单，应能看到空表格（后端此时还没数据）。

- [ ] **Step 5: Commit**

```bash
git add web/src/views/RequestLogsView.vue web/src/router/index.ts web/src/layouts/MainLayout.vue
git commit -m "$(cat <<'EOF'
feat(web): 新增请求日志视图

- RequestLogsView.vue：分页表格 + 状态过滤 + 详情抽屉
- 路由 + 菜单接入

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13 — 端到端冒烟验证

**Files:** 无（仅操作 + 观测）

- [ ] **Step 1: 起后端**

```
go run ./cmd/server -conf config/config.yaml
```

- [ ] **Step 2: 起前端**

```
cd web && bun run dev
```

- [ ] **Step 3: 用默认 admin/admin 登录后台**

打开 http://localhost:5173/login，用户名 `admin` 密码 `admin`。进入 Dashboard。

- [ ] **Step 4: 创建一个 channel**

后台 → 渠道管理 → 新建。填写：
- Type: `openai`
- Name: `local-test`
- BaseURL: `https://api.openai.com`（或你有 key 的兼容服务）
- APIKey: `sk-xxx`
- 绑定到 `default` 分组
- Model: `gpt-4o-mini`（至少填一个上游支持的模型）
- Status: `enabled`

保存。

- [ ] **Step 5: 创建一个 Token**

Token 管理 → 新建，关联 admin 用户 + default 分组。复制生成的 token 字符串（`sk-xxxxxxxx...`）。

- [ ] **Step 6: curl 测试非流式**

```
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <sk-token>" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}],"stream":false}'
```

Expected: 返回 OpenAI 格式响应 JSON，状态码 200。

- [ ] **Step 7: curl 测试流式**

```
curl -N -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <sk-token>" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"数到 10"}],"stream":true}'
```

Expected: 看到 `data: {...}\n\ndata: {...}\n\n...` 的 SSE 流，最后 `data: [DONE]`。

- [ ] **Step 8: curl 测试 Claude 协议**

```
curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <sk-token>" \
  -d '{"model":"claude-3-5-sonnet-20241022","max_tokens":256,"messages":[{"role":"user","content":"hi"}]}'
```

Expected: 如果你的渠道是 OpenAI 且配置了格式转换，会自动 OpenAI→Claude 转换；返回 Claude 风格 JSON。若上游不支持该模型则可能 4xx，这是正常的。

- [ ] **Step 9: 前端看日志**

回到后台 → 请求日志页。应能看到刚才的几条请求，点进去能看到 usage / 延迟 / 响应 headers 等。

- [ ] **Step 10: 如果全部成功，写冒烟结论**

在仓库根目录创建临时文件 `docs/superpowers/plans/2026-04-19-M1-smoke-result.md` 记录本次验证结果（可 commit 也可不 commit，仅作日志）。

- [ ] **Step 11: 最终提交**

如果前面 Task 没 issue 需要修，本任务不产生代码修改。可以 tag 一下：

```bash
git tag -a m1-complete -m "M1 MVP 完成"
```

---

## 验收检查

完成所有 Task 后，对照 spec 的 M1 验收：

- [x] `POST /v1/chat/completions` 可用
- [x] `POST /v1/messages` 可用
- [x] `POST /v1/responses` 可用
- [x] SSE 流式透传
- [x] Token 鉴权（Bearer）
- [x] 管理端 JWT 鉴权覆盖 /channel /channel-group /model /system-config /stats
- [x] RequestLog 异步写入并可管理端查询
- [x] 三 Provider 平台（OpenAI / Claude / OpenAI-compatible）通过 base.Client 工作
- [x] FetchModel 实装
- [x] LB Registry 启动加载 + CRUD 同步（修复空转 bug）
- [x] 前端请求日志页
