package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置（对应 configs/config.yaml）。
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Kubernetes    KubernetesConfig    `mapstructure:"kubernetes"`
	Helm          HelmConfig          `mapstructure:"helm"`
	Orchestration OrchestrationConfig `mapstructure:"orchestration"`
	JWT           JWTConfig           `mapstructure:"jwt"`
	RateLimit     RateLimitConfig     `mapstructure:"ratelimit"`
	CORS          CORSConfig          `mapstructure:"cors"`
	Notification  NotificationConfig  `mapstructure:"notification"`
	VM            VMConfig            `mapstructure:"vm"`
	N9E           N9EConfig           `mapstructure:"n9e"`
	Grafana       GrafanaConfig       `mapstructure:"grafana"`
}

// CORSConfig 跨域策略。AllowedOrigins 为空时允许所有域名（开发模式）。
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug | release | test
}

type DatabaseConfig struct {
	Host                   string `mapstructure:"host"`
	Port                   int    `mapstructure:"port"`
	User                   string `mapstructure:"user"`
	Password               string `mapstructure:"password"`
	Name                   string `mapstructure:"name"`
	SSLMode                string `mapstructure:"sslmode"`
	MaxOpenConns           int    `mapstructure:"max_open_conns"`
	MaxIdleConns           int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeMinutes int    `mapstructure:"conn_max_lifetime_minutes"`
}

type RedisConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type KubernetesConfig struct {
	InCluster  bool   `mapstructure:"incluster"`
	Kubeconfig string `mapstructure:"kubeconfig"`
}

type HelmRepo struct {
	Name string `mapstructure:"name"`
	URL  string `mapstructure:"url"`
}

type HelmConfig struct {
	Repos  []HelmRepo `mapstructure:"repos"`
	Charts HelmCharts `mapstructure:"charts"`
}

// HelmCharts 各模板对应的 chart 引用（如 vm/victoria-metrics-single）；空则跳过 Helm 仅做命名空间/配额。
type HelmCharts struct {
	Shared           string `mapstructure:"shared"`
	DedicatedSingle  string `mapstructure:"dedicated_single"`
	DedicatedCluster string `mapstructure:"dedicated_cluster"`
	VL               string `mapstructure:"vl"`
	Grafana          string `mapstructure:"grafana"`
	N9EEdge          string `mapstructure:"n9e_edge"`
}

