USE modlink_cloud;

-- 默认上游渠道（密钥占位：运行时可改为 plain:BASE64 或通过配置回退 openrouter_api_key）
INSERT INTO channels (id, name, channel_type, base_url, api_key_cipher, status, priority)
VALUES (1, 'openrouter-default', 'openrouter', 'https://openrouter.ai/api/v1',
  'plain:c2stb3ItdjEtcGxhY2Vob2xkZXI=', 'active', 0)
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO platform_models (model_id, display_name, enabled, default_channel_id)
VALUES ('openai/gpt-4o-mini', 'GPT-4o mini', 1, 1)
ON DUPLICATE KEY UPDATE display_name = VALUES(display_name);

INSERT INTO model_routes (client_model_id, channel_id, upstream_model_id, enabled)
VALUES ('openai/gpt-4o-mini', 1, 'openai/gpt-4o-mini', 1)
ON DUPLICATE KEY UPDATE upstream_model_id = VALUES(upstream_model_id);

INSERT INTO pricing_models (model_id, input_per_1k_cents, output_per_1k_cents)
VALUES ('openai/gpt-4o-mini', 1, 3)
ON DUPLICATE KEY UPDATE input_per_1k_cents = VALUES(input_per_1k_cents);

INSERT INTO system_config (config_key, config_value, description) VALUES
  ('registration_open', 'true', '开放注册'),
  ('maintenance_mode', 'false', '维护模式'),
  ('gateway_public_base_url', '"http://127.0.0.1:8080/mlk/v1"', '文档展示'),
  ('embeddings_enabled', 'true', 'embeddings'),
  ('min_recharge_cents', '100', '最小充值分')
ON DUPLICATE KEY UPDATE config_value = VALUES(config_value);

INSERT INTO risk_rate_limit_rules (scope_type, scope_id, limit_qps, burst, enabled)
VALUES ('global', NULL, 1000, 2000, 1);
