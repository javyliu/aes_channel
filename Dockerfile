# 使用官方 Go 镜像作为构建环境
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制项目源码
COPY . .

# 构建可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -o aes_channel ./cmd/main.go

# 使用更小的基础镜像运行
FROM alpine:latest
# 安装必要的证书（如果应用需要 HTTPS 或其他 TLS 连接）
RUN apk --no-cache add ca-certificates
WORKDIR /app

# 复制编译好的二进制文件
COPY --from=builder /app/aes_channel .

# 暴露默认端口
EXPOSE 18305

# 启动服务（可通过环境变量或参数覆盖）
ENTRYPOINT ["./aes_channel"]