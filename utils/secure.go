package utils

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"golang.org/x/crypto/pbkdf2"
)

// HashPassword パスワードをハッシュ化します
func HashPassword(pass string, salt []byte) []byte {
	return pbkdf2.Key([]byte(pass), salt, 65536, 64, sha512.New)[:]
}

// CalcHMACSHA1 HMAC-SHA-1を計算します
func CalcHMACSHA1(data []byte, secret string) []byte {
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

// CalcHMACSHA256 HMAC-SHA-256を計算します
func CalcHMACSHA256(data []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}
