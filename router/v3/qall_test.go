package v3

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_GetQallEndpoints(t *testing.T) {
	t.Parallel()

	path := "/api/v3/qall/endpoints"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	client := env.CreateOAuth2Client(t, rand, user.GetID())
	commonSession := env.S(t, user.GetID())

	issueToken := func(t *testing.T, scopes ...model.AccessScope) string {
		t.Helper()
		s := model.AccessScopes{}
		s.Add(scopes...)
		tok, err := env.Repository.IssueToken(context.TODO(), client, user.GetID(), client.RedirectURI, s, 86400, false)
		require.NoError(t, err)
		return tok.AccessToken
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success (session)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("forbidden (oauth2 token without qall scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithHeader("Authorization", "Bearer "+issueToken(t, "read", "write")).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success (oauth2 token with qall scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithHeader("Authorization", "Bearer "+issueToken(t, "read", "qall")).
			Expect().
			Status(http.StatusOK)
	})
}
