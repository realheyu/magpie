package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	mysqldriver "github.com/go-sql-driver/mysql"
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
}

type MySQLConfig struct {
	DSN      string `toml:"dsn"`
	Addr     string `toml:"addr"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
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

func Load() (Config, error) {
	cfg := Default()
	path := env("MAGPIE_CONFIG_FILE", "config.toml")
	if err := loadTOML(path, &cfg); err != nil {
		return Config{}, err
	}
	if err := applyEnvOverrides(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			AdminAddr: ":8080",
			APIAddr:   ":8081",
			GinMode:   "release",
		},
		MySQL: MySQLConfig{
			Addr:     "127.0.0.1:3306",
			User:     "magpie_user",
			Database: "magpie",
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
	if path == "" {
		return nil
	}
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

func applyEnvOverrides(cfg *Config) error {
	cfg.Server.AdminAddr = env("MAGPIE_ADMIN_ADDR", cfg.Server.AdminAddr)
	cfg.Server.APIAddr = env("MAGPIE_API_ADDR", cfg.Server.APIAddr)
	cfg.Server.GinMode = env("MAGPIE_GIN_MODE", cfg.Server.GinMode)
	cfg.MySQL.DSN = env("MAGPIE_MYSQL_DSN", cfg.MySQL.DSN)
	cfg.MySQL.Addr = env("MAGPIE_MYSQL_ADDR", cfg.MySQL.Addr)
	cfg.MySQL.User = env("MAGPIE_MYSQL_USER", cfg.MySQL.User)
	cfg.MySQL.Password = env("MAGPIE_MYSQL_PASSWORD", cfg.MySQL.Password)
	cfg.MySQL.Database = env("MAGPIE_MYSQL_DATABASE", cfg.MySQL.Database)
	cfg.Bootstrap.Username = env("MAGPIE_BOOTSTRAP_USERNAME", cfg.Bootstrap.Username)
	cfg.Bootstrap.Password = env("MAGPIE_BOOTSTRAP_PASSWORD", cfg.Bootstrap.Password)
	cfg.Log.Level = env("MAGPIE_LOG_LEVEL", cfg.Log.Level)
	cfg.Log.File.Filename = env("MAGPIE_LOG_FILE_FILENAME", cfg.Log.File.Filename)

	var err error
	if cfg.Session.TTLHours, err = envInt("MAGPIE_SESSION_TTL_HOURS", cfg.Session.TTLHours); err != nil {
		return err
	}
	if cfg.Log.Console, err = envBool("MAGPIE_LOG_CONSOLE", cfg.Log.Console); err != nil {
		return err
	}
	if cfg.Log.GinRequest, err = envBool("MAGPIE_LOG_GIN_REQUEST", cfg.Log.GinRequest); err != nil {
		return err
	}
	if cfg.Log.File.Enabled, err = envBool("MAGPIE_LOG_FILE_ENABLED", cfg.Log.File.Enabled); err != nil {
		return err
	}
	if cfg.Log.File.MaxSizeMB, err = envInt("MAGPIE_LOG_FILE_MAX_SIZE_MB", cfg.Log.File.MaxSizeMB); err != nil {
		return err
	}
	if cfg.Log.File.MaxBackups, err = envInt("MAGPIE_LOG_FILE_MAX_BACKUPS", cfg.Log.File.MaxBackups); err != nil {
		return err
	}
	if cfg.Log.File.MaxAgeDays, err = envInt("MAGPIE_LOG_FILE_MAX_AGE_DAYS", cfg.Log.File.MaxAgeDays); err != nil {
		return err
	}
	if cfg.Log.File.Compress, err = envBool("MAGPIE_LOG_FILE_COMPRESS", cfg.Log.File.Compress); err != nil {
		return err
	}
	return nil
}

func (c Config) MySQLDSN() string {
	if c.MySQL.DSN != "" {
		return c.MySQL.DSN
	}

	cfg := mysqldriver.NewConfig()
	cfg.User = c.MySQL.User
	cfg.Passwd = c.MySQL.Password
	cfg.Net = "tcp"
	cfg.Addr = c.MySQL.Addr
	cfg.DBName = c.MySQL.Database
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{
		"charset":   "utf8mb4",
		"loc":       "UTC",
		"time_zone": "'+00:00'",
	}
	return cfg.FormatDSN()
}

func (c Config) SessionTTL() time.Duration {
	if c.Session.TTLHours <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(c.Session.TTLHours) * time.Hour
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是整数：%w", key, err)
	}
	return n, nil
}

func envBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s 必须是布尔值：%w", key, err)
	}
	return b, nil
}
