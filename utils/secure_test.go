package utils

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	password1 := "test"
	password2 := "testtest"
	salt1 := GenerateSalt()
	salt2 := GenerateSalt()

	assert.EqualValues(t, HashPassword(password1, salt1), HashPassword(password1, salt1))
	assert.NotEqual(t, HashPassword(password1, salt1), HashPassword(password1, salt2))
	assert.NotEqual(t, HashPassword(password2, salt1), HashPassword(password1, salt1))
}

func TestCalcSHA1Signature(t *testing.T) {
	t.Parallel()

	// test cases from https://www.ipa.go.jp/security/rfc/RFC2202JA.html
	cases := []struct {
		Secret   string
		Data     []byte
		Expected []byte
	}{
		{
			Secret:   string(mustHexDecode("0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b")),
			Data:     []byte("Hi There"),
			Expected: mustHexDecode("b617318655057264e28bc0b6fb378c8ef146be00"),
		},
		{
			Secret:   "Jefe",
			Data:     []byte("what do ya want for nothing?"),
			Expected: mustHexDecode("effcdf6ae5eb2fa2d27416d5f184df9c259a7c79"),
		},
		{
			Secret:   string(mustHexDecode("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")),
			Data:     mustHexDecode(strings.Repeat("dd", 50)),
			Expected: mustHexDecode("125d7342b9ac11cd91a39af48aa17b4f63f175d3"),
		},
		{
			Secret:   string(mustHexDecode("0102030405060708090a0b0c0d0e0f10111213141516171819")),
			Data:     mustHexDecode(strings.Repeat("cd", 50)),
			Expected: mustHexDecode("4c9007f4026250c6bc8414f9bf50c86c2d7235da"),
		},
		{
			Secret:   string(mustHexDecode("0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c")),
			Data:     []byte("Test With Truncation"),
			Expected: mustHexDecode("4c1a03424b55e07fe7f27be1d58bb9324a9a5a04"),
		},
		{
			Secret:   string(mustHexDecode(strings.Repeat("aa", 80))),
			Data:     []byte("Test Using Larger Than Block-Size Key - Hash Key First"),
			Expected: mustHexDecode("aa4ae5e15272d00e95705637ce8a3b55ed402112"),
		},
		{
			Secret:   string(mustHexDecode(strings.Repeat("aa", 80))),
			Data:     []byte("Test Using Larger Than Block-Size Key and Larger Than One Block-Size Data"),
			Expected: mustHexDecode("e8e99d0f45237d786d6bbaa7965c7808bbff1a91"),
		},
	}

	for _, v := range cases {
		assert.Equal(t, v.Expected, CalcHMACSHA1(v.Data, v.Secret))
	}
}

func mustHexDecode(h string) []byte {
	b, _ := hex.DecodeString(h)
	return b
}
