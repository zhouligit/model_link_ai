-- ModLinkCloud schema (aligned with doc/模链云-MySQL数据库设计.md) + users.role for admin

CREATE DATABASE IF NOT EXISTS modlink_cloud
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE modlink_cloud;

SET NAMES utf8mb4;

CREATE TABLE users (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  email           VARCHAR(255) NULL,
  phone           VARCHAR(32)  NULL,
  password_hash   VARCHAR(255) NOT NULL,
  display_name    VARCHAR(128) NULL,
  avatar_url      VARCHAR(512) NULL,
  role            VARCHAR(32)  NOT NULL DEFAULT 'user' COMMENT 'user, admin',
  status          VARCHAR(32)  NOT NULL DEFAULT 'active',
  last_login_at   DATETIME(3)  NULL,
  created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_users_email (email),
  UNIQUE KEY uk_users_phone (phone),
  KEY idx_users_status (status)
) ENGINE=InnoDB;

CREATE TABLE user_refresh_tokens (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id      BIGINT UNSIGNED NOT NULL,
  token_hash   VARCHAR(64) NOT NULL,
  expires_at   DATETIME(3)   NOT NULL,
  revoked_at   DATETIME(3)   NULL,
  created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  device_info  VARCHAR(512)  NULL,
  KEY idx_urt_user (user_id),
  KEY idx_urt_expires (expires_at),
  CONSTRAINT fk_urt_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE orgs (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name        VARCHAR(256) NOT NULL,
  slug        VARCHAR(64)  NULL,
  owner_user_id BIGINT UNSIGNED NOT NULL,
  status      VARCHAR(32)  NOT NULL DEFAULT 'active',
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_orgs_slug (slug),
  KEY idx_orgs_owner (owner_user_id),
  CONSTRAINT fk_orgs_owner FOREIGN KEY (owner_user_id) REFERENCES users (id) ON DELETE RESTRICT
) ENGINE=InnoDB;

CREATE TABLE org_members (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  org_id     BIGINT UNSIGNED NOT NULL,
  user_id    BIGINT UNSIGNED NOT NULL,
  role       VARCHAR(32) NOT NULL,
  joined_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_org_member (org_id, user_id),
  KEY idx_om_user (user_id),
  CONSTRAINT fk_om_org FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE CASCADE,
  CONSTRAINT fk_om_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE org_invitations (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  org_id       BIGINT UNSIGNED NOT NULL,
  email        VARCHAR(255) NULL,
  phone        VARCHAR(32)  NULL,
  token_hash   VARCHAR(64) NOT NULL,
  role         VARCHAR(32)  NOT NULL DEFAULT 'member',
  status       VARCHAR(32)  NOT NULL DEFAULT 'pending',
  expires_at   DATETIME(3)  NOT NULL,
  created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_oi_org (org_id),
  CONSTRAINT fk_oi_org FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE api_keys (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id        BIGINT UNSIGNED NOT NULL,
  org_id         BIGINT UNSIGNED NULL,
  scope          VARCHAR(16)  NOT NULL,
  name           VARCHAR(128) NOT NULL,
  key_prefix     VARCHAR(32)  NOT NULL,
  key_hash       VARCHAR(64)  NOT NULL,
  status         VARCHAR(32)  NOT NULL DEFAULT 'active',
  expires_at     DATETIME(3)  NULL,
  ip_allowlist   JSON NULL,
  last_used_at   DATETIME(3)  NULL,
  created_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_ak_user (user_id),
  KEY idx_ak_org (org_id),
  KEY idx_ak_prefix (key_prefix),
  CONSTRAINT fk_ak_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_ak_org FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE wallet_accounts (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  owner_type     VARCHAR(16) NOT NULL,
  owner_id       BIGINT UNSIGNED NOT NULL,
  balance_cents  BIGINT NOT NULL DEFAULT 0,
  currency       CHAR(3) NOT NULL DEFAULT 'CNY',
  status         VARCHAR(32) NOT NULL DEFAULT 'active',
  version        INT UNSIGNED NOT NULL DEFAULT 0,
  updated_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_wallet_owner (owner_type, owner_id),
  KEY idx_wallet_status (status)
) ENGINE=InnoDB;

CREATE TABLE wallet_transactions (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  wallet_account_id BIGINT UNSIGNED NOT NULL,
  direction        VARCHAR(16) NOT NULL,
  amount_cents     BIGINT NOT NULL,
  balance_after    BIGINT NOT NULL,
  biz_type         VARCHAR(32) NOT NULL,
  ref_type         VARCHAR(32) NULL,
  ref_id           BIGINT UNSIGNED NULL,
  request_id       VARCHAR(64) NULL,
  billing_type     VARCHAR(16) NULL,
  remark           VARCHAR(512) NULL,
  meta             JSON NULL,
  created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_wt_wallet_time (wallet_account_id, created_at),
  KEY idx_wt_request (request_id),
  CONSTRAINT fk_wt_account FOREIGN KEY (wallet_account_id) REFERENCES wallet_accounts (id) ON DELETE RESTRICT
) ENGINE=InnoDB;

CREATE TABLE orders (
  id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id             BIGINT UNSIGNED NOT NULL,
  org_id              BIGINT UNSIGNED NULL,
  order_type          VARCHAR(32) NOT NULL DEFAULT 'recharge',
  amount_cents        BIGINT NOT NULL,
  currency            CHAR(3) NOT NULL DEFAULT 'CNY',
  channel             VARCHAR(16) NOT NULL,
  status              VARCHAR(32) NOT NULL DEFAULT 'pending',
  provider_trade_no   VARCHAR(128) NULL,
  notify_payload_digest VARCHAR(64) NULL,
  paid_at             DATETIME(3) NULL,
  created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_orders_user (user_id),
  KEY idx_orders_status (status),
  CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
  CONSTRAINT fk_orders_org FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE channels (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name           VARCHAR(128) NOT NULL,
  channel_type   VARCHAR(32) NOT NULL DEFAULT 'openrouter',
  base_url       VARCHAR(512) NOT NULL,
  api_key_cipher TEXT NOT NULL,
  status         VARCHAR(32) NOT NULL DEFAULT 'active',
  circuit_break  TINYINT(1) NOT NULL DEFAULT 0,
  priority       INT NOT NULL DEFAULT 0,
  created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_channels_status (status)
) ENGINE=InnoDB;

CREATE TABLE platform_models (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  model_id          VARCHAR(256) NOT NULL,
  display_name      VARCHAR(256) NULL,
  enabled           TINYINT(1) NOT NULL DEFAULT 1,
  default_channel_id BIGINT UNSIGNED NULL,
  max_context_hint  INT NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_pm_model_id (model_id),
  KEY fk_pm_channel (default_channel_id),
  CONSTRAINT fk_pm_channel FOREIGN KEY (default_channel_id) REFERENCES channels (id) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE model_routes (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  client_model_id    VARCHAR(256) NOT NULL,
  channel_id         BIGINT UNSIGNED NOT NULL,
  upstream_model_id  VARCHAR(256) NOT NULL,
  enabled            TINYINT(1) NOT NULL DEFAULT 1,
  created_at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_route_client (client_model_id),
  KEY idx_route_channel (channel_id),
  CONSTRAINT fk_mr_channel FOREIGN KEY (channel_id) REFERENCES channels (id) ON DELETE RESTRICT
) ENGINE=InnoDB;

CREATE TABLE pricing_models (
  id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  model_id             VARCHAR(256) NOT NULL,
  input_per_1k_cents   BIGINT NOT NULL DEFAULT 0,
  output_per_1k_cents  BIGINT NOT NULL DEFAULT 0,
  effective_from       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  effective_to         DATETIME(3) NULL,
  created_at           DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_price_model_time (model_id, effective_from),
  KEY idx_price_model (model_id)
) ENGINE=InnoDB;

CREATE TABLE plans (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  code        VARCHAR(64) NOT NULL,
  name        VARCHAR(128) NOT NULL,
  description VARCHAR(512) NULL,
  config      JSON NULL,
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_plans_code (code)
) ENGINE=InnoDB;

CREATE TABLE plan_model_grants (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  plan_id    BIGINT UNSIGNED NOT NULL,
  model_id   VARCHAR(256) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_plan_model (plan_id, model_id),
  CONSTRAINT fk_pmg_plan FOREIGN KEY (plan_id) REFERENCES plans (id) ON DELETE CASCADE
) ENGINE=InnoDB;

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
  billing_type     VARCHAR(16) NOT NULL DEFAULT 'actual',
  http_status      INT NULL,
  upstream_status  VARCHAR(64) NULL,
  latency_ms       INT UNSIGNED NULL,
  prompt_stored    TINYINT(1) NOT NULL DEFAULT 0,
  created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_infer_req (request_id),
  KEY idx_infer_user_time (user_id, created_at),
  KEY idx_infer_org_time (org_id, created_at),
  KEY idx_infer_model_time (model, created_at),
  CONSTRAINT fk_il_key FOREIGN KEY (api_key_id) REFERENCES api_keys (id) ON DELETE RESTRICT,
  CONSTRAINT fk_il_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
  CONSTRAINT fk_il_org FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE risk_rate_limit_rules (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  scope_type  VARCHAR(32) NOT NULL,
  scope_id    BIGINT UNSIGNED NULL,
  limit_qps   DECIMAL(10,2) NOT NULL DEFAULT 10,
  burst       INT UNSIGNED NOT NULL DEFAULT 20,
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_rl_scope (scope_type, scope_id)
) ENGINE=InnoDB;

CREATE TABLE risk_quota_rules (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  scope_type   VARCHAR(32) NOT NULL,
  scope_id     BIGINT UNSIGNED NOT NULL,
  quota_type   VARCHAR(32) NOT NULL,
  limit_value  BIGINT NOT NULL,
  enabled      TINYINT(1) NOT NULL DEFAULT 1,
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_rq_scope (scope_type, scope_id)
) ENGINE=InnoDB;

CREATE TABLE risk_blacklist (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  entry_type  VARCHAR(32) NOT NULL,
  entry_value VARCHAR(512) NOT NULL,
  reason      VARCHAR(512) NULL,
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_rb_type_val (entry_type, entry_value)
) ENGINE=InnoDB;

CREATE TABLE risk_whitelist (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  entry_type  VARCHAR(32) NOT NULL,
  entry_value VARCHAR(512) NOT NULL,
  remark      VARCHAR(512) NULL,
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_rw_type_val (entry_type, entry_value)
) ENGINE=InnoDB;

CREATE TABLE admin_audit_logs (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  admin_user_id BIGINT UNSIGNED NOT NULL,
  action       VARCHAR(64) NOT NULL,
  resource_type VARCHAR(64) NULL,
  resource_id   VARCHAR(128) NULL,
  detail        JSON NULL,
  ip            VARCHAR(64) NULL,
  user_agent    VARCHAR(512) NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_audit_admin_time (admin_user_id, created_at),
  CONSTRAINT fk_audit_user FOREIGN KEY (admin_user_id) REFERENCES users (id) ON DELETE RESTRICT
) ENGINE=InnoDB;

CREATE TABLE system_config (
  config_key   VARCHAR(128) NOT NULL PRIMARY KEY,
  config_value JSON NOT NULL,
  description  VARCHAR(512) NULL,
  updated_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB;

CREATE TABLE org_webhooks (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  org_id      BIGINT UNSIGNED NOT NULL,
  url         VARCHAR(1024) NOT NULL,
  events      JSON NOT NULL,
  secret      VARCHAR(128) NOT NULL,
  enabled     TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_org_webhook (org_id),
  CONSTRAINT fk_ow_org FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE CASCADE
) ENGINE=InnoDB;
