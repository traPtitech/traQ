package oauth2

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/utils/random"
)

func TestAuthorizationRedirectQuery(t *testing.T) {
	t.Parallel()
	env := Setup(t, db2)
	user := env.CreateUser(t, rand, false)
	client := &model.OAuth2Client{
		ID: random.AlphaNumeric(36), Name: "query client", CreatorID: user.GetID(),
		RedirectURI: "https://example.com/callback?app=one&app=two&state=old",
		Scopes:      model.AccessScopes{"read": {}},
	}
	require.NoError(t, env.Repository.SaveClient(context.Background(), client))
	env.IssueToken(t, client, user.GetID(), false)

	for _, tc := range []struct {
		name, prompt, challengeMethod, wantError string
		consent                                  bool
	}{
		{name: "success", prompt: "none"},
		{name: "error", prompt: "none", challengeMethod: "invalid", wantError: errInvalidRequest},
		{name: "consent", consent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := env.R(t).POST("/oauth2/authorize").
				WithFormField("client_id", client.ID).
				WithFormField("response_type", "code").
				WithFormField("state", "new").
				WithFormField("prompt", tc.prompt).
				WithFormField("code_challenge_method", tc.challengeMethod).
				WithCookie(session.CookieName, env.S(t, user.GetID())).Expect().Status(http.StatusFound)
			loc, err := res.Raw().Location()
			require.NoError(t, err)
			assert.Equal(t, []string{"new"}, loc.Query()["state"])
			if tc.consent {
				assert.Equal(t, "/consent", loc.Path)
				assert.False(t, loc.Query().Has("app"))
			} else {
				assert.Equal(t, []string{"one", "two"}, loc.Query()["app"])
				assert.Equal(t, tc.wantError, loc.Query().Get("error"))
				if tc.wantError == "" {
					assert.NotEmpty(t, loc.Query().Get("code"))
				}
			}
		})
	}

	for _, tc := range []struct {
		name, submit, wantError string
	}{
		{name: "approve", submit: "approve"},
		{name: "reject", submit: "deny", wantError: errAccessDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := env.S(t, user.GetID())
			se, err := env.SessStore.GetSessionByToken(context.Background(), token)
			require.NoError(t, err)
			require.NoError(t, se.Set(context.Background(), oauth2ContextSession, authorizeRequest{
				ClientID: client.ID, RedirectURI: client.RedirectURI, State: "new",
				Types: responseType{Code: true}, AccessTime: time.Now(),
				ValidScopes: client.Scopes,
			}))
			res := env.R(t).POST("/oauth2/authorize/decide").
				WithFormField("submit", tc.submit).
				WithCookie(session.CookieName, token).Expect().Status(http.StatusFound)
			loc, err := res.Raw().Location()
			require.NoError(t, err)
			assert.Equal(t, []string{"one", "two"}, loc.Query()["app"])
			assert.Equal(t, []string{"new"}, loc.Query()["state"])
			assert.Equal(t, tc.wantError, loc.Query().Get("error"))
			if tc.wantError == "" {
				assert.NotEmpty(t, loc.Query().Get("code"))
			}
		})
	}
}
