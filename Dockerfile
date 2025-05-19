# syntax=docker/dockerfile:1.4
# docker buildx build \
#   --platform linux/amd64,linux/arm64,linux/s390x \
#   -t ecsagent:latest \
#   .
ARG TARGETOS
ARG TARGETARCH
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

WORKDIR /app
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -o ecsagent client.go

FROM alpine:3.21.3

RUN apk update && \
    apk add --no-cache \
      fontconfig \
      ttf-noto-cjk && \
    fc-cache -f
WORKDIR /app
COPY --from=builder /app/ecsagent .
RUN cat << 'EOF' > /entrypoint.sh
#!/bin/sh
[ -z "$token" ] && printf "主控Token：" && read token
[ -z "$host" ] && printf "主控IPV4/域名：" && read host
[ -z "$api_port" ] && printf "主控API端口：" && read api_port
[ -z "$grpc_port" ] && printf "主控gRPC端口：" && read grpc_port
exec /app/ecsagent \
  --token="$token" \
  --host="$host" \
  --api-port="$api_port" \
  --grpc-port="$grpc_port"
EOF
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
