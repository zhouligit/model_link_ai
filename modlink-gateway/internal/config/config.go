package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Server struct {
	GatewayListen    string `yaml:"gateway_listen"`
	PlatformListen   string `yaml:"platform_listen"`
	PublicGatewayBase string `yaml:"public_gateway_base"`
}

type Database struct {
	DSN string `yaml:"dsn"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWT struct {
	Secret           string `yaml:"secret"`
	Issuer           string `yaml:"issuer"`
	AccessTTLMinutes int    `yaml:"access_ttl_minutes"`
	RefreshTTLDays   int    `yaml:"refresh_ttl_days"`
}

type Upstream struct {
	Mode               string `yaml:"mode"` // mock | openrouter
	TimeoutSeconds     int    `yaml:"timeout_seconds"`
	OpenRouterBaseURL  string `yaml:"openrouter_base_url"`
	OpenRouterAPIKey   string `yaml:"openrouter_api_key"`
}

type Payment struct {
	Mode string `yaml:"mode"` // mock | wechat | alipay
}

type SMS struct {
	Mode string `yaml:"mode"`
}

type Security struct {
	BootstrapAdminEmails []string `yaml:"bootstrap_admin_emails"`
}

type Embeddings struct {
	Enabled bool `yaml:"enabled"`
}

type Config struct {
	Server     Server     `yaml:"server"`
	Database   Database   `yaml:"database"`
	Redis      Redis      `yaml:"redis"`
	JWT        JWT        `yaml:"jwt"`
	Upstream   Upstream   `yaml:"upstream"`
	Payment    Payment    `yaml:"payment"`
	SMS        SMS        `yaml:"sms"`
	Security   Security   `yaml:"security"`
	Embeddings Embeddings `yaml:"embeddings"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	c.normalize()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) normalize() {
	if c.Server.GatewayListen == "" {
		c.Server.GatewayListen = ":8080"
	}
	if c.Server.PlatformListen == "" {
		c.Server.PlatformListen = ":8081"
	}
	if c.JWT.AccessTTLMinutes <= 0 {
		c.JWT.AccessTTLMinutes = 60
	}
	if c.JWT.RefreshTTLDays <= 0 {
		c.JWT.RefreshTTLDays = 30
	}
	if c.JWT.Issuer == "" {
		c.JWT.Issuer = "modlink-platform"
	}
	if c.Upstream.TimeoutSeconds <= 0 {
		c.Upstream.TimeoutSeconds = 120
	}
	if strings.TrimSpace(c.Upstream.Mode) == "" {
		c.Upstream.Mode = "mock"
	}
	if v := strings.TrimSpace(os.Getenv("MODLINK_OPENROUTER_API_KEY")); v != "" {
		c.Upstream.OpenRouterAPIKey = v
	}
	if strings.TrimSpace(c.Payment.Mode) == "" {
		c.Payment.Mode = "mock"
	}
	if strings.TrimSpace(c.SMS.Mode) == "" {
		c.SMS.Mode = "mock"
	}
}

func (c *Config) validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn required")
	}
	if len(c.JWT.Secret) < 16 {
		return fmt.Errorf("jwt.secret must be at least 16 characters")
	}
	return nil
}
