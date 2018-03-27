package openid

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"time"
)

var (
	// ErrInvalidIDToken : OpenID Connectエラー 不正なIDトークンです
	ErrInvalidIDToken = errors.New("invalid token")
)

// IDToken : OpenID Connect IdToken
type IDToken struct {
	jwt.StandardClaims

	Nonce string `json:"nonce,omitempty"`

	Name string `json:"name,omitempty"`
}

// NewIDToken : 新しくIDTokenを生成します
func NewIDToken(issueAt time.Time, expireIn int64) *IDToken {
	return &IDToken{
		StandardClaims: jwt.StandardClaims{
			Issuer:    issuer,
			IssuedAt:  issueAt.Unix(),
			ExpiresAt: issueAt.Unix() + expireIn,
		},
	}
}

// Generate : IDTokenからJWTを生成します
func (t *IDToken) Generate() (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodRS256, t).SignedString(privateKey)
}

// VerifyToken : 与えられたJWTが有効かどうかを確認します
func VerifyToken(token string) (*jwt.Token, error) {
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
