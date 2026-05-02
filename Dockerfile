# ============================================================================
# Builder stage
# ============================================================================
# 关键:--platform=$BUILDPLATFORM 让 builder 用宿主机原生架构运行,
# 不被 QEMU 模拟拖慢。Go 工具链本身支持交叉编译,通过下面的
# GOOS/GOARCH 直接产出目标平台二进制。
FROM --platform=$BUILDPLATFORM golang:1.26.1-alpine AS builder

# BuildKit 自动注入:
#   BUILDPLATFORM/BUILDOS/BUILDARCH = 构建主机(如 linux/amd64)
#   TARGETPLATFORM/TARGETOS/TARGETARCH = 目标平台(如 linux/arm64)
# 注意:ARG 不能给默认值,否则在某些 BuildKit 实现里
# 默认值会抢占 BuildKit 自动注入,导致 GOARCH 永远是默认值
ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY=https://goproxy.cn,direct

# `${VAR:-default}` 形式做 fallback,既支持 buildx 注入也兼容传统 docker build
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    GOPROXY=${GOPROXY} \
    GOSUMDB=sum.golang.org

# 诊断:打印构建时实际拿到的目标平台。
# 构建日志里这一行如果显示 "linux/amd64" 而你期望 arm64,说明 buildx 没传 --platform
#RUN echo ">>> Building for $(go env GOOS)/$(go env GOARCH)"

WORKDIR /build

# 优先复制依赖描述,利用层缓存
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 复制源码并构建静态二进制
# - SQLite 使用 modernc.org/sqlite 纯 Go 实现,无需 CGO
# - -trimpath 让构建可重现; -s -w 去除调试符号缩小体积
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ============================================================================
# Runtime stage
# ============================================================================
FROM alpine:3.20

# 运行时必要依赖:
# - ca-certificates: 调用上游 LLM HTTPS API 必需
# - tzdata:        不在镜像里写死时区,留给 compose 通过 TZ 环境变量配置
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户(UID/GID 1000,与 compose volume 挂载权限一致)
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -D -H -s /sbin/nologin appuser

WORKDIR /app

# 拷贝二进制与默认配置(运行时配置可由 compose 通过 volume 覆盖)
COPY --from=builder /out/server       /app/server
COPY --from=builder /build/config     /app/config

# 预创建数据/日志目录;真实数据预期由 compose 挂载 volume 接管
RUN mkdir -p /app/storage/logs && \
    chown -R appuser:appgroup /app

USER appuser:appgroup

EXPOSE 8080

# 使用 alpine 自带的 busybox wget 做存活探测
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://127.0.0.1:8080/v1/healthz || exit 1

ENTRYPOINT ["/app/server"]
CMD ["-conf", "config/config.yaml"]
