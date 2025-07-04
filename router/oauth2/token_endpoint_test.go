package oauth2

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	random2 "github.com/traPtitech/traQ/utils/random"
)

func TestHandlers_TokenEndpointHandler(t *testing.T) {
	t.Parallel()
	env := Setup(t, db2)

	t.Run("Unsupported Grant Type", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", "ああああ").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errUnsupportedGrantType)
	})
}

func TestHandlers_TokenEndpointClientCredentialsHandler(t *testing.T) {
	t.Run("UUIDv4", func(t *testing.T) {
		t.Parallel()
		runTokenEndpointClientCredentialsTests(t, 4)
	})

	t.Run("UUIDv7", func(t *testing.T) {
		t.Parallel()
		runTokenEndpointClientCredentialsTests(t, 7)
	})
}

func runTokenEndpointClientCredentialsTests(t *testing.T, uuidVersion int) {
	env := Setup(t, db2)

	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")

	var creatorID uuid.UUID
	if uuidVersion == 4 {
		creatorID = uuid.Must(uuid.NewV4())
	} else {
		creatorID = uuid.Must(uuid.NewV7())
	}

	client := &model.OAuth2Client{
		ID:           random2.AlphaNumeric(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    creatorID,
		Secret:       random2.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, env.Repository.SaveClient(client))

	t.Run("Success with Basic Auth", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.NotContainsKey("refresh_token")
		actual := model.AccessScopes{}
		actual.FromString(obj.Value("scope").String().Raw())
		assert.ElementsMatch(t, client.Scopes.StringArray(), actual.StringArray())
	})

	t.Run("Success with form Auth", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("client_id", client.ID).
			WithFormField("client_secret", client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.NotContainsKey("refresh_token")
		actual := model.AccessScopes{}
		actual.FromString(obj.Value("scope").String().Raw())
		assert.ElementsMatch(t, client.Scopes.StringArray(), actual.StringArray())
	})

	t.Run("Success with smaller scope", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("scope", "read").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.NotContainsKey("refresh_token")
		obj.NotContainsKey("scope")
	})

	t.Run("Success with invalid scope", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("scope", "read manage_bot").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.NotContainsKey("refresh_token")
		obj.Value("scope").String().IsEqual("read")
	})

	t.Run("Invalid Client (No credentials)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidClient)
	})

	t.Run("Invalid Client (Wrong credentials)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithBasicAuth(client.ID, "wrong password").
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidClient)
	})

	t.Run("Invalid Client (Unknown client)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithBasicAuth("wrong client", "wrong password").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidClient)
	})

	t.Run("Invalid Client (Not confidential)", func(t *testing.T) {
		t.Parallel()

		var creatorID uuid.UUID
		if uuidVersion == 4 {
			creatorID = uuid.Must(uuid.NewV4())
		} else {
			creatorID = uuid.Must(uuid.NewV7())
		}

		client := &model.OAuth2Client{
			ID:           random2.AlphaNumeric(36),
			Name:         "test client",
			Confidential: false,
			CreatorID:    creatorID,
			Secret:       random2.AlphaNumeric(36),
			RedirectURI:  "http://example.com",
			Scopes:       scopesReadWrite,
		}
		require.NoError(t, env.Repository.SaveClient(client))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errUnauthorizedClient)
	})

	t.Run("Invalid Scope (unknown scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("scope", "アイウエオ").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidScope)
	})

	t.Run("Invalid Scope (no valid scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("scope", "manage_bot").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidScope)
	})
}

func TestHandlers_TokenEndpointPasswordHandler(t *testing.T) {
	t.Run("UUIDv4", func(t *testing.T) {
		t.Parallel()
		runTokenEndpointPasswordTests(t, 4)
	})

	t.Run("UUIDv7", func(t *testing.T) {
		t.Parallel()
		runTokenEndpointPasswordTests(t, 7)
	})
}

