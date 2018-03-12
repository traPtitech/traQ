package oauth2

import (
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/auth/scope"
	"testing"
)

func TestClient_GetAvailableScopes(t *testing.T) {
	t.Parallel()

	client := &Client{
		Scope: scope.AccessScopes{
			scope.Read,
			scope.Write,
		},
	}
	assert.EqualValues(t, scope.AccessScopes{scope.Read}, client.GetAvailableScopes(scope.AccessScopes{scope.Read, scope.PrivateRead}))
}
