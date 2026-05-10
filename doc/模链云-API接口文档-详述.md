# 模链云（ModLinkCloud）接口详述（表格 + JSON + HTTP 示例）

| 属性 | 内容 |
|------|------|
| **文档版本** | v1.0 |
| **更新日期** | 2026-05-10 |
| **关联** | [模链云-API接口文档.md](./模链云-API接口文档.md)（路径与语义总表）、[OpenAPI](./openapi/README.md) |

---

## 全局约定

| 项 | 值 |
|----|-----|
| **推理网关 Base** | `https://api.{domain}/mlk/v1` |
| **平台 API Base** | `https://api.{domain}/mlk/platform/v1` |
| **Host 示例** | `api.example.com`（替换为备案域名） |
| **平台成功响应** | `{ "code": 0, "message": "ok", "data": { }, "request_id": "req_xxx" }` |
| **鉴权（平台）** | `Authorization: Bearer <access_token>`；租户可选 **`X-Org-Id: <org_id>`** |
| **鉴权（推理）** | `Authorization: Bearer <mk_live_xxx 或 mk_test_xxx>` |

以下每条均含：**请求参数表**、**JSON 示例**、**HTTP 请求示例**、**响应 data 表**、**响应 JSON 示例**。未特殊说明时 **`Content-Type: application/json`**。

---

## 一、推理网关（OpenAI 兼容）

### Chat Completions（非流式）

**POST** `/mlk/v1/chat/completions`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | `Bearer mk_live_...` |
| Idempotency-Key | header | string | 否 | 幂等键，建议 UUID |
| （Body） | body | object | 是 | 与 OpenAI Chat Completions 一致 |

#### 请求参数示例（JSON）

```json
{
  "model": "openai/gpt-4o",
  "messages": [
    { "role": "user", "content": "你好" }
  ],
  "stream": false,
  "temperature": 0.7
}
```

#### 请求示例

```http
POST /mlk/v1/chat/completions HTTP/1.1
Host: api.example.com
Authorization: Bearer mk_live_xxxxxxxx
Content-Type: application/json

{"model":"openai/gpt-4o","messages":[{"role":"user","content":"你好"}],"stream":false}
```

#### 响应说明

- **`stream: false`**：`Content-Type: application/json`，body 为 **OpenAI 兼容** `chat.completion`（含 `choices`、`usage` 等）；可额外带 **`request_id`**。
- **`stream: true`**：`Content-Type: text/event-stream`（SSE），非单一 JSON，本文不展开字节流示例。

#### 响应示例（非流式，节选）

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1715320800,
  "model": "openai/gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": { "role": "assistant", "content": "你好！" },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 5,
    "total_tokens": 15
  },
  "request_id": "req_infer_001"
}
```

---

### 列出模型

**GET** `/mlk/v1/models`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | `Bearer mk_live_...` |

#### 请求示例

```http
GET /mlk/v1/models HTTP/1.1
Host: api.example.com
Authorization: Bearer mk_live_xxxxxxxx
```

#### 响应说明

OpenAI 兼容 `list` 对象（`data` 数组等）；具体字段以上游/实现为准。

#### 响应示例（节选）

```json
{
  "object": "list",
  "data": [
    {
      "id": "openai/gpt-4o",
      "object": "model",
      "created": 1715320800,
      "owned_by": "openai"
    }
  ]
}
```

---

### 获取单个模型

**GET** `/mlk/v1/models/{model_id}`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer API Key |
| model_id | path | string | 是 | 路径中若含 `/` 需 URL 编码 |

#### 请求示例

```http
GET /mlk/v1/models/openai%2Fgpt-4o HTTP/1.1
Host: api.example.com
Authorization: Bearer mk_live_xxxxxxxx
```

#### 响应示例（节选）

```json
{
  "id": "openai/gpt-4o",
  "object": "model",
  "created": 1715320800,
  "owned_by": "openai"
}
```

---

### Embeddings

**POST** `/mlk/v1/embeddings`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer API Key |
| model | body | string | 是 | 模型 ID |
| input | body | string 或 string[] | 是 | 嵌入输入 |

#### 请求参数示例（JSON）

```json
{
  "model": "openai/text-embedding-3-small",
  "input": "hello world"
}
```

#### 请求示例

```http
POST /mlk/v1/embeddings HTTP/1.1
Host: api.example.com
Authorization: Bearer mk_live_xxxxxxxx
Content-Type: application/json

