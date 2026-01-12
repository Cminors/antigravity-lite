# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache gcc musl-dev

# 复制源码
COPY go.mod ./
RUN go mod download

COPY . .

# 编译
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o antigravity-lite .

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装证书（用于HTTPS请求）
RUN apk --no-cache add ca-certificates tzdata

# 复制二进制文件
COPY --from=builder /app/antigravity-lite .
COPY --from=builder /app/config.yaml .

# 创建数据目录
RUN mkdir -p /app/data

# 暴露端口
EXPOSE 8045

# 设置时区
ENV TZ=Asia/Shanghai

# 运行
CMD ["./antigravity-lite"]
