package openid

import (
	"encoding/base64"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestIDToken_Generate(t *testing.T) {
	require.NoError(t, LoadKeys(testPrivateKey, testPublicKey))

	idt := &IDToken{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "example.com",
			Subject:   "test_subject",
			Audience:  "test_audience",
			ExpiresAt: 123456789,
		},
		Email: "example@example.com",
	}

	tokenString, err := idt.Generate()
	assert.NoError(t, err)

	encoded := strings.Split(tokenString, ".")

	headerBytes, err := base64.RawStdEncoding.DecodeString(encoded[0])
	if assert.NoError(t, err) {
		var header struct {
			Alg string `json:"alg"`
			Typ string `json:"typ"`
		}
		if assert.NoError(t, json.Unmarshal(headerBytes, &header)) {
			assert.EqualValues(t, "RS256", header.Alg)
			assert.EqualValues(t, "JWT", header.Typ)
		}
	}

	claimsBytes, err := base64.RawStdEncoding.DecodeString(encoded[1])
	if assert.NoError(t, err) {
		var claims struct {
			Iss   string `json:"iss"`
			Sub   string `json:"sub"`
			Aud   string `json:"aud"`
			Exp   int64  `json:"exp"`
			Email string `json:"email"`
		}
		if assert.NoError(t, json.Unmarshal(claimsBytes, &claims)) {
			assert.EqualValues(t, "example.com", claims.Iss)
			assert.EqualValues(t, "test_subject", claims.Sub)
			assert.EqualValues(t, "test_audience", claims.Aud)
			assert.EqualValues(t, 123456789, claims.Exp)
			assert.EqualValues(t, "example@example.com", claims.Email)
		}
	}
}

func TestVerifyToken(t *testing.T) {
	require.NoError(t, LoadKeys(testPrivateKey, testPublicKey))

	idt := &IDToken{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "example.com",
			Subject:   "test_subject",
			Audience:  "test_audience",
			ExpiresAt: time.Now().Unix() - 1000,
		},
		Email: "example@example.com",
	}
	if token, err := idt.Generate(); assert.NoError(t, err) {
		_, err := VerifyToken(token)
		assert.Error(t, err)
	}

	idt = &IDToken{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "example.com",
			Subject:   "test_subject",
			Audience:  "test_audience",
			ExpiresAt: time.Now().Unix() + 6000,
		},
		Email: "example@example.com",
	}
	if token, err := idt.Generate(); assert.NoError(t, err) {
		parsed, err := VerifyToken(token)
		if assert.NoError(t, err) {
			claims, _ := parsed.Claims.(jwt.MapClaims)
			assert.EqualValues(t, idt.Issuer, claims["iss"])
			assert.EqualValues(t, idt.Subject, claims["sub"])
			assert.EqualValues(t, idt.Audience, claims["aud"])
			assert.EqualValues(t, idt.ExpiresAt, claims["exp"])
			assert.EqualValues(t, idt.Email, claims["email"])
		}
	}

	idt = &IDToken{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "example.com",
			Subject:   "test_wrong_subject",
			Audience:  "test_audience",
			ExpiresAt: time.Now().Unix() + 6000,
		},
		Email: "example@example.com",
	}
	if token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, idt).SignedString(testPublicKey); assert.NoError(t, err) {
		_, err := VerifyToken(token)
		assert.Error(t, err)
	}
	if token, err := jwt.NewWithClaims(jwt.SigningMethodNone, idt).SigningString(); assert.NoError(t, err) {
		_, err := VerifyToken(token)
		assert.Error(t, err)
	}
}
