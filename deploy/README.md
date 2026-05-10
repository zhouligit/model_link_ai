# 部署说明（Docker Compose 一键）

与架构建议一致：**单机用 Compose 起 MySQL + Redis + Gateway + Platform + Nginx**；日后前面加负载均衡时，只要 **多副本 Gateway/Platform 容器 + 共享 MySQL（或 RDS）+ 共享 Redis**，入口仍用 Nginx 或云 SLB。

## 与其它服务共用服务器时的端口约定（已定）

| 端口 | 用途 | 说明 |
|------|------|------|
| **8100** | **模链云对外 HTTP 入口（默认）** | `docker-compose.prod.yml` 将宿主机 **`HTTP_PORT`（默认 8100）** 映射到 **本栈 Nginx 容器 :80**。客户端只需访问 `http://<IP>:8100/mlk/v1`、`…/mlk/platform/v1`。 |
| **8101** | **预留** | 例如今后直连调试某个后端、或第二套反代；当前 compose **未占用**。 |
| **8102** | **预留** | 同上；需要第三个对外端口时再启用（文档与 compose 同步更新）。 |

**容器内** Gateway / Platform 仍监听 **:8080 / :8081**，仅在 Docker 网络内互通，**不映射到宿主机**，避免与机器上其它业务的 8080、3000 等冲突。

**整机已有 Nginx（监听 80/443）时**：不要抢全局端口；在本机 `server {}` 里 **`include`** [`deploy/nginx/host-mlk-proxy.conf`](./nginx/host-mlk-proxy.conf)，把路径 **`^~ /mlk`** 反代到 **`http://127.0.0.1:8100`**（即上面 Docker Nginx）。其它 `location`（其它产品的前后端）互不重叠。

## 一键启动（自带 MySQL）

在仓库根目录：

```bash
cp deploy/docker/config.docker.yaml deploy/docker/config.local.yaml
# 编辑 config.local.yaml：至少修改 jwt.secret、security.bootstrap_admin_emails、upstream

export MODLINK_CONFIG_FILE="$PWD/deploy/docker/config.local.yaml"
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d
```

浏览器访问：`http://<服务器IP>:8100/mlk/health`（默认 `HTTP_PORT=8100`，或经整机 Nginx 反代后省略端口）。  
对外 API：

- 网关：`http://<IP>:8100/mlk/v1`（OpenAI SDK `baseURL` 指向该前缀）
- 平台：`http://<IP>:8100/mlk/platform/v1`

默认宿主机端口 **8100**（`HTTP_PORT`，见 `.env.example`）。

## 环境变量（可选）

复制 `.env.example` 为 `.env` 后按需修改：

| 变量 | 说明 |
|------|------|
| `MYSQL_ROOT_PASSWORD` | 与 `config.local.yaml` 里 DSN 中的账号密码需一致（自建 MySQL 时） |
| `MODLINK_CONFIG_FILE` | 挂载到容器内的业务配置文件绝对路径 |
| `HTTP_PORT` | 模链云 Nginx 宿主机端口，**默认 8100** |
| `IMAGE_TAG` | 镜像标签，默认 `local` |

## 使用百度云 RDS（推荐生产）

1. 在百度云创建 **MySQL**（与 `modlink_cloud` 库、账号权限准备好）。  
2. 从 `docker-compose.prod.yml` **删除 `mysql` 服务**及 `migrate` 服务（或保留 migrate 仅指向 RDS，一般不推荐在 CI 外对 RDS 自动跑全量 SQL）。  
3. 在 `config.local.yaml` 把 `database.dsn` 改为 RDS 内网地址。  
4. **首次**在能访问 RDS 的机器上执行一次：

   ```bash
   mysql -h <RDS内网> -u<用户> -p modlink_cloud < modlink-gateway/migrations/001_schema.sql
   mysql -h <RDS内网> -u<用户> -p modlink_cloud < modlink-gateway/migrations/002_seed.sql
   ```

5. `docker compose -f docker-compose.prod.yml up -d` 只起 **redis、platform、gateway、nginx**（需自行编辑 compose 去掉 mysql/migrate 依赖与 `depends_on`）。

## HTTPS

在 `deploy/nginx/` 增加 `443`、`ssl_certificate` / `ssl_certificate_key`，或将 **80 仅反代到本机**，前面再接 **云负载均衡 + 证书**。

## 横向扩展（简要）

- **无状态**：`gateway`、`platform` 副本数 `docker compose up -d --scale gateway=3 --scale platform=2`（需把 **Nginx upstream** 改为多容器：Compose **DNS 同一 service 名会解析到多个 IP**，或使用云 LB 指向多节点）。  
- **状态**：MySQL / RDS、Redis 保持 **单集群**；JWT `secret`、配置文件在所有副本 **一致**。

## 前端静态资源（可选）

当前 compose **未内置** Vue 构建产物。构建方式：

```bash
cd modlink-cloud-web && npm ci && npm run build
# 将 dist 挂到 Nginx 或 CDN；API 仍指向同一域名下的 /mlk/platform/v1
```

可在 `deploy/nginx/modlink.conf` 取消注释静态站 `location` 段落并挂载卷（需要时再开 compose profile）。
