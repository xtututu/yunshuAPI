# Go 后端构建阶段
FROM golang:alpine AS backend-builder
ENV GO111MODULE=on
ENV CGO_ENABLED=0
# 使用国内 Go 代理加速依赖下载
ENV GOPROXY=https://goproxy.cn,direct

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux}
ENV GOARCH=${TARGETARCH:-amd64}

WORKDIR /build

# 复制 Go 依赖配置并下载（利用 Docker 缓存层）
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码和必要文件（.dockerignore 已配置排除不需要的文件）
# 使用 COPY . . 可以提高首次构建速度，避免过多 COPY 命令的开销
COPY . .

# 编译 Go 程序（注入版本信息）
# 注意：如需使用 BuildKit 缓存加速，请使用 docker buildx build 或设置 DOCKER_BUILDKIT=1
# 然后将下面的 RUN 替换为：
# RUN --mount=type=cache,target=/go/pkg/mod \
#     --mount=type=cache,target=/root/.cache/go-build \
#     go build -ldflags "-s -w -X 'yunshuAPI/common.Version=$(cat VERSION)'" -o new-api
RUN go build -ldflags "-s -w -X 'yunshuAPI/common.Version=$(cat VERSION)'" -o new-api


# 最终运行阶段
FROM alpine

# 使用国内 Alpine 镜像源加速系统依赖安装
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要系统依赖
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata \
    && update-ca-certificates

# 复制编译好的 Go 程序
COPY --from=backend-builder /build/new-api /

# 复制docs目录到容器中
COPY --from=backend-builder /build/docs /docs

# 暴露应用端口
EXPOSE 3000

# 设置工作目录
WORKDIR /data

# 启动应用
ENTRYPOINT ["/new-api"]