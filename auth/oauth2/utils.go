package oauth2

import (
	"encoding/base64"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/scope"
	"strings"
)

var (
	store Store

	// ErrInvalidScope : OAuth2エラー 不正なスコープです
	ErrInvalidScope = &errorResponse{ErrorType: errInvalidScope}
)

// AuthScheme : Authorizationヘッダーのスキーム
const AuthScheme = "Bearer"

func generateRandomString() string {
	return base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes())
}

// SetOAuth2Store : OAuth2のストアの実装をセットします
func SetOAuth2Store(s Store) {
	store = s
}

// GetOAuth2Store : OAuth2のストアを返します
func GetOAuth2Store() Store {
	return store
}

// SplitAndValidateScope : スペース区切りのスコープ文字列を分解し、検証します
func SplitAndValidateScope(str string) (scope.AccessScopes, error) {
	var scopes scope.AccessScopes
	set := map[scope.AccessScope]struct{}{}

	for _, v := range strings.Fields(str) {
		s := scope.AccessScope(v)
		if _, ok := set[s]; !scope.Valid(s) || ok {
			return nil, ErrInvalidScope
		}
		scopes = append(scopes, s)
		set[s] = struct{}{}
	}

	return scopes, nil
}
