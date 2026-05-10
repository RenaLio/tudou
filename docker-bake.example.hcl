// =============================================================================
// docker buildx bake 配置文件
// 使用方法:
//   docker buildx bake           # 本地构建（仅当前架构）
//   docker buildx bake push      # 多平台构建并推送到仓库
//   docker buildx bake --set *.platform=linux/amd64  # 仅构建指定架构
// =============================================================================

// 镜像仓库和标签配置
variable "REGISTRY" {
  default = "docker.io"  // 改为你的仓库地址
}

variable "IMAGE_NAME" {
  default = "yourname/tudou"  // 改为你的镜像名
}

variable "VERSION" {
  default = "latest"
}

// 默认组：本地构建
group "default" {
  targets = ["local"]
}

// 推送组：多平台构建并推送
group "push" {
  targets = ["release"]
}

// 本地构建目标
target "local" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64"]
  tags       = ["tudou:latest"]
  output     = ["type=docker"]

  args = {
    GOPROXY = "https://goproxy.cn,direct"
  }
}

// 发布目标（多平台推送到仓库）
target "release" {
  context    = "."
  dockerfile = "Dockerfile"

  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]

  tags = [
    "${REGISTRY}/${IMAGE_NAME}:latest",
    notequal("latest", VERSION) ? "${REGISTRY}/${IMAGE_NAME}:${VERSION}" : "",
  ]

  args = {
    GOPROXY = "https://goproxy.cn,direct"
  }

  // 注意：阿里云等部分仓库不支持 BuildKit 缓存格式
  // 如需使用远程缓存，请确认仓库支持 application/vnd.buildkit.cacheconfig.v0
  // cache-from = ["type=registry,ref=${REGISTRY}/${IMAGE_NAME}:buildcache"]
  // cache-to   = ["type=registry,ref=${REGISTRY}/${IMAGE_NAME}:buildcache,mode=max"]

  output = ["type=registry"]
}
