<div align="center">

# Tudou LLM Gateway

**一个面向个人使用的 LLM API 网关，用于统一管理多上游渠道、模型路由、负载均衡、请求日志、用量统计与计费。**

**A personal LLM API gateway for multi-upstream aggregation, model routing, load balancing, request logging, usage analytics, and billing management.**

</div>

## Features

- **Multi-Protocol Relay** - 同时支持 OpenAI Chat Completions、OpenAI Responses、OpenAI Embeddings 和 Claude Messages，并在内置 translator 中处理协议转换。
- **Provider Request Path Tracking** - 请求日志会记录实际命中的上游请求路径，便于排查不同平台、不同协议适配下的真实转发行为。
- **Retry Trace as First-Class Log Data** - 渠道失败重试不会只留下最终错误，会记录每次尝试的渠道、模型、状态码、响应体和实际上游路径。
- **Metric-Aware Load Balancing** - 负载均衡可结合 TTFT、TPS、成功率、成本、权重、最少连接等指标，而不是只做简单随机或轮询。
- **Channel Model Sync Filters** - 自动同步上游模型列表时支持白名单 / 黑名单正则过滤，适合上游模型很多但只想暴露部分模型的场景。
- **Pricing With Cache & Long Context** - 模型价格支持输入、输出、缓存读写、长上下文档位和渠道价格倍率，方便更细粒度估算成本。
- **Channel Groups & Token Isolation** - Token 绑定渠道组，不同客户端可以使用不同渠道池、模型集合、额度和路由策略。
- **Built-in Model Catalog Sync** - 支持从模型价格数据源同步模型信息，同时允许手动覆盖自定义价格。
- **Operational Dashboard** - 管理后台覆盖渠道、分组、Token、模型、请求日志、用量统计和系统配置，适合个人或小团队自托管使用。
- **Self-Hosted Storage Choice** - 支持 SQLite 开箱即用，也可以切换 MySQL / PostgreSQL。

## Quick Start

### Docker Compose

推荐使用 Docker Compose 启动：

```bash
docker compose up -d
```

如果需要本地重新构建镜像：

```bash
docker compose up -d --build
```

查看日志：

```bash
docker compose logs -f tudou
```

访问管理后台：

```text
http://localhost:8080
```

### Docker Image

GHCR:

```bash
docker pull ghcr.io/renalio/tudou:latest
```

Docker Hub:

```bash
docker pull renalio/tudou:latest
```

示例运行：

```bash
docker run -d \
  --name tudou \
  -p 8080:8080 \
  -v ./config:/app/config:ro \
  -v tudou-storage:/app/storage \
  ghcr.io/renalio/tudou:latest
```

### Download from Release

从 [Releases](https://github.com/RenaLio/tudou/releases) 下载对应平台的压缩包，解压后运行：

Linux / macOS:

```bash
./tudou -conf config/config.yaml
```

Windows:

```powershell
.\tudou.exe -conf config\config.yaml
```

### Build from Source

Requirements:

- Go 1.26+
- Bun

```bash
git clone https://github.com/RenaLio/tudou.git
cd tudou

cd web
bun install
bun run build-only
cd ..

go run ./cmd/server -conf config/config.yaml
```

> Tip: 前端构建产物会通过 Go embed 打进服务端二进制，因此直接从源码运行服务端前需要先生成 `web/dist`。

## Default Credentials

首次启动后访问 `http://localhost:8080`，使用默认管理员账号登录：

- Username: `admin`
- Password: `admin`

> Security Notice: 首次登录后请立即修改默认密码。

## Configuration

默认配置文件：

```text
config/config.yaml
```

常见配置项：

| Option | Description | Default |
| --- | --- | --- |
| `env` | 运行环境 | `dev` |
| `http.host` | HTTP 监听地址 | `0.0.0.0` |
| `http.port` | HTTP 监听端口 | `8080` |
| `data.db.user.driver` | 数据库类型 | `sqlite` |
| `data.db.user.dsn` | 数据库连接字符串 | `storage/tudou.db...` |
| `security.jwt.secret` | JWT 密钥 | 请在生产环境修改 |
| `log.log_level` | 日志级别 | `debug` |
| `log.log_path` | 日志目录 | `./storage/logs` |

数据库支持：

| Type | Driver | DSN Example |
| --- | --- | --- |
| SQLite | `sqlite` | `storage/tudou.db?_busy_timeout=5000` |
| MySQL | `mysql` | `user:password@tcp(127.0.0.1:3306)/tudou?charset=utf8mb4&parseTime=True&loc=Local` |
| PostgreSQL | `postgres` | `postgres://user:password@127.0.0.1:5432/tudou?sslmode=disable` |

生产环境建议：

- 修改默认管理员密码。
- 修改 `security.jwt.secret`。
- 使用挂载的 `config/` 覆盖镜像内置配置。
- 持久化 `/app/storage`，避免数据库和日志随容器删除。
- 不要把真实密钥提交到仓库。

## Documentation

### Channel Management

Channel 是连接上游 LLM Provider 的基础配置单元。

常见字段：

- `Type`: 平台类型或兼容协议类型。
- `Base URL`: 上游服务地址。
- `API Key`: 上游服务密钥。
- `Model`: 自动同步或手动维护的模型列表。
- `Custom Model`: 自定义模型列表。
- `Model Mappings`: 调用模型名到上游模型名的映射。
- `Price Rate`: 渠道价格倍率。
- `Auto Sync Upstream Models`: 自动同步上游模型列表。

### Channel Group

Channel Group 用于把多个渠道聚合成一个可分配给 Token 的路由集合。

- Token 绑定一个渠道组。
- 请求进入网关后，会在该渠道组内选择可用渠道。
- 渠道组和 Token 都可以配置负载均衡策略。

### Token

Token 是外部调用 Tudou Relay API 的访问凭据。

你可以为不同客户端创建不同 Token，并分别配置：

- 绑定用户
- 绑定渠道组
- 用量限制
- 过期时间
- 负载均衡策略

### Model Pricing

模型价格用于估算请求成本。

支持：

- 按 token 计费
- 按请求计费
- 缓存读写 token 价格
- 长上下文价格
- 渠道价格倍率
- 自动价格同步和手动价格覆盖

### Request Logs

请求日志会记录完整调用信息，包括：

- 用户 / Token
- 渠道 / 上游模型
- 输入 / 输出 token
- 成本
- TTFT / Transfer Time
- 请求状态和错误信息
- 实际上游请求路径
- 重试链路

## Release Archives

GitHub Release 会生成以下平台的 zip 包：

- Linux amd64 / arm64
- macOS amd64 / arm64
- Windows amd64 / arm64

每个 zip 包包含：

- 可执行文件
- `LICENSE`
- `config/`

## Acknowledgments

- [sst/models.dev](https://github.com/sst/models.dev) - AI model pricing data source.

## License

See [LICENSE](LICENSE).