func runTokenEndpointPasswordTests(t *testing.T, uuidVersion int) {
	env := Setup(t, db2)
	user := env.CreateUser(t, rand, uuidVersion)

	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")

	var creatorID uuid.UUID
	if uuidVersion == 4 {
		creatorID = uuid.Must(uuid.NewV4())
	} else {
		creatorID = uuid.Must(uuid.NewV7())
	}

	client := &model.OAuth2Client{
		ID:           random2.AlphaNumeric(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    creatorID,
		Secret:       random2.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, env.Repository.SaveClient(client))

	t.Run("Success with Basic Auth", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "!test_test@test-").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		actual := model.AccessScopes{}
		actual.FromString(obj.Value("scope").String().Raw())
		assert.ElementsMatch(t, client.Scopes.StringArray(), actual.StringArray())
	})

	t.Run("Success with form Auth", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "!test_test@test-").
			WithFormField("client_id", client.ID).
			WithFormField("client_secret", client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		actual := model.AccessScopes{}
		actual.FromString(obj.Value("scope").String().Raw())
		assert.ElementsMatch(t, client.Scopes.StringArray(), actual.StringArray())
	})

	t.Run("Success with smaller scope", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "!test_test@test-").
			WithFormField("scope", "read").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")
	})

	t.Run("Success with invalid scope", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "!test_test@test-").
			WithFormField("scope", "read manage_bot").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().IsEqual("read")
	})

	t.Run("Success with not confidential client", func(t *testing.T) {
		t.Parallel()

		var creatorID uuid.UUID
		if uuidVersion == 4 {
			creatorID = uuid.Must(uuid.NewV4())
		} else {
			creatorID = uuid.Must(uuid.NewV7())
		}

		client := &model.OAuth2Client{
			ID:           random2.AlphaNumeric(36),
			Name:         "test client",
			Confidential: false,
			CreatorID:    creatorID,
			Secret:       random2.AlphaNumeric(36),
			RedirectURI:  "http://example.com",
			Scopes:       scopesReadWrite,
		}
		require.NoError(t, env.Repository.SaveClient(client))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "!test_test@test-").
			WithFormField("client_id", client.ID).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		actual := model.AccessScopes{}
		actual.FromString(obj.Value("scope").String().Raw())
		assert.ElementsMatch(t, client.Scopes.StringArray(), actual.StringArray())
	})

	t.Run("Invalid Request (No user credentials)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidRequest)
	})

	t.Run("Invalid Grant (Wrong user credentials)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "wrong password").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidGrant)
	})

	t.Run("Invalid Client (No client credentials)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "test").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidClient)
	})

	t.Run("Invalid Client (Wrong client credentials)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "test").
			WithBasicAuth(client.ID, "wrong password").
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidClient)
	})

	t.Run("Invalid Client (Unknown client)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "test").
			WithBasicAuth("wrong client", "wrong password").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidClient)
	})

	t.Run("Invalid Scope (unknown scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "!test_test@test-").
			WithFormField("scope", "ああああ").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidScope)
	})

	t.Run("Invalid Scope (no valid scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.GetName()).
			WithFormField("password", "!test_test@test-").
			WithFormField("scope", "manage_bot").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidScope)
	})
}

func TestHandlers_TokenEndpointRefreshTokenHandler(t *testing.T) {
	t.Run("UUIDv4", func(t *testing.T) {
		t.Parallel()
		runTokenEndpointRefreshTokenTests(t, 4)
	})

	t.Run("UUIDv7", func(t *testing.T) {
		t.Parallel()
		runTokenEndpointRefreshTokenTests(t, 7)
	})
}

func runTokenEndpointRefreshTokenTests(t *testing.T, uuidVersion int) {
	env := Setup(t, db2)
	user := env.CreateUser(t, rand, uuidVersion)

	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")

	var creatorID uuid.UUID
	if uuidVersion == 4 {
		creatorID = uuid.Must(uuid.NewV4())
	} else {
		creatorID = uuid.Must(uuid.NewV7())
	}

	client := &model.OAuth2Client{
		ID:           random2.AlphaNumeric(36),
		Name:         "test client",
		Confidential: false,
		CreatorID:    creatorID,
		Secret:       random2.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, env.Repository.SaveClient(client))

	var creatorIDConf uuid.UUID
	if uuidVersion == 4 {
		creatorIDConf = uuid.Must(uuid.NewV4())
	} else {
		creatorIDConf = uuid.Must(uuid.NewV7())
	}

	clientConf := &model.OAuth2Client{
		ID:           random2.AlphaNumeric(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    creatorIDConf,
		Secret:       random2.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, env.Repository.SaveClient(clientConf))

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, client, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with smaller scope", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, client, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("scope", "read").
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().IsEqual("read")

		_, err := env.Repository.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with invalid scope", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, client, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("scope", "read manage_bot").
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().IsEqual("read")

		_, err := env.Repository.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with confidential client Basic Auth", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, clientConf, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with confidential client form Auth", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, clientConf, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("client_id", clientConf.ID).
			WithFormField("client_secret", clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Request (No refresh token)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidRequest)
	})

	t.Run("Invalid Grant (Unknown refresh token)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", "unknown token").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidGrant)
	})

	t.Run("Invalid Client (No client credentials)", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, clientConf, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidClient)
	})

	t.Run("Invalid Client (Wrong client credentials)", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, clientConf, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithBasicAuth(clientConf.ID, "wrong password").
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidClient)
	})

	t.Run("Invalid Scope (unknown scope)", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, client, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("scope", "アイウエオ").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidScope)
	})

	t.Run("Invalid Scope (no valid scope)", func(t *testing.T) {
		t.Parallel()
		token := env.IssueToken(t, client, user.GetID(), true)
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("scope", "manage_bot").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").String().IsEqual(errInvalidScope)
	})
}

