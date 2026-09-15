package domain

import "testing"

func TestPermissionAndMasking(t *testing.T) {
	if !ValidPermission(PermissionFull) || !ValidPermission(PermissionMasked) {
		t.Fatal("expected built-in permissions to be valid")
	}
	if ValidPermission("hidden") {
		t.Fatal("hidden is represented by missing permission, not a stored value")
	}
	if MaskContent("secret") != "******" {
		t.Fatal("expected non-empty content to be masked")
	}
	if MaskContent("") != "" {
		t.Fatal("expected empty content to stay empty")
	}
}

func TestValidRoleAndStatus(t *testing.T) {
	if !ValidRole(RoleAdmin) || !ValidRole(RoleUser) {
		t.Fatal("built-in roles should be valid")
	}
	if ValidRole("superuser") {
		t.Fatal("unknown role should be invalid")
	}
	if !ValidStatus(StatusActive) || !ValidStatus(StatusDisabled) {
		t.Fatal("built-in statuses should be valid")
	}
	if ValidStatus("disable") {
		t.Fatal("misspelled status should be invalid")
	}
}

func TestValidAppName(t *testing.T) {
	for _, name := range []string{"app", "a", "app2-prod", "a_b-c123"} {
		if !ValidAppName(name) {
			t.Fatalf("expected %q to be valid", name)
		}
	}
	for _, name := range []string{"", "1app", "App", "app name", "应用", "-app", "_app", "app~name", "app/name"} {
		if ValidAppName(name) {
			t.Fatalf("expected %q to be invalid", name)
		}
	}
}

func TestNormalizeFormatDefaultsToTOML(t *testing.T) {
	if NormalizeFormat("") != FormatTOML {
		t.Fatal("empty format should default to toml")
	}
	if NormalizeFormat("yml") != FormatYAML {
		t.Fatal("yml should normalize to yaml")
	}
}

func TestResolveFormatRejectsUnsupportedFormats(t *testing.T) {
	if _, err := ResolveFormat("xml"); err == nil {
		t.Fatal("unsupported format should fail")
	}
}

func TestValidateContentChecksAllStructuredFormats(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		valid   string
		invalid string
	}{
		{name: "toml", format: FormatTOML, valid: "[server]\nport = 8080\n", invalid: "[server\nport = 8080\n"},
		{name: "json", format: FormatJSON, valid: `{"server":{"port":8080}}`, invalid: `{"server":`},
		{name: "yaml", format: FormatYAML, valid: "server:\n  port: 8080\n", invalid: "server:\n  port: [\n"},
		{name: "properties", format: FormatProperties, valid: "server.port=8080\nfeature.enabled:true\n", invalid: "server.port 8080\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateContent(tt.format, tt.valid); err != nil {
				t.Fatalf("valid %s rejected: %v", tt.format, err)
			}
			if err := ValidateContent(tt.format, tt.invalid); err == nil {
				t.Fatalf("invalid %s should fail", tt.format)
			}
		})
	}
}

func TestValidateContentAllowsPlainText(t *testing.T) {
	if err := ValidateContent(FormatText, "[server\nport = 8080\n"); err != nil {
		t.Fatalf("plain text should not use structured validation: %v", err)
	}
}
