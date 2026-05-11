# 国内拉 Docker Hub 超时：compose 传入 BASE_GO_IMAGE / BASE_RUN_IMAGE（见 docker-compose.cn-mirror.yml）
# 注意：凡用于 FROM 的 ARG 必须在「第一个 FROM 之前」全部声明，否则后续 FROM ${...} 为空（Docker 规则）。
ARG BASE_GO_IMAGE=golang:1.22-bookworm
ARG BASE_RUN_IMAGE=debian:bookworm-slim

FROM ${BASE_GO_IMAGE} AS build

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/platform ./cmd/platform \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway

# 运行阶段：仅证书与二进制（无 shell 可选 distroless；此处保留轻量 shell 便于排障）
FROM ${BASE_RUN_IMAGE}

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/platform /app/platform
COPY --from=build /out/gateway /app/gateway

# 默认无 CMD，由 compose 指定运行 platform 或 gateway
EXPOSE 8080 8081
