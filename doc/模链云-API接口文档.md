# 模链云（ModLinkCloud）API 接口文档

| 属性 | 内容 |
|------|------|
| **文档版本** | v1.5 |
| **更新日期** | 2026-05-10 |
| **说明** | 本文覆盖 **全量需求对应接口**（不按阶段拆分）；实现时可同一迭代交付。 |
| **关联** | [PRD](./模链云-产品需求文档-PRD.md)、[前端技术选型](./模链云-前端技术选型说明.md) |
| **OpenAPI** | [doc/openapi/](./openapi/README.md)（`modlink-gateway.openapi.yaml` + `modlink-platform.openapi.yaml`） |
| **JSON 示例** | **附录 A**：典型接口 **请求体 / 响应 data** 的 JSON；通用包裹见 **§2.3**；逐路径字段级定义以 **OpenAPI `components.schemas`** 为准（可继续细化） |
| **详述版（表格+HTTP）** | [模链云-API接口文档-详述.md](./模链云-API接口文档-详述.md)：按「参数表 / JSON 示例 / HTTP 示例 / data 表 / 响应 JSON」编排 |

---

## 背景约定（请勿遗忘）

| 项 | 已定结论 |
|----|----------|
| **上游** | 首期 **OpenRouter**；网关兼容 OpenAI 路径形态 |
| **路径命名空间** | 与其它系统 **共用主机** 时，统一 **`/mlk`** 前缀（见 §1）；便于 Nginx `location ^~ /mlk/` 指向本服务 |
| **共机端口（部署约定）** | 与其它服务共用服务器时，模链云对外 HTTP **默认从 `8100` 起**（`8101`、`8102` 预留）；路径仍为 **`/mlk/...`**；详见仓库 **`deploy/README.md`** |
| **对外推理 Base URL** | `https://api.<备案域名>/mlk/v1`（OpenAI SDK `baseURL`，末尾 **无** 斜杠） |
| **货币与支付** | **人民币**；**微信支付 → 支付宝**；Stripe 待办 |
| **租户** | **企业租户 + 成员** 与个人账户并存 |
| **账务** | 优先真实 `usage`；缺失则估算扣费并标记；单库事务 + 流水 + 幂等 |
| **日志** | 默认 **不落 Prompt 全文**；可用户显式开启调试留存 |
| **前端** | Vue 3 + Vite + TS + Element Plus；双 Web 工程 |
| **后端** | 推荐 **Go**；持久化 **MySQL** |
| **部署** | 单机；**阿里云或腾讯云 Ubuntu**；本地开发 **macOS** |
| **SLA** | 不对客户书面承诺数值 SLA |
| **接口已定 A–H** | **§25**：同域 **`/mlk/v1`** + **`/mlk/platform/v1`**；个人/org **钱包与 Key 严格分账**；**`X-Org-Id`** 覆盖 JWT 默认 org；**Embeddings** 可用 CFG 关闭；管理员高危 **二次验证**；云厂商 **跟备案主体**；**SSE 断开** 尽力 cancel + 按 usage/估算扣费；Key 前缀 **`mk_live_` / `mk_test_`** |

---

## 1. Base URL、路径命名空间与架构划分

### 1.1 为何使用 `/mlk` 前缀

与同机其它应用 **共用域名或入口** 时，需用 **统一、可识别** 的路径前缀区分后端。**本文已定**：所有模链云 HTTP API（推理 + 平台）均挂在 **`/mlk` 命名空间下**。

- **`mlk`**：ModLinK 缩写，短、不易与其它业务冲突。  
- **不推荐**单独占用通用 **`/v1`**（多台服务都爱用 `/v1`，易与友邻路由打架）。  
- **不推荐**采用 **`/v1/mlk`** 形态：与其它服务的 **`/v1/*`** 仍共享第一段，分流时需更细粒度匹配；**`/mlk/...` 一段即可锁定本系统**。

### 1.2 Base URL 表

