package role

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"testing"
)

func TestSetRole(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	r, err := rbac.New(nil)
	require.NoError(err)

	SetRole(r)

	// Adminは全権限を持つ
	for _, v := range permission.GetAllPermissionList() {
		assert.True(r.IsGranted(uuid.Nil, Admin.ID(), v), v.ID())
	}
}
