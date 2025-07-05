package oauth2

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

func TestHandlers_RevokeTokenEndpointHandler(t *testing.T) {
	t.Parallel()
	t.Run("UUIDv4", func(tt *testing.T) {
		tt.Parallel()
		runRevokeTokenEndpointTests(tt, true)
	})

	t.Run("UUIDv7", func(tt *testing.T) {
		tt.Parallel()
		runRevokeTokenEndpointTests(tt, false)
	})
}

func runRevokeTokenEndpointTests(t *testing.T, useUUIDv4 bool) {
	env := Setup(t, db1)
	user := env.CreateUser(t, rand, useUUIDv4)

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