| 名称 | Base URL（占位域名） | 职责 |
|------|----------------------|------|
| **推理网关** | `https://api.{domain}/mlk/v1` | OpenAI 兼容接口（如 `…/mlk/v1/chat/completions`）；仅平台颁发的 **API Key** 鉴权 |
| **平台业务 API** | `https://api.{domain}/mlk/platform/v1` | 控制台、管理后台、钱包、租户、支付回调等；**JWT**（用户/管理员） |

> **路由分流（示例）**：`location ^~ /mlk/` → 反代至本服务（再由进程按 **`/mlk/v1`** vs **`/mlk/platform/v1`** 分网关 / 平台端口或同一监听）。其它业务可使用 `/api/other/` 等 **不同首段**，互不干扰。

**本地开发（Mac）示例**：`https://localhost:8443/mlk/v1`、`https://localhost:8443/mlk/platform/v1`。

**OpenAI / OpenRouter SDK**：将 **`baseURL`** 设为 **`https://api.{domain}/mlk/v1`**（与官方「指向自定义网关」用法一致）。

---

## 2. 通用约定

### 2.1 传输与安全

- 生产环境 **HTTPS**（TLS 1.2+）。  
- `Content-Type: application/json`（推理网关流式见各节）。  
- 时间戳：**ISO 8601** 字符串，时区 **Asia/Shanghai** 或与客户端约定 UTC（文档示例用 **UTC+8** 展示）。

### 2.2 鉴权方式

| 场景 | Header | 说明 |
|------|--------|------|
| **推理网关** | `Authorization: Bearer <platform_api_key>` | 前缀 **`mk_live_`**（生产）或 **`mk_test_`**（测试）；**禁止**使用上游 OpenRouter Key |
| **平台 API（用户/管理员）** | `Authorization: Bearer <access_token>` | JWT；管理员与用户 **同一颁发机制**，通过 `role`/`permissions` 区分 |
| **支付回调** | 各支付渠道 **签名头 /  body** | **不经 JWT**；验签 + 幂等处理 |

**组织上下文（已定）**：JWT 携带默认 **`current_org_id`**；请求可带 **`X-Org-Id`**。若 Header 存在且用户为该组织成员，**以 `X-Org-Id` 为准**；否则使用 JWT 默认。

可选（OpenRouter 兼容对外展示）：`HTTP-Referer`、`X-Title` 等由网关 **按渠道配置覆盖**，客户端可选。

### 2.3 通用响应包裹（平台 API）

完整 JSON 示例见 **附录 A**（注册、登录、Key、钱包、充值、用量、Chat 等）。

成功：

```json
{
  "code": 0,
  "message": "ok",
  "data": { },
  "request_id": "req_xxx"
}
```

业务错误（示例）：

```json
{
  "code": 40001,
  "message": "INSUFFICIENT_BALANCE",
  "detail": { "balance_cents": 0 },
  "request_id": "req_xxx"
}
```

- **`code`**：`0` 成功；非 `0` 为业务错误码（见 §2.5）。  
- **`request_id`**：全链路追踪 ID；与网关推理侧 **同源字段名** 便于排障。

### 2.4 分页（列表类）

查询参数：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `page` | int | 1 | ≥1 |
| `page_size` | int | 20 | 上限如 100 |

响应 `data`：

```json
{
  "items": [],
  "page": 1,
  "page_size": 20,
  "total": 100
}
```

游标分页如需高性能日志检索，可额外提供 `cursor` 接口变体（实现可选）。

### 2.5 业务错误码（平台 API，节选）

| code | 语义 |
|------|------|
| 0 | 成功 |
| 40001 | 参数错误 |
| 40101 | 未登录或 Token 失效 |
| 40301 | 无权限 |
| 40302 | 资源不属于当前租户 |
| 40401 | 资源不存在 |
| 40901 | 冲突（重复提交等） |
| 42901 | 触发限流 |
| 40201 | 余额不足 |
| 50301 | 维护模式 / 上游不可用 |

完整码表由实现维护并在附录导出。

### 2.6 推理网关错误（OpenAI 兼容形态）

与 OpenAI 类似：**HTTP 状态码** + JSON body（含 `error.message`、`error.type`），并包含 **`request_id`** 扩展字段（建议置于 body 根级或与 OpenRouter 对齐）。**禁止**在错误体中返回上游密钥。

