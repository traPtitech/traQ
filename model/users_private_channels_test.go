package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUsersPrivateChannel_TableName(t *testing.T) {
	assert.Equal(t, "users_private_channels", (&UsersPrivateChannel{}).TableName())
}

func TestMakePrivateChannel(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	channel := &Channel{}
	channel.CreatorID = testUserID
	channel.Name = "Private-Channel"
	channel.IsPublic = false
	require.NoError(t, channel.Create())

	po := CreateUUID()
	privilegedUser := []string{testUserID, po}

	for _, userID := range privilegedUser {
		usersPrivateChannel := &UsersPrivateChannel{}
		usersPrivateChannel.ChannelID = channel.ID
		usersPrivateChannel.UserID = userID
		require.NoError(t, usersPrivateChannel.Create())
	}

	channelList, err := GetChannels(testUserID)
	if assert.NoError(err) {
		assert.Len(channelList, 1)
	}

	channelList, err = GetChannels(CreateUUID())
	if assert.NoError(err) {
		assert.Len(channelList, 0)
	}
}
