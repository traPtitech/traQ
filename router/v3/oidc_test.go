package v3

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
)

func TestHandlers_GetMeOIDCScopes(t *testing.T) {
	t.Parallel()
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	client := env.CreateOAuth2Client(t, rand, user.GetID())
	for _, tc := range []struct {
		name    string
		scopes  model.AccessScopes
		profile bool
	}{
		{name: "openid", scopes: model.AccessScopes{"openid": {}}},
		{name: "openid profile", scopes: model.AccessScopes{"openid": {}, "profile": {}}, profile: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := env.Repository.IssueToken(context.Background(), client, user.GetID(), client.RedirectURI, tc.scopes, 86400, false)
			require.NoError(t, err)
			res := env.R(t).GET("/api/v3/users/me/oidc").WithHeader("Authorization", "Bearer "+token.AccessToken).
				Expect().Status(http.StatusOK).JSON().Object()
			res.Value("sub").String().IsEqual(user.GetID().String())
			res.Value("name").String().IsEqual(user.GetName())
			if tc.profile {
				res.ContainsKey("traq").ContainsKey("updated_at")
			} else {
				res.NotContainsKey("traq").NotContainsKey("updated_at")
			}
		})
	}
}
