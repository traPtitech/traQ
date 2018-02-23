package openid

import (
	"crypto/rsa"
	"github.com/dgrijalva/jwt-go"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

// LoadKeys : OpenID Connectのjwt用のRSA秘密鍵・公開鍵を読み込みます
func LoadKeys(private, public []byte) error {
	var err error
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		return err
	}
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(public)
	if err != nil {
		return err
	}
	return nil
}
