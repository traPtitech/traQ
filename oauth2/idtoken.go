package oauth2

import (
	"crypto/rsa"
	"errors"
	"github.com/dgrijalva/jwt-go"
)

var (
	// ErrInvalidIDToken : OpenID Connectエラー 不正なIDトークンです
	ErrInvalidIDToken = errors.New("invalid token")
)

// IDToken : OpenID Connect IDToken
type IDToken struct {
	jwt.StandardClaims

	Nonce string `json:"nonce,omitempty"`

	Name string `json:"name,omitempty"`
}

// Generate : IDTokenからJWTを生成します
func (t *IDToken) Generate(key *rsa.PrivateKey) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodRS256, t).SignedString(key)
}

// VerifyToken : 与えられたJWTが有効かどうかを確認します
func VerifyToken(token string, publicKey *rsa.PublicKey) (*jwt.Token, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidIDToken
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !parsedToken.Valid {
		return nil, ErrInvalidIDToken
	}

	return parsedToken, nil
}
