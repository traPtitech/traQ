package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUsersPrivateChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_private_channels", (&UsersPrivateChannel{}).TableName())
}

func TestMakePrivateChannel(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channel := &Channel{}
	channel.CreatorID = user.ID
	channel.Name = "Private-Channel"
	channel.IsPublic = false
	require.NoError(channel.Create())

	po := mustMakeUser(t, "po")
	privilegedUser := []string{user.ID, po.ID}

	for _, userID := range privilegedUser {
		usersPrivateChannel := &UsersPrivateChannel{}
		usersPrivateChannel.ChannelID = channel.ID
		usersPrivateChannel.UserID = userID
		require.NoError(usersPrivateChannel.Create())
	}

	channelList, err := GetChannels(user.ID)
	if assert.NoError(err) {
		assert.Len(channelList, 1+1)
	}

	channelList, err = GetChannels(CreateUUID())
	if assert.NoError(err) {
		assert.Len(channelList, 0+1)
	}
}
