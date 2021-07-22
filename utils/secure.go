package utils

import (
	"crypto/sha512"

	"golang.org/x/crypto/pbkdf2"
)

// HashPassword パスワードをハッシュ化します
func HashPassword(pass string, salt []byte) []byte {
	return pbkdf2.Key([]byte(pass), salt, 65536, 64, sha512.New)[:]
}
