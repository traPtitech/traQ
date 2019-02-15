package rbac

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/rbac/permission"
	"testing"
)

func TestRBACOverride_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "rbac_overrides", (&Override{}).TableName())
}

func TestRBACOverride_GetUserID(t *testing.T) {
	t.Parallel()

	id := uuid.NewV4()
	r := &Override{
		UserID: id,
	}
	assert.Equal(t, id, r.GetUserID())
}

func TestRBACOverride_GetPermission(t *testing.T) {
	t.Parallel()

	r := &Override{
		Permission: permission.GetChannel.ID(),
	}
	assert.Equal(t, permission.GetChannel, r.GetPermission())
}

func TestRBACOverride_GetValidity(t *testing.T) {
	t.Parallel()

	r := &Override{
		Validity: true,
	}
	assert.True(t, r.GetValidity())
}
