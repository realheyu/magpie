package configapi

import (
	"fmt"
	"strings"

	btcrypto "github.com/bt-smart/btutil/crypto"
)

func buildETag(appName string, version int64, content string) string {
	hash := btcrypto.Sha256(fmt.Sprintf("%s:%d:%s", appName, version, content))
	return fmt.Sprintf("\"%s-%d-%s\"", sanitizeETagPart(appName), version, hash[:12])
}

func sanitizeETagPart(value string) string {
	value = strings.ReplaceAll(value, "\"", "")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
