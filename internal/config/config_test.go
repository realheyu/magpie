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
url = "toml-user:toml-password@tcp(mysql.local:3306)/magpie_toml?charset=utf8mb4&parseTime=True&loc=UTC"

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
	t.Setenv("MAGPIE_MYSQL_URL", "env-user:env-password@tcp(env-mysql:3306)/magpie_env?charset=utf8mb4&parseTime=True&loc=UTC")
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
	if cfg.MySQL.Url != "env-user:env-password@tcp(env-mysql:3306)/magpie_env?charset=utf8mb4&parseTime=True&loc=UTC" {
		t.Fatalf("env url should override TOML url, got %q", cfg.MySQL.Url)
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
	if !strings.Contains(cfg.MySQL.Url, "loc=UTC") {
		t.Fatalf("default url should use UTC loc: %s", cfg.MySQL.Url)
	}
}

func TestLoadIgnoresMissingConfigFile(t *testing.T) {
	t.Setenv("MAGPIE_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.toml"))
	cfg, err := Load()
	if err != nil {
		t.Fatalf("missing config file should use defaults: %v", err)
	}
	if cfg.Server.AdminAddr != ":6030" || cfg.MySQL.Url != Default().MySQL.Url {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestWebBasePathSlash(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"magpie", "/magpie/"},
		{"/magpie", "/magpie/"},
		{"/magpie/", "/magpie/"},
		{" /a/b/ ", "/a/b/"},
	}
	for _, c := range cases {
		cfg := Default()
		cfg.Server.WebBasePath = c.in
		if got := cfg.WebBasePathSlash(); got != c.want {
			t.Fatalf("WebBasePathSlash(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWebBasePathEnvOverride(t *testing.T) {
	t.Setenv("MAGPIE_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.toml"))
	t.Setenv("MAGPIE_WEB_BASE_PATH", "/magpie")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.WebBasePathSlash(); got != "/magpie/" {
		t.Fatalf("web base path env override failed, got %q", got)
	}
}
