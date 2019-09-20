package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOAuth2Authorize_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "oauth2_authorizes", (&OAuth2Authorize{}).TableName())
}

func TestOAuth2Client_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "oauth2_clients", (&OAuth2Client{}).TableName())
}

func TestOAuth2Token_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "oauth2_tokens", (&OAuth2Token{}).TableName())
}

func TestAccessScopes_Value(t *testing.T) {
	t.Parallel()

	s := AccessScopes{"read", "write"}

	v, err := s.Value()
	assert.NoError(t, err)
	assert.EqualValues(t, "read write", v)
}

func TestAccessScopes_Scan(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		s := AccessScopes{}
		assert.NoError(t, s.Scan(nil))
		assert.EqualValues(t, AccessScopes{}, s)
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		s := AccessScopes{}
		assert.NoError(t, s.Scan("a b c  "))
		assert.EqualValues(t, AccessScopes{"a", "b", "c"}, s)
	})

	t.Run("[]byte", func(t *testing.T) {
		t.Parallel()

		s := AccessScopes{}
		assert.NoError(t, s.Scan([]byte("a b c  ")))
		assert.EqualValues(t, AccessScopes{"a", "b", "c"}, s)
	})

	t.Run("other", func(t *testing.T) {
		t.Parallel()

		s := AccessScopes{}
		assert.Error(t, s.Scan(123))
	})
}

func TestAccessScopes_Contains(t *testing.T) {
	t.Parallel()

	s := AccessScopes{"read", "write"}

	assert.True(t, s.Contains("read"))
	assert.True(t, s.Contains("write"))
	assert.False(t, s.Contains("private_read"))
}

func TestAccessScopes_String(t *testing.T) {
	t.Parallel()

	s := AccessScopes{"read", "write"}

	assert.EqualValues(t, "read write", s.String())
	assert.EqualValues(t, "", AccessScopes{}.String())
}

func TestOAuth2Authorize_IsExpired(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		t.Parallel()
		data := &OAuth2Authorize{
			CreatedAt: time.Date(2000, 1, 1, 12, 0, 11, 0, time.UTC),
			ExpiresIn: 10,
		}
		assert.True(t, data.IsExpired())
	})

	t.Run("False", func(t *testing.T) {
		t.Parallel()
		data := &OAuth2Authorize{
			CreatedAt: time.Date(2099, 1, 1, 12, 0, 11, 0, time.UTC),
			ExpiresIn: 10,
		}
		assert.False(t, data.IsExpired())
	})
}

func TestOAuth2Authorize_ValidatePKCE(t *testing.T) {
	t.Parallel()

	t.Run("Case1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		data := &OAuth2Authorize{
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			CodeChallengeMethod: "plain",
		}
		if ok, err := data.ValidatePKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"); assert.NoError(err) {
			assert.True(ok)
		}
		if ok, err := data.ValidatePKCE("fewfaaafaefe-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); assert.NoError(err) {
			assert.False(ok)
		}
		if ok, err := data.ValidatePKCE("fewfaaafae"); assert.NoError(err) {
			assert.False(ok)
		}
	})

	t.Run("Case2", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		data := &OAuth2Authorize{
			CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		}
		if ok, err := data.ValidatePKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"); assert.NoError(err) {
			assert.True(ok)
		}
		if ok, err := data.ValidatePKCE("fewfaaafaefe-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); assert.NoError(err) {
			assert.False(ok)
		}
	})

	t.Run("Case3", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		data := &OAuth2Authorize{}
		if ok, err := data.ValidatePKCE("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); assert.NoError(err) {
			assert.False(ok)
		}
		if ok, err := data.ValidatePKCE(""); assert.NoError(err) {
			assert.True(ok)
		}
	})

	t.Run("Case4", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		data := &OAuth2Authorize{
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			CodeChallengeMethod: "S256",
		}
		if ok, err := data.ValidatePKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"); assert.NoError(err) {
			assert.False(ok)
		}
		if ok, err := data.ValidatePKCE("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); assert.NoError(err) {
			assert.True(ok)
		}
	})

	t.Run("Case5", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		data := &OAuth2Authorize{
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			CodeChallengeMethod: "unknown",
		}
		if ok, err := data.ValidatePKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"); assert.Error(err) {
			assert.False(ok)
		}
	})
}

func TestOAuth2Client_GetAvailableScopes(t *testing.T) {
	t.Parallel()

	client := &OAuth2Client{
		Scopes: AccessScopes{
			"read",
		},
	}
	assert.EqualValues(t, AccessScopes{"read"}, client.GetAvailableScopes(AccessScopes{"read", "write"}))
}

func TestOAuth2Token_GetAvailableScopes(t *testing.T) {
	t.Parallel()

	token := &OAuth2Token{
		Scopes: AccessScopes{
			"read",
		},
	}
	assert.EqualValues(t, AccessScopes{"read"}, token.GetAvailableScopes(AccessScopes{"read", "write"}))
}

func TestOAuth2Token_IsExpired(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		t.Parallel()
		data := &OAuth2Token{
			CreatedAt: time.Date(2000, 1, 1, 12, 0, 11, 0, time.UTC),
			ExpiresIn: 10,
		}
		assert.True(t, data.IsExpired())
	})

	t.Run("False", func(t *testing.T) {
		t.Parallel()
		data := &OAuth2Token{
			CreatedAt: time.Date(2099, 1, 1, 12, 0, 11, 0, time.UTC),
			ExpiresIn: 10,
		}
		assert.False(t, data.IsExpired())
	})
}

func TestOAuth2Token_IsRefreshEnabled(t *testing.T) {
	t.Parallel()

	assert.False(t, (&OAuth2Token{RefreshToken: "test"}).IsRefreshEnabled())
	assert.True(t, (&OAuth2Token{RefreshToken: "test", RefreshEnabled: true}).IsRefreshEnabled())
}
