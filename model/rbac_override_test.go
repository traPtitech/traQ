package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/rbac/permission"
	"testing"
)

func TestRBACOverride_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "rbac_overrides", (&RBACOverride{}).TableName())
}

func TestRBACOverride_GetUserID(t *testing.T) {
	t.Parallel()

	id := uuid.NewV4()
	r := &RBACOverride{
		UserID: id.String(),
	}
	assert.Equal(t, id, r.GetUserID())
}

func TestRBACOverride_GetPermission(t *testing.T) {
	t.Parallel()

	r := &RBACOverride{
		Permission: permission.GetChannel.ID(),
	}
	assert.Equal(t, permission.GetChannel, r.GetPermission())
}

func TestRBACOverride_GetValidity(t *testing.T) {
	t.Parallel()

	r := &RBACOverride{
		Validity: true,
	}
	assert.True(t, r.GetValidity())
}

func TestRBACOverrideStore(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	s := &RBACOverrideStore{}
	user := mustMakeUser(t, "testRBACOverrideStore")
	userID := user.ID

	if assert.NoError(s.SaveOverride(userID, permission.GetChannel, true)) {
		arr, err := s.GetAllOverrides()
		if assert.NoError(err) && assert.Len(arr, 1) {
			assert.Equal(userID, arr[0].GetUserID())
			assert.Equal(permission.GetChannel, arr[0].GetPermission())
			assert.True(arr[0].GetValidity())
		}
	}

	if assert.NoError(s.SaveOverride(userID, permission.GetMessage, true)) {
		arr, err := s.GetAllOverrides()
		if assert.NoError(err) {
			assert.Len(arr, 2)
		}
	}

	if assert.NoError(s.DeleteOverride(userID, permission.GetMessage)) {
		arr, err := s.GetAllOverrides()
		if assert.NoError(err) && assert.Len(arr, 1) {
			assert.Equal(userID, arr[0].GetUserID())
			assert.Equal(permission.GetChannel, arr[0].GetPermission())
			assert.True(arr[0].GetValidity())
		}
	}

	if assert.NoError(s.SaveOverride(userID, permission.GetChannel, false)) {
		arr, err := s.GetAllOverrides()
		if assert.NoError(err) && assert.Len(arr, 1) {
			assert.Equal(userID, arr[0].GetUserID())
			assert.Equal(permission.GetChannel, arr[0].GetPermission())
			assert.False(arr[0].GetValidity())
		}
	}

}
