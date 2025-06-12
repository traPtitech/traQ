package random

import (
	crand "crypto/rand"
	"io"
	"math/rand/v2"
	"unsafe"
)

const (
	rs6Letters       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	rs6LetterIdxBits = 6
	rs6LetterIdxMask = 1<<rs6LetterIdxBits - 1
	rs6LetterIdxMax  = 63 / rs6LetterIdxBits
)

// AlphaNumeric 指定した文字数のランダム英数字文字列を生成します
// この関数はmath/randが生成する擬似乱数を使用します
func AlphaNumeric(n int) string {
	b := make([]byte, n)
	cache, remain := rand.Int64(), rs6LetterIdxMax
	for i := n - 1; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int64(), rs6LetterIdxMax
		}
		idx := int(cache & rs6LetterIdxMask)
		if idx < len(rs6Letters) {
			b[i] = rs6Letters[idx]
			i--
		}
		cache >>= rs6LetterIdxBits
		remain--
	}
	return *(*string)(unsafe.Pointer(&b))
}

// SecureAlphaNumeric 指定した文字数のランダム英数字文字列を生成します
// この関数はcrypto/randが生成する暗号学的に安全な乱数を使用します
func SecureAlphaNumeric(n int) string {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		panic(err)
	}
	for i := 0; i < n; {
		idx := int(b[i] & rs6LetterIdxMask)
		if idx < len(rs6Letters) {
			b[i] = rs6Letters[idx]
			i++
		} else {
			if _, err := crand.Read(b[i : i+1]); err != nil {
				panic(err)
			}
		}
	}
	return *(*string)(unsafe.Pointer(&b))
}

// Salt 64bytesソルトを生成します
func Salt() []byte {
	salt := make([]byte, 64)
	_, _ = io.ReadFull(crand.Reader, salt)
	return salt
}