---

## 3. 推理网关（OpenAI / OpenRouter 兼容）

**Base**：`https://api.{domain}/mlk/v1`  
**鉴权**：`Authorization: Bearer <platform_api_key>`

### 3.1 Chat Completions

**POST** `/mlk/v1/chat/completions`

**请求体**：与 OpenAI Chat Completions 一致（`model`, `messages`, `stream`, `temperature`, …）。`model` 为平台允许的上游模型 ID（如 `openai/gpt-4o`）。

**非流式响应**：标准 OpenAI completion JSON；含 `usage` 时用于计费。

**流式响应**：`Content-Type: text/event-stream`（SSE）；最后一帧或独立 chunk 携带 `usage`（若上游提供）。

**幂等（可选增强）**：请求头 `Idempotency-Key`（UUID），用于客户端重试防重复扣费（实现可选，推荐）。

### 3.2 模型列表

**GET** `/mlk/v1/models`

**鉴权**：同上 API Key。

**响应**：OpenAI 兼容 `object/list`，仅返回 **平台已启用** 且 **对当前 Key 可见** 的模型（受租户/套餐/路由策略过滤）。

### 3.3 获取单个模型

**GET** `/mlk/v1/models/{model_id}`

`model_id` 需 URL 编码（如含 `/`）。

### 3.4 Embeddings（可选但建议实现）

**POST** `/mlk/v1/embeddings`

与 OpenAI Embeddings 兼容；计费按 token 规则在平台配置（若一期关闭，可返回 `503` 或 `feature_not_enabled`）。

### 3.5 （预留）其他 OpenAI 兼容路径

若未来扩展 **Images / Audio**，路径保持 **`/mlk/v1/...`** 前缀；未上线前文档可标注 **not implemented**。

---

## 4. 平台 API — 认证与注册

**Base path**：`/mlk/platform/v1`

### 4.1 注册

**POST** `/mlk/platform/v1/auth/register`

**Body**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `email` | string | 与手机二选一 | 邮箱 |
| `phone` | string | 与邮箱二选一 | 手机 |
| `password` | string | 是 | 密码强度由 CFG 约束 |
| `invite_code` | string | 否 | 邀请码（CFG 开启时必填） |

**响应 `data`**

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | string | |
| `access_token` | string | JWT |
| `refresh_token` | string | |
| `expires_in` | int | 秒 |

### 4.2 登录

**POST** `/mlk/platform/v1/auth/login`

**Body**：`email` 或 `phone` + `password`（或验证码登录扩展）。

**响应**：同注册 `tokens` 结构。

### 4.3 刷新 Token

**POST** `/mlk/platform/v1/auth/refresh`

**Body**：`refresh_token`

**响应**：新的 `access_token` / `refresh_token`。

### 4.4 登出

**POST** `/mlk/platform/v1/auth/logout`

**Header**：Bearer JWT。服务端可使 refresh_token 失效（黑名单或版本号）。

### 4.5 找回密码

**POST** `/mlk/platform/v1/auth/password/reset-request` — 发送验证码/邮件  

**POST** `/mlk/platform/v1/auth/password/reset-confirm` — 验证码 + 新密码  

（具体字段与短信/邮件服务商对接。）

### 4.6 当前用户上下文

**GET** `/mlk/platform/v1/auth/me`

**响应 `data`**

| 字段 | 类型 | 说明 |
|------|------|------|
| `user` | object | id, email, phone, display_name, avatar_url |
| `roles` | string[] | 如 `user`, `org_owner`, `admin` |
| `current_org_id` | string \| null | 当前选中租户 |

---

## 5. 平台 API — 个人资料

**GET** `/mlk/platform/v1/users/me`  

**PATCH** `/mlk/platform/v1/users/me`

可编辑：`display_name`, `avatar_url` 等；敏感字段（手机/邮箱）走验证流程另设接口。

---

## 6. 平台 API — 企业租户与成员

### 6.1 创建租户

**POST** `/mlk/platform/v1/orgs`

**Body**：`name`, `slug`（可选，唯一）

