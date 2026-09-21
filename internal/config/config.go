package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server    ServerConfig    `toml:"server"`
	MySQL     MySQLConfig     `toml:"mysql"`
	Bootstrap BootstrapConfig `toml:"bootstrap"`
	Session   SessionConfig   `toml:"session"`
	Log       LogConfig       `toml:"log"`
}

type ServerConfig struct {
	AdminAddr string `toml:"adminAddr"`
	APIAddr   string `toml:"apiAddr"`
	GinMode   string `toml:"ginMode"`
	// WebBasePath 是管理控制台的部署子路径（如 nginx 反代时的 "/magpie"），
	// 根路径部署留空。用于向后端返回的 index.html 注入前端路由与 API 请求的基础路径。
	WebBasePath string `toml:"webBasePath"`
}

type MySQLConfig struct {
	// 标准 DSN 一行式：user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=UTC
	Url string `toml:"url"`
}

type BootstrapConfig struct {
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type SessionConfig struct {
	TTLHours int `toml:"ttlHours"`
}

type LogConfig struct {
	Level      string        `toml:"level"`
	Console    bool          `toml:"console"`
	GinRequest bool          `toml:"ginRequest"`
	File       LogFileConfig `toml:"file"`
}

type LogFileConfig struct {
	Enabled    bool   `toml:"enabled"`
	Filename   string `toml:"filename"`
	MaxSizeMB  int    `toml:"maxSizeMb"`
	MaxBackups int    `toml:"maxBackups"`
	MaxAgeDays int    `toml:"maxAgeDays"`
	Compress   bool   `toml:"compress"`
}

// Load 读取配置文件（路径取自 MAGPIE_CONFIG_FILE 环境变量，默认 config.toml），
// 文件不存在时使用 Default 提供的默认值。
func Load() (Config, error) {
	path := os.Getenv("MAGPIE_CONFIG_FILE")
	if path == "" {
		path = "config.toml"
	}
	cfg := Default()
	if err := loadTOML(path, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			AdminAddr: ":6080",
			APIAddr:   ":6081",
			GinMode:   "release",
		},
		MySQL: MySQLConfig{
			Url: "magpie_user:password@tcp(127.0.0.1:3306)/magpie?charset=utf8mb4&parseTime=True&loc=UTC",
		},
		Bootstrap: BootstrapConfig{
			Username: "admin",
			Password: "change-me",
		},
		Session: SessionConfig{
			TTLHours: 24,
		},
		Log: LogConfig{
			Level:      "info",
			Console:    true,
			GinRequest: false,
			File: LogFileConfig{
				Enabled:    true,
				Filename:   "logs/magpie.log",
				MaxSizeMB:  100,
				MaxBackups: 10,
				MaxAgeDays: 30,
				Compress:   true,
			},
		},
	}
}

func loadTOML(path string, cfg *Config) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return fmt.Errorf("解析 TOML 配置文件失败：%w", err)
	}
	return nil
}

func (c Config) SessionTTL() time.Duration {
	if c.Session.TTLHours <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(c.Session.TTLHours) * time.Hour
}

// WebBasePathSlash 返回规范化的部署基础路径（带首尾斜杠）：根路径部署为 "/"，
// 子路径部署如 "/magpie/"。非法配置按根路径处理。
func (c Config) WebBasePathSlash() string {
	p := strings.Trim(strings.TrimSpace(c.Server.WebBasePath), "/")
	if p == "" {
		return "/"
	}
	return "/" + p + "/"
}
