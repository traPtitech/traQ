package jwt

import (
	"bytes"
	"crypto/ecdsa"

	"github.com/golang-jwt/jwt/v5"
)

var priv *ecdsa.PrivateKey

// SetupSigner JWTを発行・検証するためのSignerのセットアップ
func SetupSigner(privRaw []byte) error {
	_priv, err := jwt.ParseECPrivateKeyFromPEM(bytes.TrimSpace(privRaw))
	if err != nil {
		return err
	}

	priv = _priv
	return nil
}

// Sign JWTの発行を行う
func Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(priv)
}
