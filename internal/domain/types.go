package domain

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	StatusActive   = "active"
	StatusDisabled = "disabled"

	PermissionFull   = "full"
	PermissionMasked = "masked"

	FormatText       = "text"
	FormatYAML       = "yaml"
	FormatJSON       = "json"
	FormatTOML       = "toml"
	FormatProperties = "properties"
)

func NormalizeFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "":
		return FormatTOML
	case FormatYAML, "yml":
		return FormatYAML
	case FormatJSON:
		return FormatJSON
	case FormatTOML:
		return FormatTOML
	case FormatProperties, "props":
		return FormatProperties
	case FormatText:
		return FormatText
	default:
		return FormatTOML
	}
}

func SupportedFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", FormatTOML, FormatJSON, FormatYAML, "yml", FormatProperties, "props", FormatText:
		return true
	default:
		return false
	}
}

func ResolveFormat(format string) (string, error) {
	if !SupportedFormat(format) {
		return "", fmt.Errorf("不支持的配置格式：%s", strings.TrimSpace(format))
	}
	return NormalizeFormat(format), nil
}

func ValidPermission(permission string) bool {
	return permission == PermissionFull || permission == PermissionMasked
}

func CanEdit(permission string) bool {
	return permission == PermissionFull
}

func ValidateContent(format string, content string) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	switch NormalizeFormat(format) {
	case FormatTOML:
		return validateTOMLContent(content)
	case FormatJSON:
		return validateJSONContent(content)
	case FormatYAML:
		return validateYAMLContent(content)
	case FormatProperties:
		return validatePropertiesContent(content)
	case FormatText:
		return nil
	default:
		return fmt.Errorf("不支持的配置格式：%s", strings.TrimSpace(format))
	}
}

func validateTOMLContent(content string) error {
	var parsed map[string]any
	if _, err := toml.Decode(content, &parsed); err != nil {
		return fmt.Errorf("TOML 格式错误：%w", err)
	}
	return nil
}

func validateJSONContent(content string) error {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()
	var parsed any
	if err := decoder.Decode(&parsed); err != nil {
		return fmt.Errorf("JSON 格式错误：%w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("JSON 格式错误：包含多个 JSON 文档")
	}
	return nil
}

func validateYAMLContent(content string) error {
	var parsed any
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		return fmt.Errorf("YAML 格式错误：%w", err)
	}
	return nil
}

func validatePropertiesContent(content string) error {
	for lineNumber, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
			continue
		}
		separator := firstUnescapedSeparator(line)
		if separator < 0 {
			return fmt.Errorf("第 %d 行缺少 = 或 : 分隔符", lineNumber+1)
		}
		if strings.TrimSpace(line[:separator]) == "" {
			return fmt.Errorf("第 %d 行 key 不能为空", lineNumber+1)
		}
	}
	return nil
}

func firstUnescapedSeparator(line string) int {
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '=' || r == ':' {
			return i
		}
	}
	return -1
}

func MaskContent(content string) string {
	if content == "" {
		return ""
	}
	return "******"
}

func ContentType(format string) string {
	switch NormalizeFormat(format) {
	case FormatYAML:
		return "application/yaml; charset=utf-8"
	case FormatJSON:
		return "application/json; charset=utf-8"
	case FormatTOML:
		return "application/toml; charset=utf-8"
	case FormatProperties:
		return "text/plain; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}
