# 仅保留 Go 后端构建和最终运行阶段，跳过前端构建
FROM golang:alpine AS builder2
ENV GO111MODULE=on
ENV CGO_ENABLED=0
# 使用国内 Go 代理加速依赖下载
ENV GOPROXY=https://goproxy.cn,direct

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux}
ENV GOARCH=${TARGETARCH:-amd64}

WORKDIR /build

# 复制 Go 依赖配置并下载
ADD go.mod go.sum ./
RUN go mod download

# 复制所有项目文件（包括本地预构建的 web/dist）
COPY . .

# 编译 Go 程序（注入版本信息）
RUN go build -ldflags "-s -w -X 'yishangyunApi/common.Version=$(cat VERSION)'" -o new-api


# 最终运行阶段
FROM alpine

# 使用国内 Alpine 镜像源加速系统依赖安装
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要系统依赖
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata \
    && update-ca-certificates

# 复制编译好的 Go 程序
COPY --from=builder2 /build/new-api /

# 复制docs目录到容器中
COPY --from=builder2 /build/docs /docs

# 暴露应用端口
EXPOSE 3000

# 设置工作目录
WORKDIR /data

# 启动应用
ENTRYPOINT ["/new-api"]