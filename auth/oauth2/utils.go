package oauth2

import (
	"encoding/base64"
	"errors"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/scope"
	"strings"
)

var store Store

func generateRandomString() string {
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()), "=")
}

func SetOAuth2Store(s Store) {
	store = s
}

func splitAndValidateScope(str string) (scope.AccessScopes, error) {
	var scopes scope.AccessScopes
	set := map[scope.AccessScope]struct{}{}

	for _, v := range strings.Fields(str) {
		s := scope.AccessScope(v)
		if _, ok := set[s]; !scope.Valid(s) || ok {
			return nil, errors.New(v)
		}
		scopes = append(scopes, s)
		set[s] = struct{}{}
	}

	return scopes, nil
}
