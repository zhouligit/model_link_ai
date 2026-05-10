# ModLinkCloud（模链云）—  monorepo 工作区

本目录包含 **文档**、**双后端进程（Gateway + Platform）**、**用户门户 Web**、**管理后台 Web**。各服务 **分目录** 独立开发与部署。

## 目录结构

| 路径 | 说明 |
|------|------|
| `doc/` | PRD、API、MySQL、技术方案等 |
| `modlink-gateway/` | Go：**推理网关**（`/mlk/v1`）与 **平台 API**（`/mlk/platform/v1`），两个 `cmd` 入口 |
| `modlink-cloud-web/` | Vue 3 + Vite + TS：用户门户（仅调 Platform） |
| `modlink-cloud-admin/` | Vue 3 + Vite + TS：管理端（仅调 Platform） |
| `docker-compose.yml` | 仅依赖：**MySQL + Redis**（本地开发库） |
| **`docker-compose.prod.yml`** | **一键生产栈**：MySQL + Redis + **Gateway + Platform + Nginx 反代**（见 `deploy/README.md`） |
| `deploy/` | Nginx 配置、`config.docker.yaml` 模板、迁移脚本说明 |

## Docker 一键部署（推荐上云）

```bash
cp deploy/docker/config.docker.yaml deploy/docker/config.local.yaml
# 编辑 config.local.yaml（jwt.secret、管理员邮箱、upstream 等）
export MODLINK_CONFIG_FILE=$PWD/deploy/docker/config.local.yaml
docker compose -f docker-compose.prod.yml up -d --build
```

详见 **`deploy/README.md`**（含 **百度云 RDS**、HTTPS、扩副本说明）。

**与其它服务共机**：对外默认 **`HTTP_PORT=8100`**（模链云独占该约定端口段起点）；整机 Nginx 可用 `deploy/nginx/host-mlk-proxy.conf` 把 **`/mlk`** 反代到 `127.0.0.1:8100`，与其它业务的 `location` 隔离。

---

## 后端快速启动（本机开发，非 Docker 全栈）

1. **启动 MySQL**（与本仓库 `docker-compose.yml` 一致的用户库名）：

   ```bash
   docker compose up -d mysql
   ```

2. **初始化库表**（一次性）：

   ```bash
   mysql -h127.0.0.1 -uroot -proot modlink_cloud < modlink-gateway/migrations/001_schema.sql
   mysql -h127.0.0.1 -uroot -proot modlink_cloud < modlink-gateway/migrations/002_seed.sql
   ```

3. **配置**：复制 `modlink-gateway/configs/config.example.yaml` 为 `configs/config.yaml`，修改 `database.dsn`、`jwt.secret`、`upstream.mode`（`mock` / `openrouter`）等。

4. **依赖与编译**（若访问 proxy.golang.org 较慢，可使用国内镜像）：

   ```bash
   cd modlink-gateway
   export GOPROXY=https://goproxy.cn,direct
   go mod tidy
   go run ./cmd/platform
   go run ./cmd/gateway
   ```

   默认监听：**Platform `:8081`**，**Gateway `:8080`**（见配置文件）。

5. **生产入口**：Nginx `location ^~ /mlk/` 反代至同一主机上的两个监听端口，或由单一入口进程合并路由（当前实现为 **两个独立 HTTP 服务**）。

## 前端

```bash
cd modlink-cloud-web && npm install && npm run dev
cd modlink-cloud-admin && npm install && npm run dev
```

开发环境通过 Vite **proxy** 将 `/mlk` 转到本地 Platform（见各目录 `vite.config.ts`）。

## 外部集成说明（mock / 真接）

- **推理上游**：`configs/config.yaml` 中 `upstream.mode`：`mock` 返回固定 SSE/JSON；`openrouter` 转发至渠道 `base_url`，密钥优先读数据库 `channels.api_key_cipher`（`plain:base64` 格式），否则回退 `upstream.openrouter_api_key`。
- **支付**：`payment.mode: mock` 时，充值下单后调用 Platform **`POST /mlk/platform/v1/payment/mock/complete`**（Body：`{"order_id":"..."}`）模拟入账。
- **短信**：接口占位；`sms.mode: mock`。

---

详细契约仍以 `doc/模链云-API接口文档.md` 与 `doc/openapi/` 为准；实现持续对齐中。
