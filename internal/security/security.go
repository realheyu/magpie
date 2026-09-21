package security

import (
	"crypto/subtle"
	"fmt"
	"strings"

	btcrypto "github.com/bt-smart/btutil/crypto"
	"github.com/bt-smart/btutil/strutil"
)

const apiKeyPrefix = "mgp_"

func HashPassword(password string) (passwordHash string, salt string, err error) {
	return btcrypto.GetPasswordAndSalt(password)
}

func VerifyPassword(password, passwordHash, salt string) bool {
	candidate := btcrypto.Sha256PasswordWithSalt(password, salt)
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(passwordHash)) == 1
}

func NewAPIKey() (string, error) {
	rand, err := strutil.GenerateRandomString(48, strutil.AllLettersAndDigits)
	if err != nil {
		return "", err
	}
	return apiKeyPrefix + rand, nil
}

func HashAPIKey(apiKey string) string {
	return btcrypto.Sha256(strings.TrimSpace(apiKey))
}

func APIKeyPreview(apiKey string) string {
	if len(apiKey) <= 12 {
		return apiKey
	}
	return fmt.Sprintf("%s...%s", apiKey[:8], apiKey[len(apiKey)-4:])
}

// BearerToken 从 Authorization 请求头的值里解析 Bearer token，缺失或格式不对返回空串。
func BearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
