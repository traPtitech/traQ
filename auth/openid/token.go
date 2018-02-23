package openid

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
)

// IDToken : OpenID Connect IdToken
type IDToken struct {
	jwt.StandardClaims

	Nonce string `json:"nonce,omitempty"`

	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`

	Name string `json:"name,omitempty"`
}

// Generate : IDTokenからJWTを生成します
func (t *IDToken) Generate() (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodRS256, t).SignedString(privateKey)
}

// VerifyToken : 与えられたJWTが有効かどうかを確認します
func VerifyToken(token string) (*jwt.Token, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			err := errors.New("unexpected signing method")
			return nil, err
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	return parsedToken, nil
}