**响应**：`org` 对象（含 `id`, `name`, `role`）。

### 6.2 租户列表（当前用户参与）

**GET** `/mlk/platform/v1/orgs`

### 6.3 切换当前租户（会话上下文）

**POST** `/mlk/platform/v1/orgs/{org_id}/switch`

后续请求头可带 `X-Org-Id: {org_id}` **或** 依赖服务端 session 中 `current_org_id`（二选一，建议 **显式 Header** 避免歧义）。

### 6.4 租户详情

**GET** `/mlk/platform/v1/orgs/{org_id}`

### 6.5 更新租户

**PATCH** `/mlk/platform/v1/orgs/{org_id}`  

（所有者/管理员；字段如 `name`。）

### 6.6 成员列表

**GET** `/mlk/platform/v1/orgs/{org_id}/members`

### 6.7 邀请成员

**POST** `/mlk/platform/v1/orgs/{org_id}/invitations`

**Body**：`email` 或 `phone`, `role`（`admin` | `member`）

**响应**：`invitation_id`, `expires_at`

### 6.8 接受邀请

**POST** `/mlk/platform/v1/orgs/invitations/{token}/accept`

### 6.9 变更成员角色 / 移除成员

**PATCH** `/mlk/platform/v1/orgs/{org_id}/members/{user_id}`  

**DELETE** `/mlk/platform/v1/orgs/{org_id}/members/{user_id}`

### 6.10 转让所有者（高危）

**POST** `/mlk/platform/v1/orgs/{org_id}/transfer-owner`

**Body**：`new_owner_user_id`，二次验证（密码/MFA）由实现定义。

---

## 7. 平台 API — API Key

创建 Key 时必须指定 **作用域（已定）**：**`personal`**（仅扣 **个人钱包**）或 **`org` + `org_id`**（仅扣 **该组织钱包**）；**禁止**一把 Key 跨个人与组织混扣。租户场景下创建列表/创建接口通过 **`X-Org-Id`** 或 Body `org_id` 与 JWT 约束一致性。

### 7.1 列表

**GET** `/mlk/platform/v1/api-keys`

**Query**：`org_id`（可选）、`page`, `page_size`

### 7.2 创建

**POST** `/mlk/platform/v1/api-keys`

**Body**：`name`, `org_id`（可选）, `expires_at`（可选）, `ip_allowlist`（可选，CIDR 列表）

**响应**：**仅本次返回完整 `secret`**（如 `mk_live_xxxx`），之后仅展示脱敏前缀。

### 7.3 更新

**PATCH** `/mlk/platform/v1/api-keys/{key_id}`  

可更新：`name`, `enabled`, `expires_at`, `ip_allowlist`

### 7.4 删除 / 禁用

**DELETE** `/mlk/platform/v1/api-keys/{key_id}`  

**POST** `/mlk/platform/v1/api-keys/{key_id}/disable`  

**POST** `/mlk/platform/v1/api-keys/{key_id}/enable`

---

## 8. 平台 API — 钱包与充值

### 8.1 查询余额

**GET** `/mlk/platform/v1/wallet`

**Query**：`org_id`（可选；不传为个人账户）

**响应 `data`**

| 字段 | 类型 | 说明 |
|------|------|------|
| `balance_cents` | int64 | 余额（分） |
| `currency` | string | `CNY` |
| `credit_status` | string | 如 `active`, `frozen` |

### 8.2 创建充值订单

**POST** `/mlk/platform/v1/orders/recharge`

**Body**

| 字段 | 类型 | 说明 |
|------|------|------|
| `amount_cents` | int | 充值金额（分），≥最小充值额 |
| `channel` | string | `wechat` \| `alipay` |
| `org_id` | string | 可选；租户充值 |

**响应 `data`**

| 字段 | 类型 | 说明 |
|------|------|------|
| `order_id` | string | |
| `payment_params` | object | 微信 JSAPI/APP 参数或支付宝 orderString 等，随渠道变化 |

### 8.3 订单详情（用户侧）

**GET** `/mlk/platform/v1/orders/{order_id}`

