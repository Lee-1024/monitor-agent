# ============================================
# Monitor Agent - 多阶段构建
# ============================================

# 阶段一：构建
FROM golang:1.24-alpine AS builder

WORKDIR /build

# 安装构建依赖（如需要 cgo 可取消注释）
# RUN apk add --no-cache gcc musl-dev

# 先复制依赖文件，利用缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o monitor-agent .

# 阶段二：运行
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# 使用非 root 用户运行（采集需读 /proc 等，可按需改为 root 或指定 uid）
RUN adduser -D -g "" appuser
WORKDIR /app

COPY --from=builder /build/monitor-agent .
# 可选：默认配置，运行时可通过挂载覆盖
# COPY agent-config.yaml ./

USER appuser

# 默认配置文件路径，可通过 -v 挂载或环境变量覆盖
ENV CONFIG_PATH=/app/agent-config.yaml

EXPOSE 50052

ENTRYPOINT ["./monitor-agent"]
