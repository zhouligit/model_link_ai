# 模链云（ModLinkCloud）MySQL 数据库设计

| 属性 | 内容 |
|------|------|
| **文档版本** | v1.0 |
| **更新日期** | 2026-05-10 |
| **数据库版本** | MySQL **8.0+**（建议 8.0；单机部署与阿里云 RDS / 腾讯云 MySQL 兼容） |
| **关联** | [API 接口文档](./模链云-API接口文档.md)、[PRD](./模链云-产品需求文档-PRD.md)、[服务划分](./模链云-服务划分与仓库说明.md) |

---

## 1. 说明

- **库名**：下文默认 **`modlink_cloud`**；若需多环境，可使用 `modlink_cloud_dev` / `modlink_cloud_prod`。  
- **字符集**：统一 **`utf8mb4`** + **`utf8mb4_unicode_ci`**（支持 Emoji 与完整 Unicode）。  
- **引擎**：**InnoDB**；金额使用 **`BIGINT`（单位：分）**，避免浮点误差。  
- **主键**：内部 **`BIGINT UNSIGNED AUTO_INCREMENT`**；对外 API 若使用字符串 ID，可在应用层映射或通过 **`public_id` VARCHAR(36) UNIQUE** 扩展（本 DDL 以性能优先采用数值主键，应用层生成 Snowflake/ULID 时可替换策略）。  
- **时间**：**`TIMESTAMP(3)`** 或 **`DATETIME(3)`**（毫秒）；示例使用 **`DATETIME(3)`** 便于与时区无关存储（应用层统一 UTC 或东八区）。  
- **JSON 字段**：MySQL 8 原生 **JSON** 类型用于可变配置、支付渠道扩展参数等。

**免责声明**：首次上线前请结合容量、分区、归档策略与索引监控再评审；本文不含分库分表。

---

## 2. 库与账号建议

```sql
CREATE DATABASE IF NOT EXISTS modlink_cloud
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

-- 应用账号仅授予 modlink_cloud.* 所需权限；禁止应用使用 root。
-- CREATE USER 'modlink_app'@'%' IDENTIFIED BY '***';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON modlink_cloud.* TO 'modlink_app'@'%';
```

---

## 3. 表清单（按模块）

| 模块 | 表名 | 用途摘要 |
|------|------|----------|
| 账号 | `users` | 用户账号 |
| 账号 | `user_refresh_tokens` | 刷新令牌（可轮换、吊销） |
| 租户 | `orgs` | 企业 / 组织 |
| 租户 | `org_members` | 成员与角色 |
| 租户 | `org_invitations` | 邀请记录 |
| 密钥 | `api_keys` | 平台 API Key（personal / org，仅存哈希） |
| 账务 | `wallet_accounts` | 钱包账户（用户或组织） |
| 账务 | `wallet_transactions` | 钱包流水（充值、扣费、调账等） |
| 订单 | `orders` | 充值订单（微信 / 支付宝） |
| 上游 | `channels` | OpenRouter 等渠道配置 |
| 模型 | `platform_models` | 平台侧模型目录 |
| 路由 | `model_routes` | 客户端 model → 渠道 + 上游 model |
| 计价 | `pricing_models` | 模型计价（输入/输出单价等） |
| 套餐 | `plans` | 套餐定义 |
| 套餐 | `plan_model_grants` | 套餐与模型可用关系（可选简化） |
| 用量 | `inference_logs` | 推理调用日志（元数据；Prompt 默认不落库） |
| 风控 | `risk_rate_limit_rules` | 限流规则 |
| 风控 | `risk_quota_rules` | 配额规则 |
| 风控 | `risk_blacklist` | 黑名单 |
| 风控 | `risk_whitelist` | 白名单 |
| 管理 | `admin_audit_logs` | 管理员操作审计 |
| 配置 | `system_config` | 全局 KV 配置 |
| 集成 | `org_webhooks` | 租户出站 Webhook 配置 |

可选扩展（未写入下方 DDL，可按需增加）：`admin_users`（独立管理员表，若不与 `users` 混用）、`login_attempts`（防爆破）、`email_verification_tokens`。

---

## 4. 建表语句

以下均在 **`USE modlink_cloud;`** 之后执行。

### 4.1 `users`

