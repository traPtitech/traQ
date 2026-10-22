package v3

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_OpenAPISecurity(t *testing.T) {
	t.Parallel()
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	client := env.CreateOAuth2Client(t, rand, user.GetID())
	channel := env.CreateChannel(t, rand)
	bot := env.CreateBot(t, rand, user.GetID())
	for _, tc := range []struct {
		method string
		path   string
		url    string
		status int
	}{
		{http.MethodGet, "/users/me", "/users/me", http.StatusOK},
		{http.MethodGet, "/users/me/oidc", "/users/me/oidc", http.StatusOK},
		{http.MethodPost, "/channels/{channelId}/messages", "/channels/" + channel.ID.String() + "/messages", http.StatusCreated},
		{http.MethodGet, "/bots/{botId}/logs", "/bots/" + bot.ID.String() + "/logs", http.StatusOK},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			operation := loadOpenAPIOperation(t, tc.path, strings.ToLower(tc.method))
			require.NotEmpty(t, operation.Security)
			for _, credentials := range []string{"cookie", "", "read", "write", "manage_bot", "openid", "profile", "qall", "read write manage_bot openid profile qall"} {
				t.Run(credentials, func(t *testing.T) {
					scopes := model.AccessScopes{}
					for _, scope := range strings.Fields(credentials) {
						scopes.Add(model.AccessScope(scope))
					}
					scheme := "OAuth2"
					if credentials == "cookie" {
						scheme = "cookieAuth"
					}
					allowed := false
					for _, alternative := range operation.Security {
						required, exists := alternative[scheme]
						if !exists || len(alternative) != 1 {
							continue
						}
						matches := true
						for _, scope := range required {
							matches = matches && scopes.Contains(model.AccessScope(scope))
						}
						allowed = allowed || matches
					}
					status := http.StatusForbidden
					if allowed {
						status = tc.status
					}
					req := env.R(t).Request(tc.method, "/api/v3"+tc.url)
					if tc.method == http.MethodPost {
						req.WithJSON(map[string]string{"content": "scope contract"})
					}
					if scheme == "cookieAuth" {
						req.WithCookie(session.CookieName, env.S(t, user.GetID()))
					} else {
						token, err := env.Repository.IssueToken(context.Background(), client, user.GetID(), client.RedirectURI, scopes, 86400, false)
						require.NoError(t, err)
						req.WithHeader("Authorization", "Bearer "+token.AccessToken)
					}
					req.Expect().Status(status)
				})
			}
		})
	}
}
