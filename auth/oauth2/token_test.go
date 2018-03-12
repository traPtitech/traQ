package oauth2

import (
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/auth/scope"
	"testing"
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