```sql
CREATE TABLE users (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  email           VARCHAR(255) NULL,
  phone           VARCHAR(32)  NULL,
  password_hash   VARCHAR(255) NOT NULL COMMENT 'bcrypt/argon2 等',
  display_name    VARCHAR(128) NULL,
  avatar_url      VARCHAR(512) NULL,
  status          VARCHAR(32)  NOT NULL DEFAULT 'active' COMMENT 'active, disabled',
  last_login_at   DATETIME(3)  NULL,
  created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_users_email (email),
  UNIQUE KEY uk_users_phone (phone),
  KEY idx_users_status (status)
) ENGINE=InnoDB COMMENT='终端用户与开发者账号';
```

### 4.2 `user_refresh_tokens`

```sql
CREATE TABLE user_refresh_tokens (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id      BIGINT UNSIGNED NOT NULL,
  token_hash   VARCHAR(64) NOT NULL COMMENT 'refresh_token 哈希',
  expires_at   DATETIME(3)   NOT NULL,
  revoked_at   DATETIME(3)   NULL,
  created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  device_info  VARCHAR(512)  NULL,
  KEY idx_urt_user (user_id),
  KEY idx_urt_expires (expires_at),
  CONSTRAINT fk_urt_user FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE
) ENGINE=InnoDB COMMENT='刷新令牌';
```

### 4.3 `orgs`

```sql
CREATE TABLE orgs (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name        VARCHAR(256) NOT NULL,
  slug        VARCHAR(64)  NULL COMMENT 'URL 友好标识',
  owner_user_id BIGINT UNSIGNED NOT NULL,
  status      VARCHAR(32)  NOT NULL DEFAULT 'active' COMMENT 'active, disabled',
  created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_orgs_slug (slug),
  KEY idx_orgs_owner (owner_user_id),
  CONSTRAINT fk_orgs_owner FOREIGN KEY (owner_user_id) REFERENCES users (id)
    ON DELETE RESTRICT
) ENGINE=InnoDB COMMENT='企业租户';
```

### 4.4 `org_members`

```sql
CREATE TABLE org_members (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  org_id     BIGINT UNSIGNED NOT NULL,
  user_id    BIGINT UNSIGNED NOT NULL,
  role       VARCHAR(32) NOT NULL COMMENT 'owner, admin, member',
  joined_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_org_member (org_id, user_id),
  KEY idx_om_user (user_id),
  CONSTRAINT fk_om_org FOREIGN KEY (org_id) REFERENCES orgs (id)
    ON DELETE CASCADE,
  CONSTRAINT fk_om_user FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE
) ENGINE=InnoDB COMMENT='组织成员';
```

### 4.5 `org_invitations`

```sql
CREATE TABLE org_invitations (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  org_id       BIGINT UNSIGNED NOT NULL,
  email        VARCHAR(255) NULL,
  phone        VARCHAR(32)  NULL,
  token_hash   VARCHAR(64)  NOT NULL,
  role         VARCHAR(32)  NOT NULL DEFAULT 'member',
  status       VARCHAR(32)  NOT NULL DEFAULT 'pending' COMMENT 'pending, accepted, expired, revoked',
  expires_at   DATETIME(3)  NOT NULL,
  created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_oi_org (org_id),
  CONSTRAINT fk_oi_org FOREIGN KEY (org_id) REFERENCES orgs (id)
    ON DELETE CASCADE
) ENGINE=InnoDB COMMENT='组织邀请';
```

### 4.6 `api_keys`

```sql
CREATE TABLE api_keys (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id        BIGINT UNSIGNED NOT NULL COMMENT '创建人',
  org_id         BIGINT UNSIGNED NULL COMMENT 'scope=org 时必填',
  scope          VARCHAR(16)  NOT NULL COMMENT 'personal, org',
  name           VARCHAR(128) NOT NULL,
  key_prefix     VARCHAR(16)  NOT NULL COMMENT 'mk_live_ / mk_test_ + 短前缀展示',
  key_hash       VARCHAR(64)  NOT NULL COMMENT '密钥哈希，不可逆',
  status         VARCHAR(32)  NOT NULL DEFAULT 'active' COMMENT 'active, disabled',
  expires_at     DATETIME(3)  NULL,
  ip_allowlist   JSON NULL COMMENT '["cidr", ...]，空表示不限制',
  last_used_at   DATETIME(3)  NULL,
  created_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_ak_user (user_id),
  KEY idx_ak_org (org_id),
  KEY idx_ak_prefix (key_prefix),
  CONSTRAINT fk_ak_user FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE,
  CONSTRAINT fk_ak_org FOREIGN KEY (org_id) REFERENCES orgs (id)
    ON DELETE CASCADE
) ENGINE=InnoDB COMMENT='推理网关 API Key';
```