{"model":"openai/text-embedding-3-small","input":"hello world"}
```

#### 响应示例（节选）

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.1, 0.2],
      "index": 0
    }
  ],
  "model": "openai/text-embedding-3-small",
  "usage": {
    "prompt_tokens": 3,
    "total_tokens": 3
  }
}
```

---

## 二、平台 API — 认证与账号

### 用户注册

**POST** `/mlk/platform/v1/auth/register`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| email | body | string | 与 phone 二选一 | 邮箱 |
| phone | body | string | 与 email 二选一 | 手机号 |
| password | body | string | 是 | 登录密码 |
| invite_code | body | string | 否 | 邀请码（开启 CFG 时必填） |

#### 请求参数示例（JSON）

```json
{
  "email": "user@example.com",
  "password": "YourStrongPassword1!",
  "invite_code": ""
}
```

#### 请求示例

```http
POST /mlk/platform/v1/auth/register HTTP/1.1
Host: api.example.com
Content-Type: application/json

{"email":"user@example.com","password":"YourStrongPassword1!","invite_code":""}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | string | 用户 ID |
| access_token | string | 访问令牌 JWT |
| refresh_token | string | 刷新令牌 |
| expires_in | number | access_token 有效秒数 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_001",
  "data": {
    "user_id": "usr_001",
    "access_token": "eyJhbGciOi...",
    "refresh_token": "rt_xxxxxxxx",
    "expires_in": 7200
  }
}
```

---

### 密码登录

**POST** `/mlk/platform/v1/auth/login`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| email | body | string | 与 phone 二选一 | 邮箱登录 |
| phone | body | string | 与 email 二选一 | 手机登录 |
| password | body | string | 是 | 密码 |

#### 请求参数示例（JSON）

```json
{
  "email": "user@example.com",
  "password": "YourStrongPassword1!"
}
```

#### 请求示例

```http
POST /mlk/platform/v1/auth/login HTTP/1.1
Host: api.example.com
Content-Type: application/json

{"email":"user@example.com","password":"YourStrongPassword1!"}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | string | 用户 ID |
| access_token | string | JWT |
| refresh_token | string | 刷新令牌 |
| expires_in | number | 秒 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_002",
  "data": {
    "user_id": "usr_001",
    "access_token": "eyJhbGciOi...",
    "refresh_token": "rt_yyyyyyyy",
    "expires_in": 7200
  }
}
```

---

### 发送短信验证码（扩展建议）

**POST** `/mlk/platform/v1/auth/sms/send`

> 与总表相比为 **境内短信登录/找回密码** 常见能力；实现与短信服务商对接后以 CFG 开关控制。

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| phone | body | string | 是 | 中国大陆手机号，如 13800138000 |
| scene | body | string | 否 | `login`（默认）、`reset_password`、`bind_phone` |

#### 请求参数示例（JSON）

```json
{
  "phone": "13800138000",
  "scene": "login"
}
```

#### 请求示例

```http
POST /mlk/platform/v1/auth/sms/send HTTP/1.1
Host: api.example.com
Content-Type: application/json

{"phone":"13800138000","scene":"login"}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| expireInSeconds | number | 验证码有效秒数（前端展示） |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_sms_001",
  "data": {
    "expireInSeconds": 300
  }
}
```

---

### 短信验证码登录（扩展建议）

**POST** `/mlk/platform/v1/auth/sms/login`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| phone | body | string | 是 | 手机号 |
| code | body | string | 是 | 短信验证码 |

#### 请求参数示例（JSON）

```json
{
  "phone": "13800138000",
  "code": "123456"
}
```

#### 响应 data

同 **密码登录**（`user_id`、`access_token`、`refresh_token`、`expires_in`）。

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_sms_002",
  "data": {
    "user_id": "usr_001",
    "access_token": "eyJhbGciOi...",
    "refresh_token": "rt_zzzzzzzz",
    "expires_in": 7200
  }
}
```

---

### 刷新 Token

**POST** `/mlk/platform/v1/auth/refresh`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| refresh_token | body | string | 是 | 登录时下发的 refresh_token |

#### 请求参数示例（JSON）

```json
{
  "refresh_token": "rt_xxxxxxxx"
}
```

#### 请求示例

```http
POST /mlk/platform/v1/auth/refresh HTTP/1.1
Host: api.example.com
Content-Type: application/json

