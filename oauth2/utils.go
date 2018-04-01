package oauth2

import (
	"encoding/base64"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/oauth2/scope"
	"strings"
)

func generateRandomString() string {
	return base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes())
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