### 4.7 `wallet_accounts`

```sql
CREATE TABLE wallet_accounts (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  owner_type     VARCHAR(16) NOT NULL COMMENT 'user, org',
  owner_id       BIGINT UNSIGNED NOT NULL COMMENT 'users.id 或 orgs.id',
  balance_cents  BIGINT NOT NULL DEFAULT 0 COMMENT '余额（分），非负约束在应用层',
  currency       CHAR(3) NOT NULL DEFAULT 'CNY',
  status         VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT 'active, frozen',
  version        INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '乐观锁',
  updated_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_wallet_owner (owner_type, owner_id),
  KEY idx_wallet_status (status)
) ENGINE=InnoDB COMMENT='钱包账户（用户个人或组织）';
```

### 4.8 `wallet_transactions`

```sql
CREATE TABLE wallet_transactions (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  wallet_account_id BIGINT UNSIGNED NOT NULL,
  direction        VARCHAR(16) NOT NULL COMMENT 'credit, debit',
  amount_cents     BIGINT NOT NULL COMMENT '金额（分），正数',
  balance_after    BIGINT NOT NULL COMMENT '变动后余额（分）',
  biz_type         VARCHAR(32) NOT NULL COMMENT 'recharge, inference, adjustment, refund',
  ref_type         VARCHAR(32) NULL COMMENT 'order, inference_log, manual',
  ref_id           BIGINT UNSIGNED NULL,
  request_id       VARCHAR(64) NULL COMMENT '推理 request_id 便于对账',
  billing_type     VARCHAR(16) NULL COMMENT 'actual, estimated',
  remark           VARCHAR(512) NULL,
  meta             JSON NULL,
  created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_wt_wallet_time (wallet_account_id, created_at),
  KEY idx_wt_request (request_id),
  CONSTRAINT fk_wt_account FOREIGN KEY (wallet_account_id) REFERENCES wallet_accounts (id)
    ON DELETE RESTRICT
) ENGINE=InnoDB COMMENT='钱包流水';
```

### 4.9 `orders`

```sql
CREATE TABLE orders (
  id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id             BIGINT UNSIGNED NOT NULL,
  org_id              BIGINT UNSIGNED NULL COMMENT '为组织充值时',
  order_type          VARCHAR(32) NOT NULL DEFAULT 'recharge',
  amount_cents        BIGINT NOT NULL COMMENT '应付金额（分）',
  currency            CHAR(3) NOT NULL DEFAULT 'CNY',
  channel             VARCHAR(16) NOT NULL COMMENT 'wechat, alipay',
  status              VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT 'pending, paid, failed, closed',
  provider_trade_no   VARCHAR(128) NULL COMMENT '第三方支付单号',
  notify_payload_digest VARCHAR(64) NULL COMMENT '幂等 / 审计摘要',
  paid_at             DATETIME(3) NULL,
  created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_orders_user (user_id),
  KEY idx_orders_status (status),
  CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE RESTRICT,
  CONSTRAINT fk_orders_org FOREIGN KEY (org_id) REFERENCES orgs (id)
    ON DELETE SET NULL
) ENGINE=InnoDB COMMENT='充值订单';
```

### 4.10 `channels`

```sql
CREATE TABLE channels (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name           VARCHAR(128) NOT NULL,
  channel_type   VARCHAR(32) NOT NULL DEFAULT 'openrouter' COMMENT 'openrouter, custom',
  base_url       VARCHAR(512) NOT NULL,
  api_key_cipher TEXT NOT NULL COMMENT '加密存储密文',
  status         VARCHAR(32) NOT NULL DEFAULT 'active',
  circuit_break  TINYINT(1) NOT NULL DEFAULT 0 COMMENT '熔断开关',
  priority       INT NOT NULL DEFAULT 0,
  created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_channels_status (status)
) ENGINE=InnoDB COMMENT='上游推理渠道';
```

### 4.11 `platform_models`

```sql
CREATE TABLE platform_models (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  model_id          VARCHAR(256) NOT NULL COMMENT '对外 model 名，如 openai/gpt-4o',
  display_name      VARCHAR(256) NULL,
  enabled           TINYINT(1) NOT NULL DEFAULT 1,
  default_channel_id BIGINT UNSIGNED NULL,
  max_context_hint  INT NULL COMMENT '可选校验提示',
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pm_model_id (model_id),
  KEY fk_pm_channel (default_channel_id),
  CONSTRAINT fk_pm_channel FOREIGN KEY (default_channel_id) REFERENCES channels (id)
    ON DELETE SET NULL
) ENGINE=InnoDB COMMENT='平台模型目录';
```