{"refresh_token":"rt_xxxxxxxx"}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| access_token | string | 新 JWT |
| refresh_token | string | 新刷新令牌（轮换策略下返回） |
| expires_in | number | 秒 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_003",
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "rt_newwwww",
    "expires_in": 7200
  }
}
```

---

### 登出

**POST** `/mlk/platform/v1/auth/logout`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer access_token |

#### 请求示例

```http
POST /mlk/platform/v1/auth/logout HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOi...
Content-Type: application/json

{}
```

#### 响应 data

可为空对象。

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_004",
  "data": {}
}
```

---

### 当前登录上下文

**GET** `/mlk/platform/v1/auth/me`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |

#### 请求示例

```http
GET /mlk/platform/v1/auth/me HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOi...
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| user | object | 基本信息 |
| roles | string[] | 角色列表 |
| current_org_id | string \| null | 当前默认组织 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_005",
  "data": {
    "user": {
      "id": "usr_001",
      "email": "user@example.com",
      "display_name": "张三"
    },
    "roles": ["user"],
    "current_org_id": "org_001"
  }
}
```

---

### 申请重置密码（发邮件/短信）

**POST** `/mlk/platform/v1/auth/password/reset-request`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| email | body | string | 与 phone 二选一 | 邮箱 |
| phone | body | string | 与 email 二选一 | 手机 |

#### 请求参数示例（JSON）

```json
{
  "email": "user@example.com"
}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| delivered | boolean | 是否已触发发送（防枚举时可恒 true） |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_006",
  "data": {
    "delivered": true
  }
}
```

---

### 确认重置密码

**POST** `/mlk/platform/v1/auth/password/reset-confirm`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| token | body | string | 是 | 邮件/短信中的重置令牌 |
| new_password | body | string | 是 | 新密码 |

#### 请求参数示例（JSON）

```json
{
  "token": "rst_xxxxxxxx",
  "new_password": "NewStrongPassword1!"
}
```

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_007",
  "data": {}
}
```

---

## 三、个人资料

### 获取当前用户资料

**GET** `/mlk/platform/v1/users/me`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 用户 ID |
| email | string \| null | 邮箱 |
| phone | string \| null | 手机 |
| display_name | string \| null | 昵称 |
| avatar_url | string \| null | 头像 URL |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_008",
  "data": {
    "id": "usr_001",
    "email": "user@example.com",
    "phone": null,
    "display_name": "张三",
    "avatar_url": null
  }
}
```

---

### 更新当前用户资料

**PATCH** `/mlk/platform/v1/users/me`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| display_name | body | string | 否 | 昵称 |
| avatar_url | body | string | 否 | 头像 |

#### 请求参数示例（JSON）

```json
{
  "display_name": "李四",
  "avatar_url": "https://cdn.example.com/a.png"
}
```

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_009",
  "data": {
    "id": "usr_001",
    "display_name": "李四",
    "avatar_url": "https://cdn.example.com/a.png"
  }
}
```

---

## 四、企业租户

### 创建租户

**POST** `/mlk/platform/v1/orgs`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| name | body | string | 是 | 组织名称 |
| slug | body | string | 否 | 唯一短标识 |

#### 请求参数示例（JSON）

```json
{
  "name": "示例科技有限公司",
  "slug": "demo-tech"
}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | org_id |
| name | string | 名称 |
| slug | string \| null | 短标识 |
| role | string | 当前用户在组织中的角色 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_010",
  "data": {
    "id": "org_001",
    "name": "示例科技有限公司",
    "slug": "demo-tech",
    "role": "owner"
  }
}
```

---

### 租户列表

**GET** `/mlk/platform/v1/orgs`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| items | array | 组织列表 |
| items[].id | string | 组织 ID |
| items[].name | string | 名称 |
| items[].role | string | 当前用户在该组织的角色 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_011",
  "data": {
    "items": [
      {
        "id": "org_001",
        "name": "示例科技有限公司",
        "role": "owner"
      }
    ]
  }
}
```

---

### 切换当前租户

**POST** `/mlk/platform/v1/orgs/{org_id}/switch`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| org_id | path | string | 是 | 目标组织 ID |

#### 请求示例

```http
POST /mlk/platform/v1/orgs/org_001/switch HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOi...
Content-Length: 0
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| current_org_id | string | 切换后的当前组织 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_012",
  "data": {
    "current_org_id": "org_001"
  }
}
```