### 8.4 用户订单列表

**GET** `/mlk/platform/v1/orders`

**Query**：`type=recharge`, `status`, `page`, `page_size`

---

## 9. 支付回调（服务端对支付渠道）

**不经用户 JWT**；由支付平台服务器访问；URL 在商户平台配置。

| 方法 | 路径 | 说明 |
|------|------|------|
| **POST** | `/mlk/platform/v1/payment/notify/wechat` | 微信支付异步通知；验签 → 幂等入账 |
| **POST** | `/mlk/platform/v1/payment/notify/alipay` | 支付宝异步通知 |

**响应**：按各渠道要求返回 **纯文本 success / XML / JSON**（实现严格遵循微信/支付宝文档）。

**幂等键**：平台订单号 `out_trade_no` / 商户订单号与本地 `order_id` 映射。

---

## 10. 平台 API — 用量、账单、流水

### 10.1 用量汇总

**GET** `/mlk/platform/v1/usage/summary`

**Query**：`org_id`, `from`, `to`, `granularity=day|hour`

**响应**：按模型、按日聚合的调用次数、输入/输出 token、费用（分）。

### 10.2 账单明细

**GET** `/mlk/platform/v1/bills`

**Query**：`org_id`, `from`, `to`, `model`, `page`, `page_size`

**单项字段示例**：`request_id`, `time`, `model`, `input_tokens`, `output_tokens`, `cost_cents`, `billing_type`（`actual` | `estimated`）

### 10.3 钱包流水

**GET** `/mlk/platform/v1/wallet/transactions`

充值、扣费、调账、退款等统一流水。

---

## 11. 平台 API — 开放文档元数据（供控制台渲染）

**GET** `/mlk/platform/v1/openapi/info`

**响应**：`gateway_base_url`, `doc_version`, `features[]`（如是否开启 embeddings）。

（完整 OpenAPI JSON 可提供 **GET** `/mlk/platform/v1/openapi.json` 可选。）

---

## 12. 管理后台 API — 前缀与权限

**建议使用前缀**：`/mlk/platform/v1/admin/*`  

**鉴权**：JWT + `role` 含 `admin` / `super_admin` / 细粒度权限。

下列接口均为 **管理侧**；普通用户 **403**。

---

## 13. 管理 — 用户与租户

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/users` | 搜索用户；Query：`keyword`, `status`, `page` |
| **GET** | `/mlk/platform/v1/admin/users/{user_id}` | 详情 |
| **POST** | `/mlk/platform/v1/admin/users/{user_id}/disable` | 禁用 |
| **POST** | `/mlk/platform/v1/admin/users/{user_id}/enable` | 启用 |
| **GET** | `/mlk/platform/v1/admin/orgs` | 租户列表 |
| **GET** | `/mlk/platform/v1/admin/orgs/{org_id}` | 租户详情 |
| **POST** | `/mlk/platform/v1/admin/orgs/{org_id}/disable` | 禁用租户 |
| **GET** | `/mlk/platform/v1/admin/orgs/{org_id}/members` | 成员列表 |
| **POST** | `/mlk/platform/v1/admin/orgs/{org_id}/reset-owner` | 重置管理员（高危） |

---

## 14. 管理 — 上游渠道

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/channels` | 渠道列表 |
| **POST** | `/mlk/platform/v1/admin/channels` | 创建（`name`, `type`, `base_url`, `api_key` 等） |
| **GET** | `/mlk/platform/v1/admin/channels/{channel_id}` | 详情（密钥脱敏） |
| **PATCH** | `/mlk/platform/v1/admin/channels/{channel_id}` | 更新 |
| **POST** | `/mlk/platform/v1/admin/channels/{channel_id}/test` | 连通性探测 |
| **POST** | `/mlk/platform/v1/admin/channels/{channel_id}/circuit-break` | 熔断开关 |

---

