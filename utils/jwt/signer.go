package jwt

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

var (
	pub  *ecdsa.PublicKey
	priv *ecdsa.PrivateKey
)

// SetupSigner JWTを発行・検証するためのSignerのセットアップ
func SetupSigner(privRaw []byte) error {
	_priv, err := jwt.ParseECPrivateKeyFromPEM(bytes.TrimSpace(privRaw))
	if err != nil {
		return err
	}

	pub = &_priv.PublicKey
	priv = _priv
	return nil
}

// Sign JWTの発行を行う
func Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(priv)
}

// Verify JWTの検証を行う
func Verify(tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	if err != nil {
		return fmt.Errorf("failed to parse token: %v", err)
	}
	if !token.Valid {
		return fmt.Errorf("invalid token")
	}
	return nil
}