func TestHandlers_TokenEndpointAuthorizationCodeHandler(t *testing.T) {
	t.Run("UUIDv4", func(t *testing.T) {
		t.Parallel()
		runTokenEndpointAuthorizationCodeTests(t, 4)
	})

	t.Run("UUIDv7", func(t *testing.T) {
		t.Parallel()
		runTokenEndpointAuthorizationCodeTests(t, 7)
	})
}

func runTokenEndpointAuthorizationCodeTests(t *testing.T, uuidVersion int) {
	env := Setup(t, db2)
	user := env.CreateUser(t, rand, uuidVersion)

	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")
	scopesRead := model.AccessScopes{}
	scopesRead.Add("read")
	scopesReadManageBot := model.AccessScopes{}
	scopesReadManageBot.Add("read", "manage_bot")

	var creatorID uuid.UUID
	if uuidVersion == 4 {
		creatorID = uuid.Must(uuid.NewV4())
	} else {
		creatorID = uuid.Must(uuid.NewV7())
	}

	client := &model.OAuth2Client{
		ID:           random2.AlphaNumeric(36),
		Name:         "test client",
		Confidential: false,
		CreatorID:    creatorID,
		Secret:       random2.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, env.Repository.SaveClient(client))

	var creatorIDConf uuid.UUID
	if uuidVersion == 4 {
		creatorIDConf = uuid.Must(uuid.NewV4())
	} else {
		creatorIDConf = uuid.Must(uuid.NewV7())
	}

	clientConf := &model.OAuth2Client{
		ID:           random2.AlphaNumeric(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    creatorIDConf,
		Secret:       random2.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, env.Repository.SaveClient(clientConf))

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		authorize := env.MakeAuthorizeData(t, client.ID, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("client_id", client.ID).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with confidential client Basic Auth", func(t *testing.T) {
		t.Parallel()
		authorize := env.MakeAuthorizeData(t, clientConf.ID, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with confidential client form Auth", func(t *testing.T) {
		t.Parallel()
		authorize := env.MakeAuthorizeData(t, clientConf.ID, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("client_id", clientConf.ID).
			WithFormField("client_secret", clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with PKCE(plain)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:                random2.AlphaNumeric(36),
			ClientID:            clientConf.ID,
			UserID:              user.GetID(),
			CreatedAt:           time.Now(),
			ExpiresIn:           1000,
			RedirectURI:         "http://example.com",
			Scopes:              scopesReadWrite,
			OriginalScopes:      scopesReadWrite,
			Nonce:               "nonce",
			CodeChallengeMethod: "plain",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		}
		require.NoError(t, env.Repository.SaveAuthorize(authorize))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("code_verifier", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with PKCE(S256)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:                random2.AlphaNumeric(36),
			ClientID:            clientConf.ID,
			UserID:              user.GetID(),
			CreatedAt:           time.Now(),
			ExpiresIn:           1000,
			RedirectURI:         "http://example.com",
			Scopes:              scopesReadWrite,
			OriginalScopes:      scopesReadWrite,
			Nonce:               "nonce",
			CodeChallengeMethod: "S256",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		}
		require.NoError(t, env.Repository.SaveAuthorize(authorize))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("code_verifier", "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with smaller scope", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           random2.AlphaNumeric(36),
			ClientID:       clientConf.ID,
			UserID:         user.GetID(),
			CreatedAt:      time.Now(),
			ExpiresIn:      1000,
			RedirectURI:    "http://example.com",
			Scopes:         scopesRead,
			OriginalScopes: scopesRead,
			Nonce:          "nonce",
		}
		require.NoError(t, env.Repository.SaveAuthorize(authorize))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with invalid scope", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           random2.AlphaNumeric(36),
			ClientID:       client.ID,
			UserID:         user.GetID(),
			CreatedAt:      time.Now(),
			ExpiresIn:      1000,
			RedirectURI:    "http://example.com",
			Scopes:         scopesRead,
			OriginalScopes: scopesReadManageBot,
			Nonce:          "nonce",
		}
		require.NoError(t, env.Repository.SaveAuthorize(authorize))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().IsEqual(authScheme)
		obj.Value("expires_in").Number().IsEqual(1000)
		obj.Value("refresh_token").String().NotEmpty()
		actual := model.AccessScopes{}
		actual.FromString(obj.Value("scope").String().Raw())
		assert.ElementsMatch(t, authorize.Scopes.StringArray(), actual.StringArray())

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Request (No code)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidRequest)
	})

	t.Run("Invalid Client (No client)", func(t *testing.T) {
		t.Parallel()
		authorize := env.MakeAuthorizeData(t, clientConf.ID, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidClient)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Client (Wrong client credentials)", func(t *testing.T) {
		t.Parallel()
		authorize := env.MakeAuthorizeData(t, clientConf.ID, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, "wrong password").
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidClient)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Client (Other client)", func(t *testing.T) {
		t.Parallel()
		authorize := env.MakeAuthorizeData(t, clientConf.ID, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidClient)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Grant (Wrong code)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", "unknown").
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidGrant)
	})

	t.Run("Invalid Grant (expired)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           random2.AlphaNumeric(36),
			ClientID:       clientConf.ID,
			UserID:         user.GetID(),
			CreatedAt:      time.Now(),
			ExpiresIn:      -1000,
			RedirectURI:    "http://example.com",
			Scopes:         scopesReadWrite,
			OriginalScopes: scopesReadWrite,
			Nonce:          "nonce",
		}
		require.NoError(t, env.Repository.SaveAuthorize(authorize))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidGrant)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Client (client not found)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           random2.AlphaNumeric(36),
			ClientID:       random2.AlphaNumeric(36),
			UserID:         user.GetID(),
			CreatedAt:      time.Now(),
			ExpiresIn:      1000,
			RedirectURI:    "http://example.com",
			Scopes:         scopesReadWrite,
			OriginalScopes: scopesReadWrite,
			Nonce:          "nonce",
		}
		require.NoError(t, env.Repository.SaveAuthorize(authorize))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidClient)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Grant (different redirect)", func(t *testing.T) {
		t.Parallel()
		authorize := env.MakeAuthorizeData(t, clientConf.ID, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example2.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidGrant)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Grant (unexpected redirect)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           random2.AlphaNumeric(36),
			ClientID:       clientConf.ID,
			UserID:         user.GetID(),
			CreatedAt:      time.Now(),
			ExpiresIn:      1000,
			Scopes:         scopesReadWrite,
			OriginalScopes: scopesReadWrite,
			Nonce:          "nonce",
		}
		require.NoError(t, env.Repository.SaveAuthorize(authorize))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidGrant)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Request (PKCE failure)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:                random2.AlphaNumeric(36),
			ClientID:            clientConf.ID,
			UserID:              user.GetID(),
			CreatedAt:           time.Now(),
			ExpiresIn:           1000,
			RedirectURI:         "http://example.com",
			Scopes:              scopesReadWrite,
			OriginalScopes:      scopesReadWrite,
			Nonce:               "nonce",
			CodeChallengeMethod: "plain",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		}
		require.NoError(t, env.Repository.SaveAuthorize(authorize))
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidRequest)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Request (unexpected PKCE)", func(t *testing.T) {
		t.Parallel()
		authorize := env.MakeAuthorizeData(t, clientConf.ID, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("code_verifier", "jfeiajoijioajfoiwjo").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		res.JSON().Object().Value("error").IsEqual(errInvalidRequest)

		_, err := env.Repository.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})
}
