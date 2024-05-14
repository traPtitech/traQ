package jwt

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"

	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v5"
)

var (
	priv *ecdsa.PrivateKey
	jwks jwkset.Storage
)

func init() {
	jwks = jwkset.NewMemoryStorage()
}

// SetupSigner JWTを発行・検証するためのSignerのセットアップ
func SetupSigner(privRaw []byte) error {
	_priv, err := jwt.ParseECPrivateKeyFromPEM(bytes.TrimSpace(privRaw))
	if err != nil {
		return err
	}
	priv = _priv

	jwk, err := jwkset.NewJWKFromKey(priv, jwkset.JWKOptions{})
	if err != nil {
		return err
	}
	return jwks.KeyWrite(context.Background(), jwk)
}

// Sign JWTの発行を行う
func Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(priv)
}

// SupportedAlgorithms サポートする signing algorithm の一覧
func SupportedAlgorithms() []string {
	return []string{jwt.SigningMethodES256.Alg()}
}

// JWKSet Public の JSON Web Key Set を取得する
func JWKSet(ctx context.Context) (json.RawMessage, error) {
	return jwks.JSONPublic(ctx)
}
