## 前置准备

```shell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH="$PATH:$(go env GOPATH)/bin"
source ~/.zshrc
which protoc-gen-go
which protoc-gen-go-grpc

```

## Windows

```shell
protoc --go_out=. .\client.proto
```

```shell
protoc --go-grpc_out=. .\client.proto
```

## MacOS

```shell
protoc --go_out=. ./client.proto
protoc --go-grpc_out=. ./client.proto
```