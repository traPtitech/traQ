package openid

import (
	"bytes"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"net/http"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     = "https://traq-dev.herokuapp.com" //FIXME //TODO
	discovery  = map[string]interface{}{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/api/1.0/oauth2/authorize",
		"token_endpoint":                        issuer + "/api/1.0/oauth2/token",
		"jwks_uri":                              issuer + "/publickeys",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile"},
		"grantTypesSupported":                   []string{"authorization_code", "refresh_token", "client_credentials", "password"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"display_values_supported":              []string{"page"},
		"ui_locales_supported":                  []string{"ja"},
		"request_parameter_supported":           false,
		"request_uri_parameter_supported":       false,
		"claims_supported": []string{
			"aud", "exp", "iat", "iss", "name", "sub",
		},
	}
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

// DiscoveryHandler returns the OpenID Connect discovery object.
func DiscoveryHandler(c echo.Context) error {
	if Available() {
		return c.JSON(http.StatusOK, discovery)
	}
	return c.NoContent(http.StatusNotFound)
}

// PublicKeysHandler publishes the public signing keys.
func PublicKeysHandler(c echo.Context) error {
	if Available() {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(publicKey.E))

		res := map[string]interface{}{
			"keys": map[string]interface{}{
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(bytes.TrimLeft(data, "\x00")),
			},
		}

		return c.JSON(http.StatusOK, res)
	}

	return c.NoContent(http.StatusNotFound)
}

// Available : OpenID Connectが有効かどうか
func Available() bool {
	return privateKey != nil && publicKey != nil
}
