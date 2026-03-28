package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置（对应 configs/config.yaml）。
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Redis       RedisConfig       `mapstructure:"redis"`
	Kubernetes  KubernetesConfig `mapstructure:"kubernetes"`
	Helm        HelmConfig        `mapstructure:"helm"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	RateLimit   RateLimitConfig   `mapstructure:"ratelimit"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug | release | test
}

type DatabaseConfig struct {
	Host                    string `mapstructure:"host"`
	Port                    int    `mapstructure:"port"`
	User                    string `mapstructure:"user"`
	Password                string `mapstructure:"password"`
	Name                    string `mapstructure:"name"`
	SSLMode                 string `mapstructure:"sslmode"`
	MaxOpenConns            int    `mapstructure:"max_open_conns"`
	MaxIdleConns            int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeMinutes  int    `mapstructure:"conn_max_lifetime_minutes"`
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
	Repos []HelmRepo `mapstructure:"repos"`
}

type JWTConfig struct {
	Secret       string `mapstructure:"secret"`
	ExpireHours  int    `mapstructure:"expire_hours"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	Burst             int     `mapstructure:"burst"`
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

	return &cfg, nil
}

func expandPlaceholders(cfg *Config) {
	cfg.Database.Password = os.ExpandEnv(cfg.Database.Password)
	cfg.JWT.Secret = os.ExpandEnv(cfg.JWT.Secret)
}
