package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "channels", (&Channel{}).TableName())
}

func TestUsersPrivateChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_private_channels", (&UsersPrivateChannel{}).TableName())
}

func TestUserSubscribeChannel_TableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "users_subscribe_channels", (&UserSubscribeChannel{}).TableName())
}
