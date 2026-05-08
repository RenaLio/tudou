# Tudou — LLM Gateway

A personal LLM API gateway. Proxies multiple LLM providers with load balancing, usage tracking, billing, and an admin dashboard.

## Project Structure

```
tudou/
├── api/v1/                  # API response types, error codes
├── cmd/
│   ├── server/              # Main entry + Wire DI
│   └── server_example/      # Example server
├── config/                  # YAML config files
├── internal/
│   ├── config/              # Config loading (Viper)
│   ├── constants/           # Constants (context keys, etc.)
│   ├── handler/             # HTTP handlers (Gin)
│   ├── helpers/             # Request body parsing helpers
│   ├── loadbalancer/        # Load balancer (multi-strategy, async metrics)
│   ├── middleware/           # Auth middleware (JWT + Token)
│   ├── models/              # Domain models (GORM)
│   ├── pkg/                 # Internal shared libs (log, jwt, sid, http server, app)
│   ├── repository/          # Data access layer (GORM + transaction management)
│   ├── router/              # Route registration
│   ├── server/              # HTTP server init + DB migration
│   ├── service/             # Business logic layer
│   ├── start/               # App init (default user, group, LB registry warmup)
│   ├── store/               # In-memory cache (model prices)
│   ├── tasks/               # Background tasks (stats aggregation, price sync)
│   └── types/               # Internal type definitions
├── pkg/
│   ├── cache/               # JSON cache (BigCache)
│   ├── httpclient/          # HTTP client (supports disabling HTTP/2)
│   ├── provider/            # Provider abstraction + multi-platform implementations
│   │   ├── platforms/       # Per-platform LLM adapters
│   │   ├── translator/      # Request/response format translation (OpenAI ↔ Claude ↔ Responses)
│   │   ├── types/           # Provider interface types
│   │   └── plog/            # Provider logging
│   └── timex/               # Time utilities
└── web/                     # Vue 3 frontend (see web/AGENTS.md)
```

## Tech Stack

**Backend**
- Go 1.26 + Gin + GORM + Wire (DI)
- SQLite (default) / MySQL / PostgreSQL
- Viper (config), Zap (logging), Snowflake (ID generation)
- BigCache (in-memory cache)

**Frontend**
- Vue 3 + TypeScript + Vite + Tailwind CSS v4
- Pinia (state) + @tanstack/vue-query (server state caching)
- echarts (charts), reka-ui (headless UI primitives)

## Core Concepts

| Concept | Description |
|---|---|
| **Channel** | Upstream LLM provider (base_url + api_key + type). Supports model lists, custom model mappings, and price rate multiplier. |
| **ChannelGroup** | Group of channels. Tokens bind to groups; load balancing strategy is shared across channels in a group. |
| **Token** | Relay API access token. Bound to a user and channel group. Supports usage limits, expiration, and strategy override. |
| **User** | Admin dashboard user (JWT auth). Default: admin/admin. |
| **AIModel** | Model definition + pricing info (per-token / per-request, with 200K context threshold pricing). |
| **RequestLog** | Request log capturing full metrics per relay call (TTFT, TPS, token usage, cost, retry chain). |

## Relay Request Flow

```
Client → /v1/chat/completions (Bearer Token)
  → RequireToken middleware (validate Token → inject TokenClaim)
  → RelayHandler.forward()
    → Parse body to get model name
    → LoadBalancer.Select() (rank candidates by strategy)
    → Up to 3 retries
      → buildProvider() creates platform adapter
      → provider.Execute() forwards request
      → Async metrics collection (TTFT/TPS/success rate)
      → Return on success, try next candidate on failure
    → Record RequestLog (including retry trace)
  → Return stream / non-stream response
```

## Load Balancing Strategies

`performance` (default), `random`, `ttft_first`, `tps_first`, `success_first`, `cost_first`, `weighted`, `least_conn`

Weighted scoring based on real-time metrics (success rate, TTFT, TPS, weight, cost). 10% random jitter to prevent thundering herd.

## API Routes

**Management API** (`/api/v1/...`, JWT auth)
- `/api/v1/user/login`, `/api/v1/user/register` — Auth (public)
- `/api/v1/channels`, `/api/v1/channel-groups`, `/api/v1/tokens`, `/api/v1/models` — CRUD
- `/api/v1/stats/...` — Usage statistics
- `/api/v1/request-logs` — Request logs
- `/api/v1/system-config` — System config

**Relay API** (`/v1/...`, Token auth)
- `POST /v1/chat/completions` — OpenAI Chat Completions
- `POST /v1/messages` — Claude Messages
- `POST /v1/embeddings` — OpenAI Embeddings
- `POST /v1/responses` — OpenAI Responses
- `GET /v1/models` — Token available model list

## Development Commands

```sh
# Backend
go run ./cmd/server -conf config/config.yaml

# Frontend (dev, proxies /api → localhost:8080)
cd web && bun install && bun dev

# Frontend build
cd web && bun run build

# Docker
docker compose up -d --build

# Regenerate Wire DI
cd cmd/server/wire && wire

# Frontend lint & format
cd web && bun run lint && bun run format
```

## Coding Conventions

### Backend

- Strict layering: `handler → service → repository`. Reverse dependencies are forbidden.
- All new handlers must be registered in `wire.go` and injected via Wire.
- Handlers decouple service dependencies through interfaces (e.g. `RelayService`, `StatsService`).
- Repository uses `r.DB(ctx)` for transaction-aware gorm.DB. For write operations involving cache invalidation, use `r.onCommitted(ctx, callback)` to register post-commit callbacks.
- ID generation: always use `sid.Sid` (Snowflake).
- Error handling: return `v1.AppError` types, use `v1.Fail()` for unified responses.
- JSON fields use `goccy/go-json`. Nested struct fields in models implement `driver.Valuer` / `sql.Scanner`.
- Soft delete: uses `gorm.io/plugin/soft_delete`.

### Frontend

- Vue 3 Composition API + `<script setup>`.
- Request/response types and API functions live in the same file (`src/api/*.ts`).
- Date handling: always use dayjs (`src/utils/date.ts`). API payloads use RFC3339 format.
- Style layers: `theme.css` (CSS custom properties) → `tailwind.css` → `styles/main.scss`.
- Vue templates: never use self-closing HTML tags (`<div />` → `<div></div>`).

## Database

Default: SQLite (zero-config, file at `storage/tudou.db`). Also supports MySQL and PostgreSQL. Switch via `data.db.user` in `config/config.yaml`.

## Background Tasks

- **StatsAggregation** — Periodically aggregates request logs into statistics tables.
- **PriceSync** — Syncs model pricing from models.dev every 12 hours.