---

### 邀请成员

**POST** `/mlk/platform/v1/orgs/{org_id}/invitations`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| X-Org-Id | header | string | 否 | 可与 path org_id 一致 |
| org_id | path | string | 是 | 组织 ID |
| email | body | string | 与 phone 二选一 | 被邀请邮箱 |
| phone | body | string | 与 email 二选一 | 被邀请手机 |
| role | body | string | 是 | `admin` 或 `member` |

#### 请求参数示例（JSON）

```json
{
  "email": "member@example.com",
  "role": "member"
}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| invitation_id | string | 邀请 ID |
| expires_at | string | 过期时间 ISO8601 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_013",
  "data": {
    "invitation_id": "inv_001",
    "expires_at": "2026-05-11T12:00:00+08:00"
  }
}
```

---

## 五、API Key

### 创建 API Key

**POST** `/mlk/platform/v1/api-keys`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| X-Org-Id | header | string | 否 | 创建组织级 Key 时推荐传入 |
| name | body | string | 是 | 显示名 |
| scope | body | string | 是 | `personal` 或 `org` |
| org_id | body | string | 条件必填 | `scope=org` 时必填 |
| expires_at | body | string | 否 | ISO8601，空表示不过期 |
| ip_allowlist | body | string[] | 否 | CIDR 列表 |

#### 请求参数示例（JSON）

```json
{
  "name": "生产后端",
  "scope": "org",
  "org_id": "org_001",
  "expires_at": null,
  "ip_allowlist": ["203.0.113.0/24"]
}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | Key ID |
| name | string | 名称 |
| scope | string | personal / org |
| org_id | string \| null | 组织 ID |
| secret | string | **仅本次返回完整密钥** |
| prefix | string | 脱敏前缀 |
| created_at | string | 创建时间 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_014",
  "data": {
    "id": "key_001",
    "name": "生产后端",
    "scope": "org",
    "org_id": "org_001",
    "secret": "mk_live_xxxxxxxxxxxxxxxx",
    "prefix": "mk_live_xxxx",
    "created_at": "2026-05-10T12:00:00+08:00"
  }
}
```

---

## 六、钱包与订单

### 查询钱包余额

**GET** `/mlk/platform/v1/wallet`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| X-Org-Id | header | string | 否 | 查询组织钱包时与 org_id 一致 |
| org_id | query | string | 否 | 不传则查个人钱包 |

#### 请求示例

```http
GET /mlk/platform/v1/wallet?org_id=org_001 HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOi...
X-Org-Id: org_001
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| balance_cents | number | 余额（分） |
| currency | string | 如 `CNY` |
| credit_status | string | `active` / `frozen` |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_015",
  "data": {
    "balance_cents": 150000,
    "currency": "CNY",
    "credit_status": "active"
  }
}
```

---

### 创建充值订单

**POST** `/mlk/platform/v1/orders/recharge`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| amount_cents | body | number | 是 | 充值金额（分） |
| channel | body | string | 是 | `wechat` 或 `alipay` |
| org_id | body | string | 否 | 给组织充值时填写 |

#### 请求参数示例（JSON）

```json
{
  "amount_cents": 10000,
  "channel": "wechat",
  "org_id": "org_001"
}
```

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| order_id | string | 订单号 |
| payment_params | object | 下游拉起支付所需参数（微信 JSAPI/APP、支付宝等） |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_016",
  "data": {
    "order_id": "ord_001",
    "payment_params": {
      "jsapi_params": {}
    }
  }
}
```

---

## 七、用量与账单

### 用量汇总

**GET** `/mlk/platform/v1/usage/summary`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| org_id | query | string | 否 | 组织维度 |
| from | query | string | 是 | 开始时间 ISO8601 |
| to | query | string | 是 | 结束时间 ISO8601 |
| granularity | query | string | 否 | `day` 或 `hour` |

#### 请求示例

```http
GET /mlk/platform/v1/usage/summary?from=2026-05-01T00:00:00%2B08:00&to=2026-05-10T23:59:59%2B08:00&granularity=day HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOi...
```

#### 响应 data（示例结构，可实现微调）

| 字段 | 类型 | 说明 |
|------|------|------|
| granularity | string | day / hour |
| series | array | 按时间段聚合 |

#### 响应示例（节选）

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_017",
  "data": {
    "granularity": "day",
    "series": [
      {
        "date": "2026-05-09",
        "total_calls": 1200,
        "input_tokens": 800000,
        "output_tokens": 300000,
        "cost_cents": 45000
      }
    ]
  }
}
```

