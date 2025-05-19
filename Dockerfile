# syntax=docker/dockerfile:1.4
# 构建命令示例:
# docker buildx build \
#   --platform linux/amd64,linux/arm64,linux/s390x \
#   -t ecsagent:latest \
#   .

FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod download && go mod tidy

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -a -o ecsagent client.go

FROM alpine:3.21

RUN apk update && \
    apk add --no-cache \
      fontconfig \
      font-noto-cjk && \
    fc-cache -f

WORKDIR /app

COPY --from=builder /app/ecsagent .

RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo '[ -z "$token" ] && printf "主控Token：" && read token' >> /entrypoint.sh && \
    echo '[ -z "$host" ] && printf "主控IPV4/域名：" && read host' >> /entrypoint.sh && \
    echo '[ -z "$api_port" ] && printf "主控API端口：" && read api_port' >> /entrypoint.sh && \
    echo '[ -z "$grpc_port" ] && printf "主控gRPC端口：" && read grpc_port' >> /entrypoint.sh && \
    echo 'exec /app/ecsagent \\' >> /entrypoint.sh && \
    echo '  --token="$token" \\' >> /entrypoint.sh && \
    echo '  --host="$host" \\' >> /entrypoint.sh && \
    echo '  --api-port="$api_port" \\' >> /entrypoint.sh && \
    echo '  --grpc-port="$grpc_port"' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
