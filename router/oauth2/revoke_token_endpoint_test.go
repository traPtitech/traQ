package oauth2

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"net/http"
	"testing"
)

func TestHandlers_RevokeTokenEndpointHandler(t *testing.T) {
	t.Parallel()
	env := Setup(t, db1)
	user := env.CreateUser(t, rand)

	t.Run("NoToken", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST("/oauth2/revoke").
			WithFormField("token", "").
			Expect().
			Status(http.StatusOK)
	})

	t.Run("AccessToken", func(t *testing.T) {
		t.Parallel()
		token, err := env.Repository.IssueToken(nil, user.GetID(), "", model.AccessScopes{}, 10000, false)
		require.NoError(t, err)

		e := env.R(t)
		e.POST("/oauth2/revoke").
			WithFormField("token", token.AccessToken).
			Expect().
			Status(http.StatusOK)

		_, err = env.Repository.GetTokenByID(token.ID)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("RefreshToken", func(t *testing.T) {
		t.Parallel()
		token, err := env.Repository.IssueToken(nil, user.GetID(), "", model.AccessScopes{}, 10000, true)
		require.NoError(t, err)

		e := env.R(t)
		e.POST("/oauth2/revoke").
			WithFormField("token", token.RefreshToken).
			Expect().
			Status(http.StatusOK)

		_, err = env.Repository.GetTokenByID(token.ID)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})
}
