# modlink-gateway

Go 实现：**推理网关**（`cmd/gateway` → `/mlk/v1`）与 **平台 API**（`cmd/platform` → `/mlk/platform/v1`）。

## 配置

复制 `configs/config.example.yaml` 为 `configs/config.yaml`。密钥勿提交仓库（已在 `.gitignore` 忽略 `configs/config.yaml`）。

## 运行

```bash
export MODLINK_CONFIG=configs/config.yaml
export GOPROXY=https://goproxy.cn,direct   # 可选
go run ./cmd/platform
go run ./cmd/gateway
```

数据库迁移：`migrations/*.sql`。

## 模块代理

若下载依赖超时，可使用 `GOPROXY=https://goproxy.cn,direct`（或贵司私有代理）。
