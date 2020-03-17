package hmac

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
)

// SHA1 HMAC-SHA-1を計算します
func SHA1(data []byte, secret string) []byte {
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

// SHA256 HMAC-SHA-256を計算します
func SHA256(data []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}