## 15. 管理 — 模型目录与路由

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/models` | 平台模型列表 |
| **POST** | `/mlk/platform/v1/admin/models` | 上架模型 |
| **PATCH** | `/mlk/platform/v1/admin/models/{model_id}` | 启用/禁用、显示名、计价绑定 |
| **POST** | `/mlk/platform/v1/admin/models/import` | 导入 JSON/CSV |
| **POST** | `/mlk/platform/v1/admin/models/sync` | 自上游拉取（若实现） |
| **GET** | `/mlk/platform/v1/admin/routes` | 路由映射列表 |
| **PUT** | `/mlk/platform/v1/admin/routes/{id}` | 客户端 model → 渠道 + 上游 model id |
| **GET** | `/mlk/platform/v1/admin/policies/model-access` | 按用户/租户/套餐的模型可见策略 |
| **PUT** | `/mlk/platform/v1/admin/policies/model-access` | 更新策略 |

---

## 16. 管理 — 计价与套餐

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/pricing/models` | 模型单价（输入/输出每千 token 或每百万 token，单位：分） |
| **PUT** | `/mlk/platform/v1/admin/pricing/models/{model_id}` | 设置价格 |
| **GET** | `/mlk/platform/v1/admin/plans` | 套餐列表 |
| **POST** | `/mlk/platform/v1/admin/plans` | 创建套餐（包月、赠送额度等） |
| **PATCH** | `/mlk/platform/v1/admin/plans/{plan_id}` | 更新 |

---

## 17. 管理 — 订单与财务

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/orders` | 全站订单筛选 |
| **GET** | `/mlk/platform/v1/admin/orders/{order_id}` | 详情 |
| **POST** | `/mlk/platform/v1/admin/orders/{order_id}/manual-complete` | 手工补单（审计） |
| **POST** | `/mlk/platform/v1/admin/adjustments` | 调账（**Body**：`user_id`/`org_id`, `amount_cents`, `reason`） |
| **GET** | `/mlk/platform/v1/admin/reconciliation/export` | 导出对账 CSV |

---

## 18. 管理 — 风控

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/risk/rate-limits` | 限流规则 |
| **PUT** | `/mlk/platform/v1/admin/risk/rate-limits` | 全局 / Key / 用户 维度 |
| **GET** | `/mlk/platform/v1/admin/risk/quotas` | 配额规则 |
| **PUT** | `/mlk/platform/v1/admin/risk/quotas` | |
| **GET** | `/mlk/platform/v1/admin/risk/blacklist` | |
| **POST** | `/mlk/platform/v1/admin/risk/blacklist` | 添加 IP / user_id / key_id |
| **DELETE** | `/mlk/platform/v1/admin/risk/blacklist/{entry_id}` | |
| **GET** | `/mlk/platform/v1/admin/risk/whitelist` | 同上结构 |

---

## 19. 管理 — 报表与大屏

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/dashboard/summary` | KPI：调用量、token、收入估算、错误率 |
| **GET** | `/mlk/platform/v1/admin/dashboard/series` | 时间序列；Query：`metric`, `from`, `to`, `interval` |
| **GET** | `/mlk/platform/v1/admin/dashboard/screen` | **大屏专用**：全屏布局所需聚合（可与 summary 复用） |

---

## 20. 管理 — 系统配置与审计

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/config` | 维护模式、注册开关、最小充值、**gateway_public_base_url** 等 |
| **PATCH** | `/mlk/platform/v1/admin/config` | 更新 CFG |
| **GET** | `/mlk/platform/v1/admin/audit-logs` | 审计日志；Query：操作者、资源类型、时间段 |

---

