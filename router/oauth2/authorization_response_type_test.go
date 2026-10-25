package oauth2

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/utils/random"
)

func TestAuthorizationResponseTypeBeforeIssuance(t *testing.T) {
	t.Parallel()
	env := Setup(t, db2)
	user := env.CreateUser(t, rand, false)
	client := &model.OAuth2Client{
		ID: random.AlphaNumeric(36), Name: "response type client", CreatorID: user.GetID(),
		RedirectURI: "https://example.com/callback", Scopes: model.AccessScopes{"read": {}},
	}
	require.NoError(t, env.Repository.SaveClient(context.Background(), client))
	env.IssueToken(t, client, user.GetID(), false)
	token := env.S(t, user.GetID())
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, responseType := range []string{"", "token", "id_token", "none", "code token", "unknown"} {
			t.Run(method+"/"+responseType, func(t *testing.T) {
				req := env.R(t).Request(method, "/oauth2/authorize").WithCookie(session.CookieName, token)
				params := map[string]string{"client_id": client.ID, "response_type": responseType, "prompt": "none"}
				for key, value := range params {
					if method == http.MethodGet {
						req.WithQuery(key, value)
					} else {
						req.WithFormField(key, value)
					}
				}
				loc, err := req.Expect().Status(http.StatusFound).Raw().Location()
				require.NoError(t, err)
				wantError := errUnsupportedResponseType
				if responseType == "" {
					wantError = errInvalidRequest
				}
				assert.Equal(t, wantError, loc.Query().Get("error"))
				assert.Empty(t, loc.Query().Get("code"))
				var count int64
				require.NoError(t, env.DB.Model(&model.OAuth2Authorize{}).Where("client_id = ?", client.ID).Count(&count).Error)
				assert.Zero(t, count)
			})
		}
	}
}
