package utils

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

var (
	// Signer JWTを発行・検証するための構造体
	Signer *signer
)

type (
	signer struct {
		pub  *ecdsa.PublicKey
		priv *ecdsa.PrivateKey
	}
)

// SetupSigner JWTを発行・検証するためのSignerのセットアップ
func SetupSigner(pubRaw, privRaw []byte) error {
	pub, err := jwt.ParseECPublicKeyFromPEM(bytes.TrimSpace(pubRaw))
	if err != nil {
		return err
	}

	priv, err := jwt.ParseECPrivateKeyFromPEM(bytes.TrimSpace(privRaw))
	if err != nil {
		return err
	}

	Signer = &signer{pub, priv}
	return nil
}

// Sign JWTの発行を行う
func (s *signer) Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(s.priv)
}

// Verify JWTの検証を行う
func (s *signer) Verify(tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.pub, nil
	})
	if err != nil {
		return fmt.Errorf("failed to parse token: %v", err)
	}
	if !token.Valid {
		return fmt.Errorf("invalid token")
	}
	return nil
}