### 4.12 `model_routes`

```sql
CREATE TABLE model_routes (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  client_model_id    VARCHAR(256) NOT NULL COMMENT '客户端请求的 model',
  channel_id         BIGINT UNSIGNED NOT NULL,
  upstream_model_id  VARCHAR(256) NOT NULL COMMENT '上游实际模型 ID',
  enabled            TINYINT(1) NOT NULL DEFAULT 1,
  created_at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_route_client (client_model_id),
  KEY idx_route_channel (channel_id),
  CONSTRAINT fk_mr_channel FOREIGN KEY (channel_id) REFERENCES channels (id)
    ON DELETE RESTRICT
) ENGINE=InnoDB COMMENT='模型路由映射';
```

### 4.13 `pricing_models`

```sql
CREATE TABLE pricing_models (
  id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  model_id             VARCHAR(256) NOT NULL,
  input_per_1k_cents   BIGINT NOT NULL DEFAULT 0 COMMENT '每 1K input tokens 价格（分）',
  output_per_1k_cents  BIGINT NOT NULL DEFAULT 0 COMMENT '每 1K output tokens 价格（分）',
  effective_from       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  effective_to         DATETIME(3) NULL,
  created_at           DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_price_model_time (model_id, effective_from),
  KEY idx_price_model (model_id)
) ENGINE=InnoDB COMMENT='模型计价（支持时段版本）';
```

### 4.14 `plans`

```sql
CREATE TABLE plans (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  code        VARCHAR(64) NOT NULL,
  name        VARCHAR(128) NOT NULL,
  description VARCHAR(512) NULL,
  config      JSON NULL COMMENT '套餐参数：额度、周期等',
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_plans_code (code)
) ENGINE=InnoDB COMMENT='套餐定义';
```

### 4.15 `plan_model_grants`

```sql
CREATE TABLE plan_model_grants (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  plan_id    BIGINT UNSIGNED NOT NULL,
  model_id   VARCHAR(256) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_plan_model (plan_id, model_id),
  CONSTRAINT fk_pmg_plan FOREIGN KEY (plan_id) REFERENCES plans (id)
    ON DELETE CASCADE
) ENGINE=InnoDB COMMENT='套餐可用模型（可选）';
```

### 4.16 `inference_logs`

```sql
CREATE TABLE inference_logs (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  request_id       VARCHAR(64) NOT NULL,
  api_key_id       BIGINT UNSIGNED NOT NULL,
  user_id          BIGINT UNSIGNED NOT NULL,
  org_id           BIGINT UNSIGNED NULL,
  model            VARCHAR(256) NOT NULL,
  channel_id       BIGINT UNSIGNED NULL,
  input_tokens     INT UNSIGNED NULL,
  output_tokens    INT UNSIGNED NULL,
  cost_cents       BIGINT NOT NULL DEFAULT 0,
  billing_type     VARCHAR(16) NOT NULL DEFAULT 'actual' COMMENT 'actual, estimated',
  http_status      INT NULL,
  upstream_status  VARCHAR(64) NULL,
  latency_ms       INT UNSIGNED NULL,
  prompt_stored    TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否按调试策略落库',
  created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_infer_req (request_id),
  KEY idx_infer_user_time (user_id, created_at),
  KEY idx_infer_org_time (org_id, created_at),
  KEY idx_infer_model_time (model, created_at),
  CONSTRAINT fk_il_key FOREIGN KEY (api_key_id) REFERENCES api_keys (id)
    ON DELETE RESTRICT,
  CONSTRAINT fk_il_user FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE RESTRICT,
  CONSTRAINT fk_il_org FOREIGN KEY (org_id) REFERENCES orgs (id)
    ON DELETE SET NULL
) ENGINE=InnoDB COMMENT='推理调用元数据日志';
```

### 4.17 `risk_rate_limit_rules`

```sql
CREATE TABLE risk_rate_limit_rules (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  scope_type  VARCHAR(32) NOT NULL COMMENT 'global, user, api_key, org',
  scope_id    BIGINT UNSIGNED NULL COMMENT '对应实体 ID，global 时 NULL',
  limit_qps   DECIMAL(10,2) NOT NULL DEFAULT 10,
  burst       INT UNSIGNED NOT NULL DEFAULT 20,
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_rl_scope (scope_type, scope_id)
) ENGINE=InnoDB COMMENT='限流规则';
```

### 4.18 `risk_quota_rules`

