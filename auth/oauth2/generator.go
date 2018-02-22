package oauth2

import (
	"encoding/base64"
	"github.com/satori/go.uuid"
	"strings"
)

func generateRandomString() string {
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()), "=")
}

func generateAuthorizeCode() (code string) {
	return generateRandomString()
}

func generateAccessToken() (accessToken, refreshToken string) {
	return generateRandomString(), generateRandomString()
}
