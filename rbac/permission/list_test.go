package permission

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPermission(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	assert.Nil(GetPermission("存在しない"))
	assert.Equal(GetChannel, GetPermission(GetChannel.ID()))
	assert.Equal(CreateChannel, GetPermission(CreateChannel.ID()))
}

func TestGetAllPermissionList(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	l := GetAllPermissionList()
	for k, v := range l {
		assert.Equal(list[k], v, k)
	}
}