```sql
CREATE TABLE risk_quota_rules (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  scope_type   VARCHAR(32) NOT NULL COMMENT 'user, org, api_key',
  scope_id     BIGINT UNSIGNED NOT NULL,
  quota_type   VARCHAR(32) NOT NULL COMMENT 'daily_tokens, monthly_cents, concurrent',
  limit_value  BIGINT NOT NULL,
  enabled      TINYINT(1) NOT NULL DEFAULT 1,
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_rq_scope (scope_type, scope_id)
) ENGINE=InnoDB COMMENT='配额规则';
```

### 4.19 `risk_blacklist`

```sql
CREATE TABLE risk_blacklist (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  entry_type  VARCHAR(32) NOT NULL COMMENT 'ip, user_id, api_key_id',
  entry_value VARCHAR(512) NOT NULL,
  reason      VARCHAR(512) NULL,
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_rb_type_val (entry_type, entry_value)
) ENGINE=InnoDB COMMENT='黑名单';
```

### 4.20 `risk_whitelist`

```sql
CREATE TABLE risk_whitelist (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  entry_type  VARCHAR(32) NOT NULL COMMENT 'ip, user_id, api_key_id',
  entry_value VARCHAR(512) NOT NULL,
  remark      VARCHAR(512) NULL,
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_rw_type_val (entry_type, entry_value)
) ENGINE=InnoDB COMMENT='白名单';
```

### 4.21 `admin_audit_logs`

```sql
CREATE TABLE admin_audit_logs (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  admin_user_id BIGINT UNSIGNED NOT NULL COMMENT 'users.id 中具有管理员角色的用户',
  action       VARCHAR(64) NOT NULL,
  resource_type VARCHAR(64) NULL,
  resource_id   VARCHAR(128) NULL,
  detail        JSON NULL,
  ip            VARCHAR(64) NULL,
  user_agent    VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_audit_admin_time (admin_user_id, created_at),
  CONSTRAINT fk_audit_user FOREIGN KEY (admin_user_id) REFERENCES users (id)
    ON DELETE RESTRICT
) ENGINE=InnoDB COMMENT='管理端审计日志';
```

### 4.22 `system_config`

```sql
CREATE TABLE system_config (
  config_key   VARCHAR(128) NOT NULL PRIMARY KEY,
  config_value JSON NOT NULL,
  description  VARCHAR(512) NULL,
  updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB COMMENT='全局配置（注册开关、维护模式、gateway_public_base_url、embeddings 开关等）';
```

### 4.23 `org_webhooks`

```sql
CREATE TABLE org_webhooks (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  org_id      BIGINT UNSIGNED NOT NULL,
  url         VARCHAR(1024) NOT NULL,
  events      JSON NOT NULL COMMENT '订阅事件列表',
  secret      VARCHAR(128) NOT NULL COMMENT '签名密钥',
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_org_webhook (org_id),
  CONSTRAINT fk_ow_org FOREIGN KEY (org_id) REFERENCES orgs (id)
    ON DELETE CASCADE
) ENGINE=InnoDB COMMENT='租户出站 Webhook';
```

---

## 5. 初始化配置示例（可选）

```sql
INSERT INTO system_config (config_key, config_value, description) VALUES
  ('registration_open', 'true', '是否开放注册'),
  ('maintenance_mode', 'false', '维护模式'),
  ('gateway_public_base_url', '"https://api.example.com/mlk/v1"', '对外文档展示的网关 Base'),
  ('embeddings_enabled', 'true', '是否开放 embeddings'),
  ('min_recharge_cents', '500', '最小充值金额（分）');
```

---

## 6. 与 OpenAPI / 钱包逻辑对齐说明

- **`api_keys.scope` + `org_id`**：与 PRD「个人 / 组织分账」一致；扣费时解析 Key → **解析 `wallet_accounts.owner_*`**。  
- **`wallet_transactions`**：`biz_type=inference` 与 **`request_id`**、**`billing_type`** 对齐附录与 PRD 估算策略。  
- **订单入账**：支付回调验签成功后 **`orders.status=paid`**，并写入 **`wallet_transactions`（credit）** 与更新 **`wallet_accounts`**（同一事务 + 乐观锁）。

---

## 7. 修订记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2026-05-10 | 初稿：库名 `modlink_cloud` + 23 张表 DDL |

---

**合并执行**：可将 §4 各 `CREATE TABLE` 合并为单个 `.sql` 文件用于 CI 迁移（Flyway / golang-migrate）；本文保持可读性为主。
