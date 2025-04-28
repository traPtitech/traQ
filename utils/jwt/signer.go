package jwt

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"

	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/jwk"
)

var (
	priv  *ecdsa.PrivateKey
	keyID string
	jwks  jwkset.Storage
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

	j, err := jwk.New(priv.Public().(*ecdsa.PublicKey))
	if err != nil {
		return err
	}

	thumb, err := j.Thumbprint(crypto.SHA256)
	if err != nil {
		return err
	}
	keyID = base64.RawURLEncoding.EncodeToString(thumb)

	err = j.Set(j.KeyID(), keyID)
	if err != nil {
		return err
	}

	jwk, err := jwkset.NewJWKFromKey(priv, jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			KID: keyID,
		},
	})

	if err != nil {
		return err
	}

	return jwks.KeyWrite(context.Background(), jwk)
}

// Sign JWTの発行を行う
func Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = keyID
	return token.SignedString(priv)
}

// SupportedAlgorithms サポートする signing algorithm の一覧
func SupportedAlgorithms() []string {
	return []string{jwt.SigningMethodES256.Alg()}
}

// JWKSet Public の JSON Web Key Set を取得する
func JWKSet(ctx context.Context) (json.RawMessage, error) {
	return jwks.JSONPublic(ctx)
}