## 21. 日志查询（管理 / 授权用户）

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/platform/v1/admin/inference-logs` | 推理调用日志（元数据）；Query：`request_id`, `user_id`, `org_id`, `model`, `status`, `from`, `to` |
| **GET** | `/mlk/platform/v1/inference-logs` | 租户用户可查 **本租户** 范围内（权限收敛） |

**隐私**：默认 **无** `prompt`/`completion` 字段；若用户开启调试且策略允许，可增加 query 参数 `include_debug=1`（仅授权角色）。

---

## 22. 运维与健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET** | `/mlk/health` | 进程存活（可不带鉴权）；与其它业务 **`/health`** 区分 |
| **GET** | `/mlk/platform/v1/health` | 业务依赖探测：DB、Redis、上游渠道可选 ping |
| **GET** | `/mlk/ready` | 就绪探针（可选） |

---

## 23. Webhook（可选扩展）

若需向 **客户系统** 推送余额告警、订单结果：

**POST** 客户配置的 URL（出站）：签名头 `X-ModLink-Signature`，body JSON。**本期可为空实现**，仅预留配置接口：

| 方法 | 路径 | 说明 |
|------|------|------|
| **GET/PUT** | `/mlk/platform/v1/orgs/{org_id}/webhooks` | 配置回调 URL 与事件类型 |

---

## 附录 A：典型 JSON 入参、出参示例

下列示例便于联调与文档评审；**字段可为实现微调**，须与 OpenAPI 同步。平台接口均在 **`code===0`** 时使用 **`data`** 承载业务载荷（见 §2.3）。

### A.1 认证 — 注册 `POST .../auth/register`

**请求体（application/json）**

```json
{
  "email": "user@example.com",
  "password": "YourStrongPassword",
  "invite_code": ""
}
```

**成功响应（节选）**

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_abc123",
  "data": {
    "user_id": "usr_001",
    "access_token": "eyJhbGciOi...",
    "refresh_token": "rt_xxx",
    "expires_in": 7200
  }
}
```

### A.2 认证 — 登录 `POST .../auth/login`

**请求体**

```json
{
  "email": "user@example.com",
  "password": "YourStrongPassword"
}
```

**成功响应 `data`**

```json
{
  "user_id": "usr_001",
  "access_token": "eyJhbGciOi...",
  "refresh_token": "rt_xxx",
  "expires_in": 7200
}
```

### A.3 API Key — 创建 `POST .../api-keys`

**请求体**

```json
{
  "name": "prod-backend",
  "scope": "org",
  "org_id": "org_001",
  "expires_at": null,
  "ip_allowlist": ["203.0.113.0/24"]
}
```

**成功响应 `data`（`secret` 仅本次返回）**

```json
{
  "id": "key_001",
  "name": "prod-backend",
  "scope": "org",
  "org_id": "org_001",
  "secret": "mk_live_xxxxxxxxxxxxxxxx",
  "created_at": "2026-05-10T12:00:00+08:00",
  "prefix": "mk_live_xxxx"
}
```

### A.4 钱包 — 查询 `GET .../wallet`

**Query**：`org_id`（可选，查组织钱包）

**成功响应 `data`**

```json
{
  "balance_cents": 150000,
  "currency": "CNY",
  "credit_status": "active"
}
```

### A.5 充值订单 — 创建 `POST .../orders/recharge`

**请求体**

```json
{
  "amount_cents": 10000,
  "channel": "wechat",
  "org_id": "org_001"
}
```

**成功响应 `data`**

```json
{
  "order_id": "ord_001",
  "payment_params": {
    "jsapi_params": {}
  }
}
```

> `payment_params` 结构随微信 / 支付宝 SDK 要求变化，以实现为准。

### A.6 用量汇总 — `GET .../usage/summary`

**成功响应 `data`（示例结构）**

```json
{
  "granularity": "day",
  "series": [
    {
      "date": "2026-05-09",
      "total_calls": 1200,
      "input_tokens": 800000,
      "output_tokens": 300000,
      "cost_cents": 45000,
      "by_model": [
        {
          "model": "openai/gpt-4o",
          "calls": 800,
          "cost_cents": 40000
        }
      ]
    }
  ]
}
```

### A.7 推理网关 — Chat Completions（非流式）`POST .../mlk/v1/chat/completions`

**请求头**：`Authorization: Bearer mk_live_xxx`，`Content-Type: application/json`

**请求体**（与 OpenAI 兼容）

```json
{
  "model": "openai/gpt-4o",
  "messages": [
    { "role": "user", "content": "你好，简要介绍你自己。" }
  ],
  "stream": false,
  "temperature": 0.7
}
```

