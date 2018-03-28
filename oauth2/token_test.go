package oauth2

import (
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/oauth2/scope"
	"testing"
	"time"
)

func TestToken_GetAvailableScopes(t *testing.T) {
	t.Parallel()

	token := &Token{
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.Write,
		},
	}
	assert.EqualValues(t, scope.AccessScopes{scope.Read}, token.GetAvailableScopes(scope.AccessScopes{scope.Read, scope.PrivateRead}))
}

func TestToken_IsExpired(t *testing.T) {
	t.Parallel()

	{
		token := &Token{
			CreatedAt: time.Date(2000, 1, 1, 12, 0, 11, 0, time.UTC),
			ExpiresIn: 10,
		}
		assert.True(t, token.IsExpired())
	}

	{
		token := &Token{
			CreatedAt: time.Date(2099, 1, 1, 12, 0, 11, 0, time.UTC),
			ExpiresIn: 10,
		}
		assert.False(t, token.IsExpired())
	}
}
