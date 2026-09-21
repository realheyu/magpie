package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[server]
adminAddr = ":18080"
apiAddr = ":18081"
ginMode = "debug"
webBasePath = "/magpie"

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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Server.AdminAddr != ":18080" || cfg.Server.APIAddr != ":18081" {
		t.Fatalf("server addr mismatch: %+v", cfg.Server)
	}
	if cfg.Server.GinMode != "debug" {
		t.Fatalf("gin mode mismatch: %q", cfg.Server.GinMode)
	}
	if got := cfg.WebBasePathSlash(); got != "/magpie/" {
		t.Fatalf("web base path mismatch: %q", got)
	}
	if cfg.MySQL.Url != "toml-user:toml-password@tcp(mysql.local:3306)/magpie_toml?charset=utf8mb4&parseTime=True&loc=UTC" {
		t.Fatalf("mysql url mismatch: %q", cfg.MySQL.Url)
	}
	if cfg.Session.TTLHours != 12 || cfg.SessionTTL() != 12*time.Hour {
		t.Fatalf("session ttl mismatch: %+v", cfg.Session)
	}
	if !cfg.Log.GinRequest || cfg.Log.Console {
		t.Fatalf("log flags mismatch: %+v", cfg.Log)
	}
	if cfg.Log.File.MaxBackups != 3 || cfg.Log.File.MaxAgeDays != 7 || cfg.Log.File.MaxSizeMB != 12 {
		t.Fatalf("log rotate config mismatch: %+v", cfg.Log.File)
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
