package hmac

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		assert.Equal(t, v.Expected, SHA1(v.Data, v.Secret))
	}
}

func TestCalcSHA256Signature(t *testing.T) {
	t.Parallel()

	// test cases from https://tools.ietf.org/html/rfc4231#section-4
	cases := []struct {
		Secret   string
		Data     []byte
		Expected []byte
	}{
		{
			Secret:   string(mustHexDecode("0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b")),
			Data:     []byte("Hi There"),
			Expected: mustHexDecode("b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7"),
		},
		{
			Secret:   "Jefe",
			Data:     []byte("what do ya want for nothing?"),
			Expected: mustHexDecode("5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843"),
		},
		{
			Secret:   string(mustHexDecode("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")),
			Data:     mustHexDecode(strings.Repeat("dd", 50)),
			Expected: mustHexDecode("773ea91e36800e46854db8ebd09181a72959098b3ef8c122d9635514ced565fe"),
		},
		{
			Secret:   string(mustHexDecode("0102030405060708090a0b0c0d0e0f10111213141516171819")),
			Data:     mustHexDecode(strings.Repeat("cd", 50)),
			Expected: mustHexDecode("82558a389a443c0ea4cc819899f2083a85f0faa3e578f8077a2e3ff46729665b"),
		},
	}
	for _, v := range cases {
		assert.Equal(t, v.Expected, SHA256(v.Data, v.Secret))
	}
}

func mustHexDecode(h string) []byte {
	b, _ := hex.DecodeString(h)
	return b
}