// OrchestrationConfig Helm/K8s 租户编排（§2.6）。
type OrchestrationConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// NamespacePrefix 租户命名空间前缀，默认 ops，形如 ops-{uuid_compact}。
	NamespacePrefix string `mapstructure:"namespace_prefix"`
	// ApplyNetworkPolicy 为 true 时创建默认同命名空间互通 NetworkPolicy。
	ApplyNetworkPolicy bool `mapstructure:"apply_network_policy"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	Burst             int     `mapstructure:"burst"`
}

// NotificationConfig 通知渠道配置（§3.3）。
type NotificationConfig struct {
	Email EmailNotifyConfig `mapstructure:"email"`
}

// EmailNotifyConfig SMTP 邮件配置。
type EmailNotifyConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

// VMConfig VictoriaMetrics / vmauth 相关（§2.3）。
type VMConfig struct {
	// VMAuthBaseURL vmauth 对外 HTTP 根地址，用于拼接 remote_write 路径前缀。
	VMAuthBaseURL string `mapstructure:"vmauth_base_url"`
	// SyncEnabled 为 true 且配置了 Webhook 时，租户创建/删除会向 Webhook POST JSON。
	SyncEnabled bool `mapstructure:"sync_enabled"`
	// VMAuthWebhookURL 可选；接收租户 vmuser 同步事件（侧车或自定义服务）。
	VMAuthWebhookURL string `mapstructure:"vmauth_webhook_url"`
	// HTTPTimeoutSeconds Webhook 请求超时。
	HTTPTimeoutSeconds int `mapstructure:"http_timeout_seconds"`
}

// N9EConfig 夜莺 / N9E（§2.4）。
type N9EConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// BaseURL 中心地址，如 http://n9e.platform:18000
	BaseURL string `mapstructure:"base_url"`
	// Username / Password 用于登录换 Token；若填写 Token 则跳过登录。
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Token    string `mapstructure:"token"`
	// APIPrefix 夜莺 API 前缀，默认 /api/n9e。
	APIPrefix          string `mapstructure:"api_prefix"`
	HTTPTimeoutSeconds int    `mapstructure:"http_timeout_seconds"`
	// PrometheusDatasourceURL 写入 N9E 的 Prometheus 类数据源地址（如 VM select）。
	PrometheusDatasourceURL string `mapstructure:"prometheus_datasource_url"`
}

// GrafanaConfig Grafana（§2.5）。
type GrafanaConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// BaseURL 如 http://grafana.platform:3000
	BaseURL string `mapstructure:"base_url"`
	// APIKey 服务账号 API Key（Viewer/Editor/Admin 依实际权限）。
	APIKey             string `mapstructure:"api_key"`
	HTTPTimeoutSeconds int    `mapstructure:"http_timeout_seconds"`
	// PrometheusDatasourceURL 租户组织内默认 Prometheus 数据源（如 VM select）。
	PrometheusDatasourceURL string `mapstructure:"prometheus_datasource_url"`
	// OrgNamePrefix 若非空，组织名 = 前缀 + vmuser_id，否则使用租户名。
	OrgNamePrefix string `mapstructure:"org_name_prefix"`
}

// Load 从配置文件加载；支持环境变量覆盖（前缀 OPS_，例如 OPS_SERVER_PORT）。
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
		v.SetConfigName("config")
	}

	v.SetEnvPrefix("OPS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	expandPlaceholders(&cfg)

	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.JWT.ExpireHours == 0 {
		cfg.JWT.ExpireHours = 24
	}
	if strings.TrimSpace(cfg.JWT.Secret) == "" {
		return nil, fmt.Errorf("jwt.secret is required; set OPS_JWT_SECRET or jwt.secret")
	}
	if cfg.Server.Mode == "release" && len(cfg.CORS.AllowedOrigins) == 0 {
		return nil, fmt.Errorf("cors.allowed_origins is required in release mode")
	}
	if cfg.RateLimit.RequestsPerSecond <= 0 {
		cfg.RateLimit.RequestsPerSecond = 100
	}
	if cfg.RateLimit.Burst <= 0 {
		cfg.RateLimit.Burst = 200
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.MaxOpenConns <= 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns <= 0 {
		cfg.Database.MaxIdleConns = 10
	}
	if cfg.Database.ConnMaxLifetimeMinutes <= 0 {
		cfg.Database.ConnMaxLifetimeMinutes = 5
	}
	if cfg.VM.HTTPTimeoutSeconds <= 0 {
		cfg.VM.HTTPTimeoutSeconds = 15
	}
	if cfg.N9E.HTTPTimeoutSeconds <= 0 {
		cfg.N9E.HTTPTimeoutSeconds = 20
	}
	if cfg.Grafana.HTTPTimeoutSeconds <= 0 {
		cfg.Grafana.HTTPTimeoutSeconds = 30
	}
	if cfg.Orchestration.NamespacePrefix == "" {
		cfg.Orchestration.NamespacePrefix = "ops"
	}

	return &cfg, nil
}

func expandPlaceholders(cfg *Config) {
	cfg.Database.Password = os.ExpandEnv(cfg.Database.Password)
	cfg.JWT.Secret = os.ExpandEnv(cfg.JWT.Secret)
	cfg.VM.VMAuthBaseURL = os.ExpandEnv(cfg.VM.VMAuthBaseURL)
	cfg.VM.VMAuthWebhookURL = os.ExpandEnv(cfg.VM.VMAuthWebhookURL)
	cfg.N9E.Password = os.ExpandEnv(cfg.N9E.Password)
	cfg.N9E.Token = os.ExpandEnv(cfg.N9E.Token)
	cfg.N9E.BaseURL = os.ExpandEnv(cfg.N9E.BaseURL)
	cfg.N9E.PrometheusDatasourceURL = os.ExpandEnv(cfg.N9E.PrometheusDatasourceURL)
	cfg.Grafana.APIKey = os.ExpandEnv(cfg.Grafana.APIKey)
	cfg.Grafana.BaseURL = os.ExpandEnv(cfg.Grafana.BaseURL)
	cfg.Grafana.PrometheusDatasourceURL = os.ExpandEnv(cfg.Grafana.PrometheusDatasourceURL)
	cfg.Kubernetes.Kubeconfig = os.ExpandEnv(cfg.Kubernetes.Kubeconfig)
	cfg.Notification.Email.Username = os.ExpandEnv(cfg.Notification.Email.Username)
	cfg.Notification.Email.Password = os.ExpandEnv(cfg.Notification.Email.Password)
}
