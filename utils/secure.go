// revive:disable-next-line FIXME: https://github.com/traPtitech/traQ/issues/2717
package utils

import (
	"crypto/sha512"

	"golang.org/x/crypto/pbkdf2"
)

// HashPassword パスワードをハッシュ化します
func HashPassword(pass string, salt []byte) []byte {
	return pbkdf2.Key([]byte(pass), salt, 65536, 64, sha512.New)[:]
}