**响应（application/json，OpenAI 兼容形态；节选）**

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1715320800,
  "model": "openai/gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "……"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 80,
    "total_tokens": 100
  },
  "request_id": "req_infer_001"
}
```

### A.8 推理网关 — 流式说明

`stream: true` 时响应为 **`text/event-stream`**（SSE），非单一 JSON；最后一帧或尾部 chunk 可能含 **`usage`**；详见上游 OpenRouter/OpenAI 文档。

### A.9 推理网关 — 错误（示例）

```json
{
  "error": {
    "message": "Insufficient balance",
    "type": "billing_error",
    "code": "insufficient_balance"
  },
  "request_id": "req_infer_err_001"
}
```

---

## 24. 接口与 PRD 需求追溯（简表）

| PRD 章节 | 接口章节 |
|----------|----------|
| 网关 GW-* | §3 |
| WEB / KEY / PAY / USG | §4–§10 |
| 管理 ADM / CH / MDL / RT / ORD / CFG | §12–§20 |
| RSK | §18 |
| RPT / 大屏 | §19 |
| 日志 RPT-005 | §21 |

---

## 25. 已定决策（A–H）

以下为客户 **采纳** 的产品侧建议（2026-05-10），实现与评审以此为准。

| # | 主题 | 已定结论 | 备注 |
|---|------|----------|------|
| **A** | 平台 API 与网关同域 | **同主机、路径分流**：`https://api.{domain}/mlk/v1` + `https://api.{domain}/mlk/platform/v1`（共用前缀 **`/mlk`**） | 未来如需安全域隔离可再拆子域 |
| **B** | 租户钱包 vs 个人钱包 | **严格分账**：Key 固定 `personal` **或** `org+org_id`，分别只扣对应钱包；**禁止混扣** | 与 §7、§8 一致 |
| **C** | 组织上下文 | **JWT 默认 org + `X-Org-Id` 优先**（合法成员时） | 见 §2.2 |
| **D** | `/mlk/v1/embeddings` | **实现路由**；计价按 token；可通过 **CFG** 全局关闭并返回明确错误 | 关开关不关路径 |
| **E** | 管理员高危操作 | **调账、渠道密钥、重置 owner、全局支付/网关 CFG** → **强制二次验证**；只读列表 JWT；**TOTP** 可在 CFG 对管理员可选开启 | |
| **F** | 阿里云 vs 腾讯云 | **优先跟备案主体 / 已有账号**；无则微信支付重心可偏 **腾讯云**，否则任选 RDS/OSS/日志等价方案 | |
| **G** | 客户端断开 SSE | **尽力 cancel 上游**；有 **`usage`** 按 usage；否则 **按 PRD §17.2 估算**；无费用时可不扣或最小单位（实现择一并标记） | |
| **H** | API Key 前缀 | **`mk_live_`** / **`mk_test_`** | 见 §2.2 |

---

## 26. 修订记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2026-05-10 | 初版：全量接口清单与约定 |
| v1.1 | 2026-05-10 | §25 补充 A–H **产品侧建议** 与理由 |
| v1.2 | 2026-05-10 | 客户 **采纳 A–H**；§25 改为已定；§2.2 / §7 / 文首背景表同步 |
| v1.3 | 2026-05-10 | 增补 **OpenAPI 3.1**：[openapi/modlink-gateway.openapi.yaml](./openapi/modlink-gateway.openapi.yaml)、[openapi/modlink-platform.openapi.yaml](./openapi/modlink-platform.openapi.yaml) |
| v1.4 | 2026-05-10 | **统一路径前缀 `/mlk`**（共机部署）；推理 **`/mlk/v1`**、平台 **`/mlk/platform/v1`**；探活 **`/mlk/health`**；OpenAPI **1.4.0** |
| v1.5 | 2026-05-10 | **附录 A**：典型接口 **JSON 入参 / 出参** 示例（认证、Key、钱包、充值、用量、Chat） |

---

**说明**：**机器可读接口定义** 位于 **`doc/openapi/`**（网关与平台 **两份 YAML**）；字段级 **`components.schemas`** 可随实现细化。**人读**以本文 **§2**（通用包裹）+ **附录 A**（典型 JSON）+ 各章路径表为准。占位域名 `api.{domain}` 部署时替换。
