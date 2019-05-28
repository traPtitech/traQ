package model

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "channels", (&Channel{}).TableName())
}

func TestChannel_IsDMChannel(t *testing.T) {
	t.Parallel()
	assert.False(t, (&Channel{ParentID: uuid.Nil}).IsDMChannel())
	assert.True(t, (&Channel{ParentID: dmChannelRootUUID}).IsDMChannel())
}

func TestUsersPrivateChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_private_channels", (&UsersPrivateChannel{}).TableName())
}

func TestUserSubscribeChannel_TableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "users_subscribe_channels", (&UserSubscribeChannel{}).TableName())
}

func TestDMChannelMapping_TableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dm_channel_mappings", (&DMChannelMapping{}).TableName())
}