---

### 账单明细列表

**GET** `/mlk/platform/v1/bills`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 是 | Bearer JWT |
| org_id | query | string | 否 | |
| from | query | string | 否 | |
| to | query | string | 否 | |
| model | query | string | 否 | |
| page | query | number | 否 | 默认 1 |
| page_size | query | number | 否 | 默认 20 |

#### 响应 data

| 字段 | 类型 | 说明 |
|------|------|------|
| items | array | 账单行 |
| page | number | 页码 |
| page_size | number | 每页条数 |
| total | number | 总条数 |

#### 响应示例（节选）

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_018",
  "data": {
    "items": [
      {
        "request_id": "req_infer_001",
        "time": "2026-05-10T12:00:01+08:00",
        "model": "openai/gpt-4o",
        "input_tokens": 100,
        "output_tokens": 50,
        "cost_cents": 120,
        "billing_type": "actual"
      }
    ],
    "page": 1,
    "page_size": 20,
    "total": 1
  }
}
```

---

## 八、支付回调（平台接收第三方）

### 微信支付异步通知

**POST** `/mlk/platform/v1/payment/notify/wechat`

#### 说明

- **不经 JWT**；请求体 / 签名依 **微信支付文档**（可能为 XML 或 JSON）。  
- **响应正文**须符合微信要求（如返回 `SUCCESS` 字符串），**非**本节通用 JSON 包裹。

#### 请求示例（占位）

```http
POST /mlk/platform/v1/payment/notify/wechat HTTP/1.1
Host: api.example.com
Content-Type: application/xml

<xml>...</xml>
```

---

### 支付宝异步通知

**POST** `/mlk/platform/v1/payment/notify/alipay`

#### 说明

通常为 **`application/x-www-form-urlencoded`**；返回 **`success`** 等字符串；详见支付宝文档。

---

## 九、健康检查

### 存活探针

**GET** `/mlk/health`

#### 请求示例

```http
GET /mlk/health HTTP/1.1
Host: api.example.com
```

#### 响应示例

```http
HTTP/1.1 200 OK
Content-Type: text/plain

OK
```

（实现可选用纯文本或极简 JSON。）

---

### 依赖健康（平台）

**GET** `/mlk/platform/v1/health`

#### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| Authorization | header | string | 否 | 可选，用于内网探针 |

#### 响应示例

```json
{
  "code": 0,
  "message": "ok",
  "request_id": "req_health",
  "data": {
    "mysql": "up",
    "redis": "up"
  }
}
```

---

## 十、管理端与其它路径

管理端前缀：**`/mlk/platform/v1/admin/`**，须 **管理员 JWT**。列表类接口 **`data`** 一般为：

```json
{
  "items": [],
  "page": 1,
  "page_size": 20,
  "total": 0
}
```

### 管理端接口一览（详述版可与总表对照补全）

| 接口说明 | 方法 | 路径 |
|----------|------|------|
| 用户列表 | GET | `/mlk/platform/v1/admin/users` |
| 用户详情 | GET | `/mlk/platform/v1/admin/users/{user_id}` |
| 禁用用户 | POST | `/mlk/platform/v1/admin/users/{user_id}/disable` |
| 启用用户 | POST | `/mlk/platform/v1/admin/users/{user_id}/enable` |
| 租户列表 | GET | `/mlk/platform/v1/admin/orgs` |
| 渠道列表 | GET | `/mlk/platform/v1/admin/channels` |
| 创建渠道 | POST | `/mlk/platform/v1/admin/channels` |
| 模型列表 | GET | `/mlk/platform/v1/admin/models` |
| 全局配置 | GET/PATCH | `/mlk/platform/v1/admin/config` |
| 大屏 KPI | GET | `/mlk/platform/v1/admin/dashboard/screen` |

每条均可按本文相同结构扩展：**Query/Body 参数表 → JSON 示例 → HTTP 示例 → data 表 → 响应 JSON**。字段与 OpenAPI 对齐。

---

## 修订记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2026-05-10 | 初稿：按示例版式撰写网关、认证（含短信扩展）、租户、Key、钱包、用量、回调说明、管理端索引 |
