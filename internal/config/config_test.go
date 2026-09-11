package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTOMLAndEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[server]
adminAddr = ":18080"
apiAddr = ":18081"
ginMode = "debug"

[mysql]
addr = "mysql.local:3306"
user = "toml-user"
password = "toml-password"
database = "magpie_toml"

[bootstrap]
username = "admin"
password = "admin-password"

[session]
ttlHours = 12

[log]
level = "debug"
console = false
ginRequest = true

[log.file]
enabled = true
filename = "logs/test.log"
maxSizeMb = 12
maxBackups = 3
maxAgeDays = 7
compress = true
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	t.Setenv("MAGPIE_CONFIG_FILE", path)
	t.Setenv("MAGPIE_MYSQL_PASSWORD", "env-password")
	t.Setenv("MAGPIE_LOG_FILE_ENABLED", "false")
	t.Setenv("MAGPIE_LOG_GIN_REQUEST", "false")
	t.Setenv("MAGPIE_GIN_MODE", "release")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Server.AdminAddr != ":18080" {
		t.Fatalf("admin addr mismatch: %q", cfg.Server.AdminAddr)
	}
	if cfg.Server.GinMode != "release" {
		t.Fatalf("gin mode env should override TOML value, got %q", cfg.Server.GinMode)
	}
	if cfg.MySQL.Password != "env-password" {
		t.Fatalf("env password should override TOML password, got %q", cfg.MySQL.Password)
	}
	if cfg.Log.File.Enabled {
		t.Fatal("env bool should override TOML bool")
	}
	if cfg.Log.GinRequest {
		t.Fatal("gin request log env bool should override TOML bool")
	}
	if cfg.Log.File.MaxBackups != 3 || cfg.Log.File.MaxAgeDays != 7 {
		t.Fatalf("log rotate config mismatch: %+v", cfg.Log.File)
	}
	if !strings.Contains(cfg.MySQLDSN(), "loc=UTC") {
		t.Fatalf("dsn should use UTC: %s", cfg.MySQLDSN())
	}
}

func TestLoadIgnoresMissingConfigFile(t *testing.T) {
	t.Setenv("MAGPIE_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.toml"))
	cfg, err := Load()
	if err != nil {
		t.Fatalf("missing config file should use defaults: %v", err)
	}
	if cfg.Server.AdminAddr != ":8080" || cfg.MySQL.Database != "magpie" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}
