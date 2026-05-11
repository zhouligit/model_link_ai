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

## 国内 / 百度云：拉取 Docker Hub 超时（`registry-1.docker.io` i/o timeout）

**现象**：`failed to resolve reference "docker.io/library/nginx:..."` 或 `dial tcp ...:443: i/o timeout`。

**做法一（推荐，立刻可用）**：使用仓库自带的 **国内镜像前缀** 覆盖 compose：

```bash
export MODLINK_CONFIG_FILE=$PWD/deploy/docker/config.local.yaml
docker compose -f docker-compose.prod.yml -f docker-compose.cn-mirror.yml up -d --build
```

**做法二（全机加速）**：配置 Docker 守护进程镜像加速后 **重启 Docker**，再执行原来的 `docker compose -f docker-compose.prod.yml ...`：

```bash
sudo mkdir -p /etc/docker
sudo cp deploy/docker/daemon.json.example /etc/docker/daemon.json
# 按需编辑 mirrors；百度云可改用控制台提供的「容器镜像服务」加速地址
sudo systemctl restart docker
```

若出现 **`buildx isn't installed`**：Compose 仍可用默认 `docker build` 构建，一般可忽略；需要 Bake 时再装 **`docker-buildx-plugin`**（`apt-cache search buildx`）。

**验证镜像站是否通**：

```bash
docker pull docker.m.daocloud.io/library/nginx:1.26-alpine
```

若 DaoCloud 也失败，可换 **阿里云 ACR 个人版** 的 Docker Hub 代理地址（在阿里云控制台复制），写进 `daemon.json` 的 `registry-mirrors`，或把 `docker-compose.cn-mirror.yml` 里的前缀整体替换为云厂商文档给出的前缀。

---

## 故障：`migrate` 容器 `exit 1`

先看日志：

```bash
docker logs model_link_ai-migrate-1 --tail=80
```

常见原因：

1. **数据卷里的 MySQL 仍是旧 root 密码**，而 migrate 用的是 compose 里新的默认 **`123456`** → 认证失败。处理：在 **`.env`** 里设 **`MYSQL_ROOT_PASSWORD=` 与当前数据卷一致**（例如仍是 `root`），或 **`docker compose ... down -v` 清空卷** 后重起（**会删库**）。  
2. **001 曾执行一半又重跑**：表已存在导致 `CREATE TABLE` 报错 → 开发环境可 **`down -v`** 重来。  
3. **网络**：migrate 连不上 `mysql:3306`（极少见，若 `depends_on: healthy` 已满足一般不是）。

仓库已加强 **`deploy/docker/migrate.sh`**（显式 `-p`、失败时打印提示），更新代码后重新 `up` 即可。

---

## 故障：`platform` / `gateway` / `nginx` 一直 `Restarting`，8100 连不上

**最常见原因**：`config.local.yaml` 里 **`database.dsn` 写成了 `127.0.0.1` 或宿主机 IP**。  
在 Compose 网络里，MySQL 的服务名是 **`mysql`**，DSN 必须是 **`tcp(mysql:3306)/...`**（与 `deploy/docker/config.docker.yaml` 一致）。

先看退出原因：

```bash
docker logs model_link_ai-platform-1 --tail=40
docker logs model_link_ai-gateway-1 --tail=40
```

若出现 **`database ping: ... connection refused`** 或 **`dial tcp 127.0.0.1:3306`**，请改 DSN 后重启：

```bash
nano deploy/docker/config.local.yaml   # database.dsn 中 host 改为 mysql
docker compose -f docker-compose.prod.yml -f docker-compose.cn-mirror.yml up -d
```

**次常见原因**：`jwt.secret` 长度不足 16（`config.Load` 会直接失败），日志里会有 `jwt.secret` 相关报错。

---

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
| `MYSQL_ROOT_PASSWORD` | 默认 **`123456`**（仓库约定，仅开发/首次部署）；生产在 `.env` 覆盖并 **ALTER USER** 改库；**已有数据卷时改默认值不会自动改库里 root 密码** |
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
