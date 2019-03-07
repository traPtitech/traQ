package utils

import (
	"crypto/hmac"
	"crypto/sha1"
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
	mac.Write(data)
	return mac.Sum(nil)
}
