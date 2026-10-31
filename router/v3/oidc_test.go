package v3

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
)

func TestHandlers_GetMeOIDCWithOpenIDScope(t *testing.T) {
	t.Parallel()
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	client := env.CreateOAuth2Client(t, rand, user.GetID())
	token, err := env.Repository.IssueToken(context.Background(), client, user.GetID(), client.RedirectURI, model.AccessScopes{"openid": {}}, 86400, false)
	require.NoError(t, err)
	env.R(t).GET("/api/v3/users/me/oidc").WithHeader("Authorization", "Bearer "+token.AccessToken).
		Expect().Status(http.StatusOK).JSON().Object().Value("sub").String().IsEqual(user.GetID().String())
}
