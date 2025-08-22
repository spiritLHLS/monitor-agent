# syntax=docker/dockerfile:1.4
FROM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY . .

RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -a -o ecsagent client.go

FROM alpine:3.23

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
    echo 'exec_args="-token $token -host $host -api-port $api_port -grpc-port $grpc_port"' >> /entrypoint.sh && \
    echo '[ -n "$task_flag" ] && exec_args="$exec_args -task-flag $task_flag"' >> /entrypoint.sh && \
    echo 'exec /app/ecsagent $exec_args' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]